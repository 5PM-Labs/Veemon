package config

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLoadsInfisicalSecretsBeforeUnmarshal(t *testing.T) {
	t.Setenv("INFISICAL_ENABLED", "true")
	t.Setenv("INFISICAL_CLIENT_ID", "client-id")
	t.Setenv("INFISICAL_CLIENT_SECRET", "client-secret")
	t.Setenv("INFISICAL_PROJECT_ID", "project-id")
	t.Setenv("INFISICAL_ENVIRONMENT", "dev")
	t.Setenv("DB_PASSWORD", "")

	restore := setInfisicalSecretLoaderForTest(t, func(ctx context.Context, cfg InfisicalConfig) (map[string]string, error) {
		assert.Equal(t, "client-id", cfg.ClientID)
		assert.Equal(t, "client-secret", cfg.ClientSecret)
		assert.Equal(t, "project-id", cfg.ProjectID)
		assert.Equal(t, "dev", cfg.Environment)
		return map[string]string{
			"DB_PASSWORD": "from-infisical",
			"DB_NAME":     "infisical_db",
		}, nil
	})
	defer restore()

	cfg, err := New()

	require.NoError(t, err)
	assert.Equal(t, "from-infisical", cfg.DBPassword)
	assert.Equal(t, "infisical_db", cfg.DBName)
}

func TestNewKeepsExistingEnvOverInfisicalByDefault(t *testing.T) {
	t.Setenv("INFISICAL_ENABLED", "true")
	t.Setenv("INFISICAL_CLIENT_ID", "client-id")
	t.Setenv("INFISICAL_CLIENT_SECRET", "client-secret")
	t.Setenv("INFISICAL_PROJECT_ID", "project-id")
	t.Setenv("INFISICAL_ENVIRONMENT", "dev")
	t.Setenv("DB_PASSWORD", "from-process-env")

	restore := setInfisicalSecretLoaderForTest(t, func(ctx context.Context, cfg InfisicalConfig) (map[string]string, error) {
		return map[string]string{"DB_PASSWORD": "from-infisical"}, nil
	})
	defer restore()

	cfg, err := New()

	require.NoError(t, err)
	assert.Equal(t, "from-process-env", cfg.DBPassword)
}

func TestNewCanOverrideExistingEnvWithInfisical(t *testing.T) {
	t.Setenv("INFISICAL_ENABLED", "true")
	t.Setenv("INFISICAL_OVERRIDE", "true")
	t.Setenv("INFISICAL_CLIENT_ID", "client-id")
	t.Setenv("INFISICAL_CLIENT_SECRET", "client-secret")
	t.Setenv("INFISICAL_PROJECT_ID", "project-id")
	t.Setenv("INFISICAL_ENVIRONMENT", "dev")
	t.Setenv("DB_PASSWORD", "from-process-env")

	restore := setInfisicalSecretLoaderForTest(t, func(ctx context.Context, cfg InfisicalConfig) (map[string]string, error) {
		return map[string]string{"DB_PASSWORD": "from-infisical"}, nil
	})
	defer restore()

	cfg, err := New()

	require.NoError(t, err)
	assert.Equal(t, "from-infisical", cfg.DBPassword)
}

func TestNewSkipsInfisicalWhenDisabled(t *testing.T) {
	t.Setenv("INFISICAL_ENABLED", "false")

	restore := setInfisicalSecretLoaderForTest(t, func(ctx context.Context, cfg InfisicalConfig) (map[string]string, error) {
		t.Fatal("infisical loader should not run when disabled")
		return nil, nil
	})
	defer restore()

	_, err := New()

	require.NoError(t, err)
}

func TestNewReturnsInfisicalLoaderError(t *testing.T) {
	t.Setenv("INFISICAL_ENABLED", "true")
	t.Setenv("INFISICAL_CLIENT_ID", "client-id")
	t.Setenv("INFISICAL_CLIENT_SECRET", "client-secret")
	t.Setenv("INFISICAL_PROJECT_ID", "project-id")
	t.Setenv("INFISICAL_ENVIRONMENT", "dev")

	restore := setInfisicalSecretLoaderForTest(t, func(ctx context.Context, cfg InfisicalConfig) (map[string]string, error) {
		return nil, errors.New("auth failed")
	})
	defer restore()

	_, err := New()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "load infisical secrets")
}

func setInfisicalSecretLoaderForTest(t *testing.T, loader infisicalSecretLoader) func() {
	t.Helper()

	previousLoader := loadInfisicalSecrets
	previousEnv := map[string]string{}
	for _, key := range []string{"DB_PASSWORD", "DB_NAME"} {
		previousEnv[key] = os.Getenv(key)
	}

	loadInfisicalSecrets = loader

	return func() {
		loadInfisicalSecrets = previousLoader
		for key, value := range previousEnv {
			if value == "" {
				os.Unsetenv(key)
				continue
			}
			os.Setenv(key, value)
		}
	}
}
