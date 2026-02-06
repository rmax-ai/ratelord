package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/redis/go-redis/v9"
	"github.com/rmax-ai/ratelord/pkg/engine"
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
	data, err := json.Marshal(state)
	if err != nil {
		log.Printf("Failed to marshal PoolState: %v", err)
		return
	}
	ctx := context.Background()
	if err := s.client.Set(ctx, key, data, 0).Err(); err != nil {
		log.Printf("Failed to SET key %s: %v", key, err)
		return
	}
	if err := s.client.SAdd(ctx, poolsSet, key).Err(); err != nil {
		log.Printf("Failed to SADD key %s to set: %v", key, err)
	}
}

func (s *RedisUsageStore) Get(providerID, poolID string) (engine.PoolState, bool) {
	key := s.makeKey(providerID, poolID)
	ctx := context.Background()
	data, err := s.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return engine.PoolState{}, false
		}
		log.Printf("Failed to GET key %s: %v", key, err)
		return engine.PoolState{}, false
	}
	var state engine.PoolState
	if err := json.Unmarshal([]byte(data), &state); err != nil {
		log.Printf("Failed to unmarshal PoolState from key %s: %v", key, err)
		return engine.PoolState{}, false
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
	values, err := s.client.MGet(ctx, keys...).Result()
	if err != nil {
		log.Printf("Failed to MGET keys: %v", err)
		return nil
	}
	var states []engine.PoolState
	for i, val := range values {
		if val == nil {
			continue
		}
		str, ok := val.(string)
		if !ok {
			log.Printf("MGET returned non-string for key %s", keys[i])
			continue
		}
		var state engine.PoolState
		if err := json.Unmarshal([]byte(str), &state); err != nil {
			log.Printf("Failed to unmarshal PoolState for key %s: %v", keys[i], err)
			continue
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
