package redis

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rmax-ai/ratelord/pkg/store"
)

type RedisLeaseStore struct {
	client *redis.Client
}

func NewRedisLeaseStore(client *redis.Client) *RedisLeaseStore {
	return &RedisLeaseStore{client: client}
}

func (s *RedisLeaseStore) makeKey(name string) string {
	return fmt.Sprintf("ratelord:lease:%s", name)
}

func (s *RedisLeaseStore) makeEpochKey(name string) string {
	return fmt.Sprintf("ratelord:epoch:%s", name)
}

func (s *RedisLeaseStore) Acquire(ctx context.Context, name, holderID string, ttl time.Duration) (bool, error) {
	leaseKey := s.makeKey(name)
	epochKey := s.makeEpochKey(name)

	// Lua script to acquire lease and increment epoch if successful
	script := `
		local leaseKey = KEYS[1]
		local epochKey = KEYS[2]
		local holderID = ARGV[1]
		local ttlMs = ARGV[2]

		-- Check if lease exists
		local currentHolder = redis.call("GET", leaseKey)

		if currentHolder then
			if currentHolder == holderID then
				-- We already hold it, renew TTL
				redis.call("PEXPIRE", leaseKey, ttlMs)
				return 1
			else
				-- Someone else holds it
				return 0
			end
		else
			-- Lease is free, take it!
			-- Increment epoch
			redis.call("INCR", epochKey)
			-- Set lease
			redis.call("SET", leaseKey, holderID, "PX", ttlMs)
			return 1
		end
	`

	ttlMs := int64(ttl / time.Millisecond)
	res, err := s.client.Eval(ctx, script, []string{leaseKey, epochKey}, holderID, ttlMs).Result()
	if err != nil {
		return false, fmt.Errorf("failed to execute acquire script: %w", err)
	}

	success, ok := res.(int64)
	if !ok {
		return false, fmt.Errorf("unexpected return type from acquire script")
	}

	return success == 1, nil
}

func (s *RedisLeaseStore) Renew(ctx context.Context, name, holderID string, ttl time.Duration) error {
	leaseKey := s.makeKey(name)

	// Lua script to check if holder matches before extending expiry
	script := `
		if redis.call("GET", KEYS[1]) == ARGV[1] then
			return redis.call("PEXPIRE", KEYS[1], ARGV[2])
		else
			return 0
		end
	`

	// PEXPIRE takes milliseconds
	ttlMs := int64(ttl / time.Millisecond)

	res, err := s.client.Eval(ctx, script, []string{leaseKey}, holderID, ttlMs).Result()
	if err != nil {
		return fmt.Errorf("failed to execute renew script: %w", err)
	}

	success, ok := res.(int64)
	if !ok {
		return fmt.Errorf("unexpected return type from renew script")
	}

	if success == 1 {
		return nil
	}

	return fmt.Errorf("lease lost or stolen")
}

func (s *RedisLeaseStore) Release(ctx context.Context, name, holderID string) error {
	leaseKey := s.makeKey(name)

	// Lua script to check if holder matches before deleting
	script := `
		if redis.call("GET", KEYS[1]) == ARGV[1] then
			return redis.call("DEL", KEYS[1])
		else
			return 0
		end
	`

	_, err := s.client.Eval(ctx, script, []string{leaseKey}, holderID).Result()
	if err != nil {
		return fmt.Errorf("failed to execute release script: %w", err)
	}

	return nil
}

func (s *RedisLeaseStore) Get(ctx context.Context, name string) (*store.Lease, error) {
	leaseKey := s.makeKey(name)
	epochKey := s.makeEpochKey(name)

	// Use pipeline to get both values
	pipe := s.client.Pipeline()
	holderCmd := pipe.Get(ctx, leaseKey)
	ttlCmd := pipe.PTTL(ctx, leaseKey)
	epochCmd := pipe.Get(ctx, epochKey)

	_, _ = pipe.Exec(ctx) // Ignore errors here, check individual cmds

	holder, err := holderCmd.Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return nil, fmt.Errorf("failed to get lease holder: %w", err)
	}

	if errors.Is(err, redis.Nil) {
		return nil, nil // No lease held
	}

	ttl, err := ttlCmd.Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get lease ttl: %w", err)
	}

	epochStr, err := epochCmd.Result()
	var epoch int64
	if err == nil {
		fmt.Sscanf(epochStr, "%d", &epoch)
	}
	// If epoch key missing, default to 0

	return &store.Lease{
		Name:      name,
		HolderID:  holder,
		ExpiresAt: time.Now().Add(ttl),
		Version:   epoch,
		Epoch:     epoch,
	}, nil
}
