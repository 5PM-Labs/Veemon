package config

import (
	"context"

	"github.com/5PM-Labs/veemon/internal/infra/logging"
	"github.com/5PM-Labs/veemon/internal/infra/monitoring/telemetry"
	"github.com/5PM-Labs/veemon/internal/infra/redis"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// CommonModule provides the infrastructure shared byte-for-byte between
// cmd/server and cmd/worker: config, logger, telemetry, database, and an
// optional Redis client. Each provider registers its own OnStop hook via the
// injected fx.Lifecycle, so fx tears infra down in the reverse order it was
// built — this is what replaces the near-identical hand-rolled init/defer
// blocks that used to live in both main.go files.
var CommonModule = fx.Module("infra",
	fx.Provide(
		New, // config.New: load + bind env, no validation (server-only; see ServerModule)
		provideLogger,
		provideZapLogger,
		provideTelemetry,
		provideDatabase,
		provideOptionalRedis,
	),
)

// provideLogger builds the structured logger and registers a best-effort
// Sync on shutdown. Sync's error is intentionally swallowed (matching
// .golangci.yml's errcheck exclude for (*zap.Logger).Sync): syncing stdout
// commonly errors on Linux (ENOTTY) and must never fail a clean shutdown.
func provideLogger(lc fx.Lifecycle, cfg *Config) (*logging.Logger, error) {
	log, err := NewLogger(cfg)
	if err != nil {
		return nil, err
	}
	lc.Append(fx.Hook{
		OnStop: func(context.Context) error {
			_ = log.Sync()
			return nil
		},
	})
	return log, nil
}

// provideZapLogger exposes the embedded *zap.Logger so the rest of the graph
// (which only knows about go.uber.org/zap, not this package's Logger wrapper)
// can depend on it directly, same as every config/*.go constructor already did.
func provideZapLogger(log *logging.Logger) *zap.Logger {
	return log.Logger
}

// provideTelemetry initializes OpenTelemetry and registers a shutdown hook.
// Like the logger, a shutdown error is logged but never fails teardown.
func provideTelemetry(lc fx.Lifecycle, cfg *Config, log *zap.Logger) (*telemetry.Telemetry, error) {
	tel, err := NewTelemetry(context.Background(), cfg)
	if err != nil {
		return nil, err
	}
	log.Info("OpenTelemetry initialized",
		zap.Bool("enabled", cfg.OTelEnabled),
		zap.String("exporter", cfg.OTelExporterType),
	)
	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			if err := tel.Shutdown(ctx); err != nil {
				log.Warn("telemetry shutdown error", zap.Error(err))
			}
			return nil
		},
	})
	return tel, nil
}

// provideDatabase connects to Postgres and registers a close hook. A
// connection failure is fatal in both cmd/server and cmd/worker (matching
// both main.go's original log.Fatal), so the error is propagated as-is —
// fx aborts startup (and rolls back anything already started) on error.
func provideDatabase(lc fx.Lifecycle, cfg *Config, log *zap.Logger) (*pgxpool.Pool, error) {
	pool, err := NewDatabase(context.Background(), cfg, log)
	if err != nil {
		return nil, err
	}
	lc.Append(fx.Hook{
		OnStop: func(context.Context) error {
			pool.Close()
			return nil
		},
	})
	return pool, nil
}

// provideOptionalRedis connects to Redis but never fails application
// startup: a Redis outage degrades to caching/lockout disabled, matching
// both original main.go's "log.Warn and continue with a nil client" handling.
func provideOptionalRedis(lc fx.Lifecycle, cfg *Config, log *zap.Logger) *redis.Client {
	client, err := NewRedis(cfg)
	if err != nil {
		log.Warn("Failed to connect to Redis, caching disabled", zap.Error(err))
		return nil
	}
	log.Info("Redis connection established",
		zap.String("host", cfg.RedisHost),
		zap.Int("port", cfg.RedisPort),
	)
	lc.Append(fx.Hook{
		OnStop: func(context.Context) error {
			_ = client.Close()
			return nil
		},
	})
	return client
}
