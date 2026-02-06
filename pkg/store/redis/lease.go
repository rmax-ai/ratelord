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

func (s *RedisLeaseStore) Acquire(ctx context.Context, name, holderID string, ttl time.Duration) (bool, error) {
	key := s.makeKey(name)

	// NX: Only set if not exists
	// Expiration: Set the TTL
	success, err := s.client.SetNX(ctx, key, holderID, ttl).Result()
	if err != nil {
		return false, fmt.Errorf("failed to acquire lease: %w", err)
	}

	if success {
		return true, nil
	}

	// If failed, check if we already hold it and just need to renew (idempotency)
	val, err := s.client.Get(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check existing lease: %w", err)
	}

	if val == holderID {
		// We already hold it, renew it
		return true, s.Renew(ctx, name, holderID, ttl)
	}

	return false, nil
}

func (s *RedisLeaseStore) Renew(ctx context.Context, name, holderID string, ttl time.Duration) error {
	key := s.makeKey(name)

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

	res, err := s.client.Eval(ctx, script, []string{key}, holderID, ttlMs).Result()
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
	key := s.makeKey(name)

	// Lua script to check if holder matches before deleting
	script := `
		if redis.call("GET", KEYS[1]) == ARGV[1] then
			return redis.call("DEL", KEYS[1])
		else
			return 0
		end
	`

	_, err := s.client.Eval(ctx, script, []string{key}, holderID).Result()
	if err != nil {
		return fmt.Errorf("failed to execute release script: %w", err)
	}

	// We don't necessarily error if we didn't hold it (idempotency),
	// but strictly speaking Release() implies we thought we held it.
	// The interface contract says "releases the lease if held".
	// So returning nil is fine even if we didn't delete anything (already gone or stolen).
	// However, for debugging it might be useful to know.
	// Let's stick to the interface: no error means "operation completed",
	// ensuring we don't hold it anymore (which is true if we didn't hold it).

	return nil
}

func (s *RedisLeaseStore) Get(ctx context.Context, name string) (*store.Lease, error) {
	key := s.makeKey(name)

	val, err := s.client.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil // No lease held
		}
		return nil, fmt.Errorf("failed to get lease: %w", err)
	}

	ttl, err := s.client.PTTL(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get lease ttl: %w", err)
	}

	return &store.Lease{
		Name:      name,
		HolderID:  val,
		ExpiresAt: time.Now().Add(ttl),
		Version:   0, // Redis simplistic implementation doesn't strictly track version CAS
	}, nil
}
