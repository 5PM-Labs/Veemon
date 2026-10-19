package config

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestRegisterObservabilityRoutes(t *testing.T) {
	app := fiber.New()

	registerObservabilityRoutes(app, &Config{ServiceName: "test_service"}, zap.NewNop())

	metricsResp, err := app.Test(httptest.NewRequest(http.MethodGet, "/metrics", nil))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, metricsResp.StatusCode)

	docsResp, err := app.Test(httptest.NewRequest(http.MethodGet, "/docs/openapi.json", nil))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, docsResp.StatusCode)
}

func TestMetricsAuthToken(t *testing.T) {
	app := fiber.New()
	registerObservabilityRoutes(app, &Config{ServiceName: "test_service", MetricsAuthToken: "secret"}, zap.NewNop())

	// Without the token, /metrics is rejected.
	unauth, err := app.Test(httptest.NewRequest(http.MethodGet, "/metrics", nil))
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, unauth.StatusCode)

	// With the correct bearer token, it succeeds.
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	req.Header.Set("Authorization", "Bearer secret")
	authed, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, authed.StatusCode)
}

// Production must not publish the API docs UI even though /metrics stays up.
func TestRegisterObservabilityRoutes_HidesDocsInProduction(t *testing.T) {
	app := fiber.New()
	registerObservabilityRoutes(app, &Config{ServiceName: "test_service", Environment: "production"}, zap.NewNop())

	metricsResp, err := app.Test(httptest.NewRequest(http.MethodGet, "/metrics", nil))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, metricsResp.StatusCode)

	docsResp, err := app.Test(httptest.NewRequest(http.MethodGet, "/docs/openapi.json", nil))
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, docsResp.StatusCode)
}
