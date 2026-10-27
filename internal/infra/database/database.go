// Package database provides the Postgres connection pool (pgx/v5) and the thin
// query helpers the repositories build on.
//
// pgx is used directly rather than through an ORM: every query in this repo is
// explicit SQL, so schema changes stay reviewable and there is no tag/identity-
// map indirection between the structs and the tables.
package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

// Config holds database connection parameters. Password may be empty for
// passwordless (peer/cert) authentication.
type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	Name     string
	SSLMode  string
	Timezone string

	// Connection pool. Zero values fall back to the defaults below.
	MaxIdleConns    int
	MaxOpenConns    int
	ConnMaxLifetime time.Duration
}

// Queryer is satisfied by both *pgxpool.Pool and pgx.Tx, so repository methods
// written against it work inside and outside a transaction unchanged.
type Queryer interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// DSN returns the libpq connection string for cfg.
func (c Config) DSN() string {
	sslMode := c.SSLMode
	if sslMode == "" {
		sslMode = "disable"
	}
	// Key/value form (not URL) so passwords with URL-reserved characters need no
	// escaping. TimeZone is passed as a runtime parameter.
	return fmt.Sprintf(
		"host=%s port=%d user=%s dbname=%s sslmode=%s TimeZone=%s",
		c.Host, c.Port, c.User, c.Name, sslMode, c.Timezone,
	)
}

// password returns the password for the DSN, or "" when unset.
// (Kept separate so DSN stays readable.)
func (c Config) password() string { return c.Password }

// WithPassword appends the password to a rendered DSN. Exposed so callers that
// render the DSN themselves (cmd/migrate) share one code path.
func (c Config) WithPassword(dsn string) string {
	p := c.password()
	if p == "" {
		return dsn
	}
	return dsn + " password=" + p
}

// New opens the pool and fails fast if the database is unreachable.
func New(ctx context.Context, cfg Config, log *zap.Logger) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.WithPassword(cfg.DSN()))
	if err != nil {
		return nil, fmt.Errorf("failed to parse database config: %w", err)
	}

	maxOpen := cfg.MaxOpenConns
	if maxOpen <= 0 {
		maxOpen = 100
	}
	poolCfg.MaxConns = int32(maxOpen)
	if cfg.MaxIdleConns > 0 {
		poolCfg.MinConns = int32(cfg.MaxIdleConns)
	}
	if cfg.ConnMaxLifetime > 0 {
		poolCfg.MaxConnLifetime = cfg.ConnMaxLifetime
	}
	poolCfg.MaxConnIdleTime = 30 * time.Minute

	// Trace every query through OTel. Only the low-cardinality operation+table
	// summary is attached — never bound values (which carry emails and hashes).
	poolCfg.ConnConfig.Tracer = &otelTracer{}

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	log.Info("Database connection established",
		zap.String("host", cfg.Host),
		zap.Int("port", cfg.Port),
		zap.String("database", cfg.Name),
		zap.Int("max_conns", int(poolCfg.MaxConns)),
	)
	return pool, nil
}

// otelTracer is a pgx QueryTracer emitting one span per query.
type otelTracer struct{}

var tracer = otel.Tracer("github.com/5PM-Labs/veemon/internal/infra/database")

func (t *otelTracer) TraceQueryStart(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	ctx, span := tracer.Start(ctx, "db.query")
	span.SetAttributes(attribute.String("db.statement", OperationSummary(data.SQL)))
	return ctx
}

func (t *otelTracer) TraceQueryEnd(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryEndData) {
	span := tracerFromContext(ctx)
	if span == nil {
		return
	}
	if data.Err != nil {
		span.RecordError(data.Err)
	}
	span.SetAttributes(attribute.Int64("db.rows_affected", data.CommandTag.RowsAffected()))
	span.End()
}

// InTx runs fn inside a transaction, committing on success and rolling back on
// error or panic.
func InTx(ctx context.Context, pool *pgxpool.Pool, fn func(pgx.Tx) error) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		// Rollback after a successful commit is a no-op error; ignore it.
		_ = tx.Rollback(context.WithoutCancel(ctx))
	}()

	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// IsUniqueViolation reports whether err is a Postgres unique-constraint error.
// Callers translate it to domain.ErrDuplicateEntry so no layer above the
// repository imports pgconn.
func IsUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
