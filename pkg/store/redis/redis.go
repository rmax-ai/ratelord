package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rmax-ai/ratelord/pkg/engine"
	"github.com/rmax-ai/ratelord/pkg/engine/currency"
	"github.com/rmax-ai/ratelord/pkg/engine/forecast"
)

const poolsSet = "ratelord:pools"

type RedisUsageStore struct {
	client *redis.Client
}

func NewRedisUsageStore(client *redis.Client) *RedisUsageStore {
	return &RedisUsageStore{client: client}
}

func (s *RedisUsageStore) makeKey(providerID, poolID string) string {
	return fmt.Sprintf("ratelord:pool:%s:%s", providerID, poolID)
}

func (s *RedisUsageStore) Set(state engine.PoolState) {
	key := s.makeKey(state.ProviderID, state.PoolID)
	ctx := context.Background()

	fields := map[string]interface{}{
		"used":         strconv.FormatInt(state.Used, 10),
		"remaining":    strconv.FormatInt(state.Remaining, 10),
		"cost":         strconv.FormatInt(int64(state.Cost), 10),
		"reset_at":     state.ResetAt.Format(time.RFC3339),
		"last_updated": state.LastUpdated.Format(time.RFC3339),
	}

	if state.LatestForecast != nil {
		forecastData, err := json.Marshal(state.LatestForecast)
		if err != nil {
			log.Printf("Failed to marshal LatestForecast: %v", err)
			return
		}
		fields["latest_forecast"] = string(forecastData)
	}

	if err := s.client.HSet(ctx, key, fields).Err(); err != nil {
		log.Printf("Failed to HSET key %s: %v", key, err)
		return
	}
	if err := s.client.SAdd(ctx, poolsSet, key).Err(); err != nil {
		log.Printf("Failed to SADD key %s to set: %v", key, err)
	}
}

func (s *RedisUsageStore) Get(providerID, poolID string) (engine.PoolState, bool) {
	key := s.makeKey(providerID, poolID)
	ctx := context.Background()
	fields, err := s.client.HGetAll(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return engine.PoolState{}, false
		}
		log.Printf("Failed to HGETALL key %s: %v", key, err)
		return engine.PoolState{}, false
	}
	if len(fields) == 0 {
		return engine.PoolState{}, false
	}

	state := engine.PoolState{
		ProviderID: providerID,
		PoolID:     poolID,
	}

	if usedStr, ok := fields["used"]; ok {
		if used, err := strconv.ParseInt(usedStr, 10, 64); err == nil {
			state.Used = used
		}
	}
	if remainingStr, ok := fields["remaining"]; ok {
		if remaining, err := strconv.ParseInt(remainingStr, 10, 64); err == nil {
			state.Remaining = remaining
		}
	}
	if costStr, ok := fields["cost"]; ok {
		if cost, err := strconv.ParseInt(costStr, 10, 64); err == nil {
			state.Cost = currency.MicroUSD(cost)
		}
	}
	if resetAtStr, ok := fields["reset_at"]; ok {
		if resetAt, err := time.Parse(time.RFC3339, resetAtStr); err == nil {
			state.ResetAt = resetAt
		}
	}
	if lastUpdatedStr, ok := fields["last_updated"]; ok {
		if lastUpdated, err := time.Parse(time.RFC3339, lastUpdatedStr); err == nil {
			state.LastUpdated = lastUpdated
		}
	}
	if forecastStr, ok := fields["latest_forecast"]; ok && forecastStr != "" {
		var forecast forecast.Forecast
		if err := json.Unmarshal([]byte(forecastStr), &forecast); err == nil {
			state.LatestForecast = &forecast
		}
	}

	return state, true
}

func (s *RedisUsageStore) GetAll() []engine.PoolState {
	ctx := context.Background()
	keys, err := s.client.SMembers(ctx, poolsSet).Result()
	if err != nil {
		log.Printf("Failed to SMEMBERS %s: %v", poolsSet, err)
		return nil
	}
	if len(keys) == 0 {
		return []engine.PoolState{}
	}

	pipe := s.client.Pipeline()
	cmds := make(map[string]*redis.MapStringStringCmd)
	for _, key := range keys {
		cmds[key] = pipe.HGetAll(ctx, key)
	}
	_, err = pipe.Exec(ctx)
	if err != nil {
		log.Printf("Failed to execute pipeline: %v", err)
		return nil
	}

	var states []engine.PoolState
	for key, cmd := range cmds {
		fields, err := cmd.Result()
		if err != nil {
			log.Printf("Failed to HGETALL key %s: %v", key, err)
			continue
		}
		if len(fields) == 0 {
			continue
		}

		// Parse providerID and poolID from key
		var providerID, poolID string
		if _, err := fmt.Sscanf(key, "ratelord:pool:%s:%s", &providerID, &poolID); err != nil {
			log.Printf("Failed to parse key %s: %v", key, err)
			continue
		}

		state := engine.PoolState{
			ProviderID: providerID,
			PoolID:     poolID,
		}

		if usedStr, ok := fields["used"]; ok {
			if used, err := strconv.ParseInt(usedStr, 10, 64); err == nil {
				state.Used = used
			}
		}
		if remainingStr, ok := fields["remaining"]; ok {
			if remaining, err := strconv.ParseInt(remainingStr, 10, 64); err == nil {
				state.Remaining = remaining
			}
		}
		if costStr, ok := fields["cost"]; ok {
			if cost, err := strconv.ParseInt(costStr, 10, 64); err == nil {
				state.Cost = currency.MicroUSD(cost)
			}
		}
		if resetAtStr, ok := fields["reset_at"]; ok {
			if resetAt, err := time.Parse(time.RFC3339, resetAtStr); err == nil {
				state.ResetAt = resetAt
			}
		}
		if lastUpdatedStr, ok := fields["last_updated"]; ok {
			if lastUpdated, err := time.Parse(time.RFC3339, lastUpdatedStr); err == nil {
				state.LastUpdated = lastUpdated
			}
		}
		if forecastStr, ok := fields["latest_forecast"]; ok && forecastStr != "" {
			var f forecast.Forecast
			if err := json.Unmarshal([]byte(forecastStr), &f); err == nil {
				state.LatestForecast = &f
			}
		}

		states = append(states, state)
	}
	return states
}

func (s *RedisUsageStore) Clear() {
	ctx := context.Background()
	keys, err := s.client.SMembers(ctx, poolsSet).Result()
	if err != nil {
		log.Printf("Failed to SMEMBERS %s during clear: %v", poolsSet, err)
		return
	}
	if len(keys) > 0 {
		if err := s.client.Del(ctx, keys...).Err(); err != nil {
			log.Printf("Failed to DEL keys: %v", err)
		}
	}
	if err := s.client.Del(ctx, poolsSet).Err(); err != nil {
		log.Printf("Failed to DEL set %s: %v", poolsSet, err)
	}
}

func (s *RedisUsageStore) Increment(providerID, poolID string, usedDelta, remainingDelta int64, costDelta currency.MicroUSD) {
	key := s.makeKey(providerID, poolID)
	ctx := context.Background()
	currentTime := time.Now().Format(time.RFC3339)

	script := `
		local key = KEYS[1]
		local usedDelta = tonumber(ARGV[1])
		local remainingDelta = tonumber(ARGV[2])
		local costDelta = tonumber(ARGV[3])
		local currentTime = ARGV[4]
		redis.call('HINCRBY', key, 'used', usedDelta)
		redis.call('HINCRBY', key, 'remaining', remainingDelta)
		redis.call('HINCRBY', key, 'cost', costDelta)
		redis.call('HSET', key, 'last_updated', currentTime)
	`

	if err := s.client.Eval(ctx, script, []string{key}, usedDelta, remainingDelta, int64(costDelta), currentTime).Err(); err != nil {
		log.Printf("Failed to increment key %s: %v", key, err)
	}
}
