package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunAuthenticatedFlowUsesPasetoTokenAndListsUsers(t *testing.T) {
	var seen []string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = append(seen, r.Method+" "+r.URL.RequestURI()+" "+r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/api/v1/auth/register":
			assert.Equal(t, http.MethodPost, r.Method)
			writeJSON(t, w, http.StatusCreated, map[string]any{
				"success": true,
				"data": map[string]any{
					"id":    "user-1",
					"email": "admin@example.com",
					"name":  "Demo Admin",
				},
			})
		case "/api/v1/auth/login":
			assert.Equal(t, http.MethodPost, r.Method)
			writeJSON(t, w, http.StatusOK, map[string]any{
				"success": true,
				"data": map[string]any{
					"token": "v4.local.initial",
					"user": map[string]any{
						"id":        "user-1",
						"email":     "admin@example.com",
						"name":      "Demo Admin",
						"phone":     "081234567890",
						"status":    "active",
						"createdAt": "2026-06-07T10:00:00Z",
					},
				},
			})
		case "/api/v1/auth/me":
			assert.Equal(t, "Bearer v4.local.initial", r.Header.Get("Authorization"))
			writeJSON(t, w, http.StatusOK, map[string]any{
				"success": true,
				"data": map[string]any{
					"id":        "user-1",
					"email":     "admin@example.com",
					"name":      "Demo Admin",
					"phone":     "081234567890",
					"status":    "active",
					"createdAt": "2026-06-07T10:00:00Z",
				},
			})
		case "/api/v1/auth/refresh":
			assert.Equal(t, "Bearer v4.local.initial", r.Header.Get("Authorization"))
			writeJSON(t, w, http.StatusOK, map[string]any{
				"success": true,
				"data": map[string]any{
					"token": "v4.local.refreshed",
				},
			})
		case "/api/v1/users/":
			assert.Equal(t, "Bearer v4.local.refreshed", r.Header.Get("Authorization"))
			assert.Equal(t, "1", r.URL.Query().Get("page"))
			assert.Equal(t, "10", r.URL.Query().Get("size"))
			writeJSON(t, w, http.StatusOK, map[string]any{
				"success": true,
				"data": []map[string]any{
					{
						"id":        "user-1",
						"email":     "admin@example.com",
						"name":      "Demo Admin",
						"phone":     "081234567890",
						"status":    "active",
						"createdAt": "2026-06-07T10:00:00Z",
					},
				},
				"meta": map[string]any{
					"page":       1,
					"size":       10,
					"total":      1,
					"totalPages": 1,
				},
			})
		case "/api/v1/auth/logout":
			assert.Equal(t, "Bearer v4.local.refreshed", r.Header.Get("Authorization"))
			writeJSON(t, w, http.StatusOK, map[string]any{
				"success": true,
				"data": map[string]any{
					"message": "successfully logged out",
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewAPIClient(server.URL)
	result, err := RunAuthenticatedFlow(context.Background(), client, FlowInput{
		Email:    "admin@example.com",
		Password: "Password123",
		Name:     "Demo Admin",
		Phone:    "081234567890",
		Register: true,
	})

	require.NoError(t, err)
	assert.Equal(t, "user-1", result.Registered.ID)
	assert.Equal(t, "v4.local.initial", result.LoginToken)
	assert.Equal(t, "v4.local.refreshed", result.RefreshedToken)
	assert.Equal(t, "admin@example.com", result.CurrentUser.Email)
	assert.Len(t, result.Users, 1)
	assert.Equal(t, int64(1), result.Pagination.Total)
	assert.Contains(t, seen, "GET /api/v1/users/?page=1&size=10&sortBy=created_at&sortOrder=desc Bearer v4.local.refreshed")
}

func writeJSON(t *testing.T, w http.ResponseWriter, status int, value any) {
	t.Helper()

	w.WriteHeader(status)
	require.NoError(t, json.NewEncoder(w).Encode(value))
}
