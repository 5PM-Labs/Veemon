package middleware

import (
	"context"
	"time"

	"veemon-common/infra/redis"

	goredis "github.com/gomodule/redigo/redis"
)

// RedisStorage adapts the shared go-common Redigo client to fiber.Storage, so
// rate limits are shared across every replica instead of counted per process.
// Without it, a login limit of 10/min becomes 10/min/replica — no limit at all
// once the service scales horizontally.
//
// The fixed-window limiter keeps one counter per key, so plain SET/GET with a
// TTL is enough; no atomic INCR is needed beyond what that permits.
type RedisStorage struct {
	client *redis.Client
	prefix string
}

// NewRedisStorage wraps client. prefix namespaces keys, e.g. "veemon:ratelimit:".
func NewRedisStorage(client *redis.Client, prefix string) *RedisStorage {
	return &RedisStorage{client: client, prefix: prefix}
}

func (s *RedisStorage) Get(key string) ([]byte, error) {
	return s.GetWithContext(context.Background(), key)
}

func (s *RedisStorage) GetWithContext(ctx context.Context, key string) ([]byte, error) {
	conn := s.client.Conn()
	defer conn.Close() //nolint:errcheck // best-effort cleanup

	b, err := goredis.Bytes(conn.Do("GET", s.prefix+key))
	if err == goredis.ErrNil {
		return nil, nil // fiber.Storage contract: missing key is (nil, nil)
	}
	return b, err
}

func (s *RedisStorage) Set(key string, val []byte, exp time.Duration) error {
	return s.SetWithContext(context.Background(), key, val, exp)
}

func (s *RedisStorage) SetWithContext(_ context.Context, key string, val []byte, exp time.Duration) error {
	conn := s.client.Conn()
	defer conn.Close() //nolint:errcheck // best-effort cleanup

	if exp > 0 {
		_, err := conn.Do("SET", s.prefix+key, val, "PX", exp.Milliseconds())
		return err
	}
	_, err := conn.Do("SET", s.prefix+key, val)
	return err
}

func (s *RedisStorage) Delete(key string) error {
	return s.DeleteWithContext(context.Background(), key)
}

func (s *RedisStorage) DeleteWithContext(_ context.Context, key string) error {
	conn := s.client.Conn()
	defer conn.Close() //nolint:errcheck // best-effort cleanup

	_, err := conn.Do("DEL", s.prefix+key)
	return err
}

// Reset intentionally does nothing: honoring it would mean SCAN + DEL across
// the whole keyspace, wiping every replica's counters. Keys expire on their own.
func (s *RedisStorage) Reset() error { return nil }

func (s *RedisStorage) ResetWithContext(context.Context) error { return nil }

// Close intentionally does nothing: the client is owned by the fx lifecycle,
// which closes it on shutdown.
func (s *RedisStorage) Close() error { return nil }
