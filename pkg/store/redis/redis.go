package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rmax-ai/ratelord/pkg/engine"
)

type RedisUsageStore struct {
	client *redis.Client
}

func NewRedisUsageStore(url string) *RedisUsageStore {
	opt, err := redis.ParseURL(url)
	if err != nil {
		panic(fmt.Sprintf("failed to parse Redis URL: %v", err))
	}
	client := redis.NewClient(opt)
	return &RedisUsageStore{client: client}
}

func (s *RedisUsageStore) Get(providerID, poolID string) (engine.PoolState, bool) {
	ctx := context.Background()
	key := fmt.Sprintf("ratelord:usage:%s:%s", providerID, poolID)
	val, err := s.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return engine.PoolState{}, false
	}
	if err != nil {
		return engine.PoolState{}, false
	}
	var state engine.PoolState
	if err := json.Unmarshal([]byte(val), &state); err != nil {
		return engine.PoolState{}, false
	}
	return state, true
}

func (s *RedisUsageStore) Set(state engine.PoolState) {
	ctx := context.Background()
	key := fmt.Sprintf("ratelord:usage:%s:%s", state.ProviderID, state.PoolID)
	data, err := json.Marshal(state)
	if err != nil {
		return
	}
	var exp time.Duration
	if state.ResetAt.After(time.Now()) {
		exp = time.Until(state.ResetAt)
	}
	s.client.Set(ctx, key, data, exp)
}

func (s *RedisUsageStore) GetAll() []engine.PoolState {
	ctx := context.Background()
	var states []engine.PoolState
	iter := s.client.Scan(ctx, 0, "ratelord:usage:*", 0).Iterator()
	var keys []string
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}
	if err := iter.Err(); err != nil {
		return nil
	}
	if len(keys) == 0 {
		return states
	}
	vals, err := s.client.MGet(ctx, keys...).Result()
	if err != nil {
		return nil
	}
	for _, v := range vals {
		if v == nil {
			continue
		}
		str, ok := v.(string)
		if !ok {
			continue
		}
		var state engine.PoolState
		if err := json.Unmarshal([]byte(str), &state); err != nil {
			continue
		}
		states = append(states, state)
	}
	return states
}

func (s *RedisUsageStore) Clear() {
	ctx := context.Background()
	iter := s.client.Scan(ctx, 0, "ratelord:usage:*", 0).Iterator()
	var keys []string
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}
	if err := iter.Err(); err != nil {
		return
	}
	if len(keys) > 0 {
		s.client.Del(ctx, keys...)
	}
}
