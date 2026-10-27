package config

import (
	"context"
	"fmt"
	"time"

	"github.com/5PM-Labs/veemon/internal/infra/database"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// NewDatabase opens the pgx pool.
//
// Schema management is external: golang-migrate SQL migrations (see
// migrations/) are the single source of truth, so this constructor never
// creates or alters tables. It does verify the schema exists so a forgotten
// `make migrate` fails at startup instead of on the first request.
func NewDatabase(ctx context.Context, cfg *Config, log *zap.Logger) (*pgxpool.Pool, error) {
	pool, err := database.New(ctx, database.Config{
		Host:     cfg.DBHost,
		Port:     cfg.DBPort,
		User:     cfg.DBUser,
		Password: cfg.DBPassword,
		Name:     cfg.DBName,
		SSLMode:  cfg.DBSSLMode,
		Timezone: cfg.DBTimezone,

		MaxIdleConns:    cfg.DBMaxIdleConns,
		MaxOpenConns:    cfg.DBMaxOpenConns,
		ConnMaxLifetime: time.Duration(cfg.DBConnMaxLifetime) * time.Minute,
	}, log)
	if err != nil {
		return nil, err
	}

	var usersTable *string
	if err := pool.QueryRow(ctx, "SELECT to_regclass('public.users')").Scan(&usersTable); err != nil || usersTable == nil {
		pool.Close()
		return nil, fmt.Errorf("users table missing; run `make migrate` first")
	}

	return pool, nil
}
