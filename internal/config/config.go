// Package config loads configuration and wires application infrastructure.
package config

import (
	"context"
	"encoding/hex"
	"fmt"
	"reflect"
	"strings"

	"github.com/spf13/viper"
)

// placeholderPasetoSecret is the insecure value that used to ship as the default.
// It is rejected by Validate so a deployment cannot accidentally run with a
// publicly known token-signing key.
const placeholderPasetoSecret = "your-secret-key-change-in-production"

type Config struct {
	ServiceName string `mapstructure:"SERVICE_NAME"`
	Environment string `mapstructure:"ENVIRONMENT"`

	// HTTP Server
	HTTPPort         int  `mapstructure:"HTTP_PORT"`
	Prefork          bool `mapstructure:"PREFORK"`
	HTTPReadTimeout  int  `mapstructure:"HTTP_READ_TIMEOUT"`  // seconds
	HTTPWriteTimeout int  `mapstructure:"HTTP_WRITE_TIMEOUT"` // seconds
	HTTPIdleTimeout  int  `mapstructure:"HTTP_IDLE_TIMEOUT"`  // seconds
	// RequestTimeout bounds how long a single request's downstream work
	// (DB/Redis/etc.) may run before its context is canceled. seconds.
	RequestTimeout int `mapstructure:"REQUEST_TIMEOUT"`

	// gRPC Server
	GRPCPort int `mapstructure:"GRPC_PORT"`

	// Database
	DBHost     string `mapstructure:"DB_HOST"`
	DBPort     int    `mapstructure:"DB_PORT"`
	DBUser     string `mapstructure:"DB_USER"`
	DBPassword string `mapstructure:"DB_PASSWORD"`
	DBName     string `mapstructure:"DB_NAME"`
	DBTimezone string `mapstructure:"DB_TIMEZONE"`
	DBSSLMode  string `mapstructure:"DB_SSL_MODE"`

	// Database Performance
	DBPrepareStmt            bool `mapstructure:"DB_PREPARE_STMT"`             // Enable prepared statement cache
	DBSkipDefaultTransaction bool `mapstructure:"DB_SKIP_DEFAULT_TRANSACTION"` // Disable transactions for better performance

	// Database connection pool
	DBMaxIdleConns    int `mapstructure:"DB_MAX_IDLE_CONNS"`
	DBMaxOpenConns    int `mapstructure:"DB_MAX_OPEN_CONNS"`
	DBConnMaxLifetime int `mapstructure:"DB_CONN_MAX_LIFETIME"` // minutes

	// DBAutoMigrate runs GORM AutoMigrate on startup. Defaults to false —
	// golang-migrate SQL migrations are the source of truth. Enable only for
	// local development convenience.
	DBAutoMigrate bool `mapstructure:"DB_AUTO_MIGRATE"`

	// Redis
	RedisHost         string `mapstructure:"REDIS_HOST"`
	RedisPort         int    `mapstructure:"REDIS_PORT"`
	RedisPassword     string `mapstructure:"REDIS_PASSWORD"`
	RedisDB           int    `mapstructure:"REDIS_DB"`
	RedisMaxIdle      int    `mapstructure:"REDIS_MAX_IDLE"`
	RedisMaxActive    int    `mapstructure:"REDIS_MAX_ACTIVE"`
	RedisIdleTimeout  int    `mapstructure:"REDIS_IDLE_TIMEOUT"`  // seconds
	RedisDialTimeout  int    `mapstructure:"REDIS_DIAL_TIMEOUT"`  // seconds
	RedisReadTimeout  int    `mapstructure:"REDIS_READ_TIMEOUT"`  // seconds
	RedisWriteTimeout int    `mapstructure:"REDIS_WRITE_TIMEOUT"` // seconds

	// Login protection (account lockout after repeated failures)
	LoginMaxAttempts    int `mapstructure:"LOGIN_MAX_ATTEMPTS"`
	LoginLockoutMinutes int `mapstructure:"LOGIN_LOCKOUT_MINUTES"`

	// Auth0 (optional federated login). Both AUTH0_DOMAIN and AUTH0_AUDIENCE
	// must be set to enable the /auth/auth0/login route; AUTH0_ISSUER defaults
	// to https://<domain>/ and is enforced when set explicitly.
	Auth0Domain   string `mapstructure:"AUTH0_DOMAIN"`
	Auth0Audience string `mapstructure:"AUTH0_AUDIENCE"`
	Auth0Issuer   string `mapstructure:"AUTH0_ISSUER"`

	// RabbitMQ
	RabbitMQHost     string `mapstructure:"RABBITMQ_HOST"`
	RabbitMQPort     int    `mapstructure:"RABBITMQ_PORT"`
	RabbitMQUser     string `mapstructure:"RABBITMQ_USER"`
	RabbitMQPassword string `mapstructure:"RABBITMQ_PASSWORD"`
	RabbitMQVHost    string `mapstructure:"RABBITMQ_VHOST"`

	// PASETO
	PasetoSecret          string `mapstructure:"PASETO_SECRET"`
	PasetoExpirationHours int    `mapstructure:"PASETO_EXPIRATION_HOURS"`

	// CORS
	CORSOrigins string `mapstructure:"CORS_ORIGINS"`

	// OpenTelemetry
	OTelEnabled      bool    `mapstructure:"OTEL_ENABLED"`
	OTelEndpoint     string  `mapstructure:"OTEL_ENDPOINT"`
	OTelServiceName  string  `mapstructure:"OTEL_SERVICE_NAME"`
	OTelExporterType string  `mapstructure:"OTEL_EXPORTER_TYPE"`
	OTelSampleRatio  float64 `mapstructure:"OTEL_SAMPLE_RATIO"` // 0.0-1.0; parent-based ratio sampler

	// Observability
	// MetricsAuthToken, when set, requires `Authorization: Bearer <token>` on
	// the /metrics endpoint. Empty means open (restrict at the network layer).
	MetricsAuthToken string `mapstructure:"METRICS_AUTH_TOKEN"`

	// Logger
	LogLevel  string `mapstructure:"LOG_LEVEL"`
	LogFormat string `mapstructure:"LOG_FORMAT"`
}

func New() (*Config, error) {
	v := viper.New()

	// Set defaults
	setDefaults(v)

	// Read from .env file
	v.SetConfigFile(".env")
	v.SetConfigType("env")

	// Read config file (ignore error if not found)
	_ = v.ReadInConfig() //nolint:errcheck

	// Read from environment variables.
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	// Explicitly bind every Config key to its env var. viper's Unmarshal does
	// NOT consult AutomaticEnv for keys it isn't otherwise aware of (e.g. those
	// without a default and absent from the .env file), so an env-only value
	// like PASETO_SECRET would be silently dropped without this binding.
	bindEnvs(v)

	if err := loadRemoteEnvironment(context.Background(), v); err != nil {
		return nil, err
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// bindEnvs binds every mapstructure-tagged Config field to its environment
// variable so viper.Unmarshal reliably picks up env-only overrides.
func bindEnvs(v *viper.Viper) {
	t := reflect.TypeOf(Config{})
	for i := 0; i < t.NumField(); i++ {
		if tag := t.Field(i).Tag.Get("mapstructure"); tag != "" {
			_ = v.BindEnv(tag)
		}
	}
}

func setDefaults(v *viper.Viper) {
	// Service
	v.SetDefault("SERVICE_NAME", "veemon")
	v.SetDefault("ENVIRONMENT", "development")

	// HTTP/gRPC
	v.SetDefault("HTTP_PORT", 3000)
	v.SetDefault("PREFORK", false)
	v.SetDefault("GRPC_PORT", 50051)
	v.SetDefault("HTTP_READ_TIMEOUT", 15)
	v.SetDefault("HTTP_WRITE_TIMEOUT", 30)
	v.SetDefault("HTTP_IDLE_TIMEOUT", 60)
	v.SetDefault("REQUEST_TIMEOUT", 30)

	// Database
	v.SetDefault("DB_HOST", "localhost")
	v.SetDefault("DB_PORT", 5432)
	v.SetDefault("DB_USER", "postgres")
	v.SetDefault("DB_PASSWORD", "postgres")
	v.SetDefault("DB_NAME", "veemon_db")
	v.SetDefault("DB_TIMEZONE", "Asia/Jakarta")
	v.SetDefault("DB_SSL_MODE", "disable")

	// Database Performance
	v.SetDefault("DB_PREPARE_STMT", true)              // Enable prepared statement cache for better performance
	v.SetDefault("DB_SKIP_DEFAULT_TRANSACTION", false) // Keep transactions enabled by default for data consistency

	// Database connection pool
	v.SetDefault("DB_MAX_IDLE_CONNS", 10)
	v.SetDefault("DB_MAX_OPEN_CONNS", 100)
	v.SetDefault("DB_CONN_MAX_LIFETIME", 60) // minutes

	// Schema management: golang-migrate is the source of truth; AutoMigrate off.
	v.SetDefault("DB_AUTO_MIGRATE", false)

	// Redis
	v.SetDefault("REDIS_HOST", "localhost")
	v.SetDefault("REDIS_PORT", 6379)
	v.SetDefault("REDIS_PASSWORD", "")
	v.SetDefault("REDIS_DB", 0)
	v.SetDefault("REDIS_MAX_IDLE", 10)
	v.SetDefault("REDIS_MAX_ACTIVE", 100)
	v.SetDefault("REDIS_IDLE_TIMEOUT", 240)
	v.SetDefault("REDIS_DIAL_TIMEOUT", 5)
	v.SetDefault("REDIS_READ_TIMEOUT", 3)
	v.SetDefault("REDIS_WRITE_TIMEOUT", 3)

	// Login protection
	v.SetDefault("LOGIN_MAX_ATTEMPTS", 5)
	v.SetDefault("LOGIN_LOCKOUT_MINUTES", 15)

	// Auth0 (optional)
	v.SetDefault("AUTH0_DOMAIN", "")
	v.SetDefault("AUTH0_AUDIENCE", "")
	v.SetDefault("AUTH0_ISSUER", "")

	// RabbitMQ
	v.SetDefault("RABBITMQ_HOST", "localhost")
	v.SetDefault("RABBITMQ_PORT", 5672)
	v.SetDefault("RABBITMQ_USER", "guest")
	v.SetDefault("RABBITMQ_PASSWORD", "guest")
	v.SetDefault("RABBITMQ_VHOST", "/")

	// PASETO
	// NOTE: PASETO_SECRET has no default on purpose — a shipped default is a
	// publicly known key. It must be provided via env/secret manager and is
	// enforced by Config.Validate.
	v.SetDefault("PASETO_EXPIRATION_HOURS", 24)

	// CORS
	v.SetDefault("CORS_ORIGINS", "*")

	// OpenTelemetry
	v.SetDefault("OTEL_ENABLED", true)
	v.SetDefault("OTEL_ENDPOINT", "localhost:4317")
	v.SetDefault("OTEL_SERVICE_NAME", "veemon")
	v.SetDefault("OTEL_EXPORTER_TYPE", "noop")
	v.SetDefault("OTEL_SAMPLE_RATIO", 1.0)

	// Logger
	v.SetDefault("LOG_LEVEL", "info")
	v.SetDefault("LOG_FORMAT", "json")

	// Infisical
	v.SetDefault("INFISICAL_ENABLED", false)
	v.SetDefault("INFISICAL_SITE_URL", "https://app.infisical.com")
	v.SetDefault("INFISICAL_ENVIRONMENT", "dev")
	v.SetDefault("INFISICAL_SECRET_PATH", "/")
	v.SetDefault("INFISICAL_INCLUDE_IMPORTS", true)
	v.SetDefault("INFISICAL_RECURSIVE", false)
	v.SetDefault("INFISICAL_EXPAND_SECRET_REFERENCES", true)
	v.SetDefault("INFISICAL_OVERRIDE", false)
}

// MustNew returns config or panics
func MustNew() *Config {
	cfg, err := New()
	if err != nil {
		panic(err)
	}
	return cfg
}

// Validate checks that security-sensitive configuration is safe to run with.
// It is called explicitly by processes that mint or verify tokens (the API
// server) so the process fails fast rather than silently accepting a weak or
// publicly known signing key.
func (c *Config) Validate() error {
	// Fiber prefork re-execs the whole binary per child; each child would then
	// try to bind the same gRPC port and split the Prometheus registry. This
	// topology embeds a gRPC server, so prefork is unsupported — scale out with
	// multiple replicas instead.
	if c.Prefork {
		return fmt.Errorf("PREFORK is not supported with the embedded gRPC server; run multiple replicas to scale horizontally")
	}

	s := c.PasetoSecret
	switch {
	case s == "":
		return fmt.Errorf("PASETO_SECRET is not set; generate one with `token.GenerateSecretKey` and set PASETO_SECRET")
	case s == placeholderPasetoSecret:
		return fmt.Errorf("PASETO_SECRET still uses the insecure placeholder value; set a unique secret before starting")
	}

	// Accept a 64-char hex string (32 bytes) …
	if len(s) == 64 {
		if _, err := hex.DecodeString(s); err == nil {
			return c.validateCORS()
		}
	}
	// … or a raw secret of at least 32 bytes.
	if len(s) < 32 {
		return fmt.Errorf("PASETO_SECRET must be a 64-character hex string or at least 32 bytes (got %d bytes)", len(s))
	}
	return c.validateCORS()
}

// validateCORS rejects a wildcard CORS origin in production, where it would
// allow any site to make credentialed cross-origin requests.
func (c *Config) validateCORS() error {
	if c.Environment == "production" && strings.TrimSpace(c.CORSOrigins) == "*" {
		return fmt.Errorf("CORS_ORIGINS must not be '*' in production; set explicit allowed origins")
	}
	return nil
}

func loadRemoteEnvironment(ctx context.Context, v *viper.Viper) error {
	infisicalCfg := InfisicalConfig{
		Enabled:                v.GetBool("INFISICAL_ENABLED"),
		SiteURL:                v.GetString("INFISICAL_SITE_URL"),
		ClientID:               v.GetString("INFISICAL_CLIENT_ID"),
		ClientSecret:           v.GetString("INFISICAL_CLIENT_SECRET"),
		ProjectID:              v.GetString("INFISICAL_PROJECT_ID"),
		ProjectSlug:            v.GetString("INFISICAL_PROJECT_SLUG"),
		Environment:            v.GetString("INFISICAL_ENVIRONMENT"),
		SecretPath:             v.GetString("INFISICAL_SECRET_PATH"),
		IncludeImports:         v.GetBool("INFISICAL_INCLUDE_IMPORTS"),
		Recursive:              v.GetBool("INFISICAL_RECURSIVE"),
		ExpandSecretReferences: v.GetBool("INFISICAL_EXPAND_SECRET_REFERENCES"),
		Override:               v.GetBool("INFISICAL_OVERRIDE"),
		OrganizationSlug:       v.GetString("INFISICAL_ORGANIZATION_SLUG"),
	}
	if !infisicalCfg.Enabled {
		return nil
	}

	secrets, err := loadInfisicalSecrets(ctx, infisicalCfg)
	if err != nil {
		return fmt.Errorf("load infisical secrets: %w", err)
	}

	applyInfisicalSecrets(secrets, infisicalCfg.Override)
	return nil
}
