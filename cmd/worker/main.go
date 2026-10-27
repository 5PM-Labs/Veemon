// Command worker consumes messages from RabbitMQ.
package main

import (
	"time"

	"github.com/5PM-Labs/veemon/internal/config"

	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
)

func main() {
	fx.New(
		fx.StopTimeout(30*time.Second),
		fx.WithLogger(func(log *zap.Logger) fxevent.Logger {
			return &fxevent.ZapLogger{Logger: log}
		}),
		config.CommonModule,
		config.WorkerModule,
	).Run()
}
