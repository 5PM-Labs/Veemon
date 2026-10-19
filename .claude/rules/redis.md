# Redis Best Practices

## 1. Connection Pool with Redigo

```go
// ✓ Good: Use connection pool
pool := &redis.Pool{
    MaxIdle:     10,
    MaxActive:   100,
    IdleTimeout: 240 * time.Second,
    Wait:        true,
    Dial: func() (redis.Conn, error) {
        return redis.Dial("tcp", addr)
    },
}

// ✓ Good: Always close connection
conn := pool.Get()
defer conn.Close()

// ✓ Good: Check connection errors
if err := conn.Err(); err != nil {
    return err
}
```

Redis backs the login lockout + token revocation store (`internal/pkg/authguard`); see
[security](security.md) A07.
