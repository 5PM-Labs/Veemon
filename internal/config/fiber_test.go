package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	apperrors "github.com/5PM-Labs/veemon/internal/pkg/errors"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// One envelope for every error source: a handler-returned AppError, a wrapped
// AppError (errors.As), a fiber.Error, and an unknown error.
func TestNewErrorHandler_Envelope(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		wantStatus  int
		wantCode    int
		wantMessage string
	}{
		{"app error keeps its code", apperrors.Conflict(40901, "email already registered"), 409, 40901, "email already registered"},
		{"wrapped app error is unwrapped", wrapErr(apperrors.Unauthorized("invalid token")), 401, 401, "invalid token"},
		{"fiber error keeps status", fiber.NewError(fiber.StatusNotFound, "not here"), 404, 404, "not here"},
		{"unknown error is sanitized", errors.New("db password leaked in message"), 500, 500, "Internal Server Error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New(fiber.Config{ErrorHandler: NewErrorHandler(zap.NewNop())})
			app.Get("/", func(fiber.Ctx) error { return tt.err })

			resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/", nil))
			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, resp.StatusCode)

			var body struct {
				Success bool `json:"success"`
				Error   struct {
					Code    int    `json:"code"`
					Message string `json:"message"`
				} `json:"error"`
			}
			require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
			assert.False(t, body.Success)
			assert.Equal(t, tt.wantCode, body.Error.Code)
			assert.Equal(t, tt.wantMessage, body.Error.Message)
		})
	}
}

func wrapErr(err error) error { return fmt.Errorf("login: %w", err) }
