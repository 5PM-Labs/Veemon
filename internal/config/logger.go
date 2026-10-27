package config

import (
	"github.com/5PM-Labs/veemon/internal/infra/logging"
)

func NewLogger(cfg *Config) (*logging.Logger, error) {
	return logging.New(logging.Config{
		Level:       cfg.LogLevel,
		Format:      cfg.LogFormat,
		Environment: cfg.Environment,
		ServiceName: cfg.ServiceName,
	})
}
