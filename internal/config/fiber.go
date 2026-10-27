package config

import (
	stdErrors "errors"
	"strings"
	"time"

	"github.com/5PM-Labs/veemon/internal/infra/redis"
	"github.com/5PM-Labs/veemon/internal/pkg/errors"
	"github.com/5PM-Labs/veemon/internal/pkg/middleware"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/helmet"
	"go.uber.org/zap"
)

// NewFiber builds the HTTP app. redisClient may be nil (Redis unavailable): the
// rate limiter then falls back to per-process counters, which is still correct
// for a single replica but does not share limits across replicas.
func NewFiber(cfg *Config, log *zap.Logger, redisClient *redis.Client) *fiber.App {
	// Fiber v3 moved Prefork and DisableStartupMessage out of fiber.Config and
	// into fiber.ListenConfig; they are applied by NewListenConfig at Listen time.
	app := fiber.New(fiber.Config{
		AppName:      cfg.ServiceName,
		ErrorHandler: NewErrorHandler(log),
		// Bound slow/idle connections so a stuck client cannot hold a worker
		// indefinitely. Values are conservative defaults; tune per workload.
		ReadTimeout:  time.Duration(cfg.HTTPReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.HTTPWriteTimeout) * time.Second,
		IdleTimeout:  time.Duration(cfg.HTTPIdleTimeout) * time.Second,
	})

	// Security response headers (X-Frame-Options, X-Content-Type-Options, etc.)
	app.Use(helmet.New())

	// CORS. Fiber v3 takes these as slices rather than comma-joined strings.
	corsConfig := cors.Config{
		AllowOrigins: splitAndTrim(cfg.CORSOrigins),
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		AllowHeaders: []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Request-ID", "X-Trace-ID"},
	}
	// AllowCredentials cannot be used with wildcard origins
	if cfg.CORSOrigins != "*" {
		corsConfig.AllowCredentials = true
	}
	app.Use(cors.New(corsConfig))

	// Middleware (order matters — outermost first). Logger is placed OUTSIDE
	// Recovery so that a panic (recovered below into a 500) is still logged and
	// traced; Recovery wraps the handler so it can turn panics into responses.
	app.Use(middleware.RequestIDMiddleware())
	app.Use(middleware.TracingMiddleware(cfg.ServiceName))
	app.Use(middleware.LoggerMiddleware(log))
	// Global per-IP rate limit as a coarse abuse guard. Stricter, endpoint-
	// specific limits are applied on auth routes during route registration.
	app.Use(middleware.RateLimitMiddleware(rateLimitConfig(middleware.DefaultRateLimitConfig(), redisClient, "global")))
	app.Use(middleware.RecoveryMiddleware(log))
	app.Use(middleware.TimeoutMiddleware(time.Duration(cfg.RequestTimeout) * time.Second))

	return app
}

// rateLimitConfig attaches a shared Redis storage (when available) so the
// counter is cluster-wide rather than per replica. scope namespaces the keys so
// the global and per-route limiters never count into each other.
func rateLimitConfig(cfg middleware.RateLimitConfig, redisClient *redis.Client, scope string) middleware.RateLimitConfig {
	if redisClient != nil {
		cfg.Storage = middleware.NewRedisStorage(redisClient, "veemon:ratelimit:"+scope+":")
	}
	return cfg
}

// NewListenConfig carries the settings Fiber v3 moved out of fiber.Config.
// Prefork stays configurable here, but Config.Validate() rejects PREFORK=true
// because the embedded gRPC server is incompatible with it.
func NewListenConfig(cfg *Config) fiber.ListenConfig {
	return fiber.ListenConfig{
		DisableStartupMessage: true,
		EnablePrefork:         cfg.Prefork,
	}
}

// splitAndTrim turns a comma-separated config value into a slice, dropping
// empty entries so a trailing comma does not become an empty origin.
func splitAndTrim(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// NewErrorHandler is the single place that turns a returned error into the
// REST error envelope: *errors.AppError keeps its status/code/message, a
// *fiber.Error keeps its status, and anything else becomes a sanitized 500
// (the cause is logged, never sent to the client).
func NewErrorHandler(log *zap.Logger) fiber.ErrorHandler {
	return func(c fiber.Ctx, err error) error {
		var appErr *errors.AppError
		var fiberErr *fiber.Error

		switch {
		case stdErrors.As(err, &appErr):
			// Application errors are expected responses, not failures: log at
			// info/warn by status instead of error level for every 4xx.
			if appErr.HTTPStatus >= fiber.StatusInternalServerError {
				log.Error("Request failed", zap.Int("status", appErr.HTTPStatus),
					zap.Error(err), zap.String("path", c.Path()))
			}
			return appErr.FiberError(c)
		case stdErrors.As(err, &fiberErr):
			code := fiberErr.Code
			if code >= fiber.StatusInternalServerError {
				log.Error("Request failed", zap.Int("status", code),
					zap.Error(err), zap.String("path", c.Path()))
			}
			return c.Status(code).JSON(fiber.Map{
				"success": false,
				"error":   errors.ErrorBody{Code: code, Message: fiberErr.Message},
			})
		default:
			log.Error("Request failed",
				zap.Int("status", fiber.StatusInternalServerError),
				zap.Error(err),
				zap.String("path", c.Path()),
			)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"error":   errors.ErrorBody{Code: 500, Message: "Internal Server Error"},
			})
		}
	}
}
