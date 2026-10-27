package middleware

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestGRPCAuthInterceptor_AllowsPublicMethod(t *testing.T) {
	interceptor := GRPCAuthInterceptor(nil, map[string]AuthConfig{
		"/user.UserApi/Login": {NeedAuth: false},
	})

	called := false
	resp, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{
		FullMethod: "/user.UserApi/Login",
	}, func(ctx context.Context, req interface{}) (interface{}, error) {
		called = true
		return "ok", nil
	})

	require.NoError(t, err)
	assert.True(t, called)
	assert.Equal(t, "ok", resp)
}

func TestGRPCAuthInterceptor_RejectsMissingAuthorization(t *testing.T) {
	interceptor := GRPCAuthInterceptor(func(token string) (*AuthContext, error) {
		t.Fatal("validator should not be called without an authorization header")
		return nil, nil
	}, map[string]AuthConfig{
		"/user.UserApi/ListUsers": {NeedAuth: true},
	})

	called := false
	_, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{
		FullMethod: "/user.UserApi/ListUsers",
	}, func(ctx context.Context, req interface{}) (interface{}, error) {
		called = true
		return "ok", nil
	})

	require.Error(t, err)
	assert.False(t, called)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestGRPCAuthInterceptor_RejectsInvalidToken(t *testing.T) {
	interceptor := GRPCAuthInterceptor(func(token string) (*AuthContext, error) {
		assert.Equal(t, "bad-token", token)
		return nil, errors.New("invalid token")
	}, map[string]AuthConfig{
		"/user.UserApi/GetMe": {NeedAuth: true},
	})

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer bad-token"))
	_, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{
		FullMethod: "/user.UserApi/GetMe",
	}, func(ctx context.Context, req interface{}) (interface{}, error) {
		t.Fatal("handler should not run with invalid token")
		return nil, nil
	})

	require.Error(t, err)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestGRPCAuthInterceptor_RejectsMissingRole(t *testing.T) {
	interceptor := GRPCAuthInterceptor(func(token string) (*AuthContext, error) {
		return &AuthContext{UserID: "user-1", Roles: []string{"user"}}, nil
	}, map[string]AuthConfig{
		"/user.UserApi/ListUsers": {NeedAuth: true, AllowedRoles: []string{"admin", "superadmin"}},
	})

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer valid-token"))
	_, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{
		FullMethod: "/user.UserApi/ListUsers",
	}, func(ctx context.Context, req interface{}) (interface{}, error) {
		t.Fatal("handler should not run with insufficient role")
		return nil, nil
	})

	require.Error(t, err)
	assert.Equal(t, codes.PermissionDenied, status.Code(err))
}

func TestGRPCAuthInterceptor_AddsAuthContextForProtectedMethod(t *testing.T) {
	expectedAuth := &AuthContext{
		UserID:      "admin-1",
		Email:       "admin@example.com",
		Roles:       []string{"admin"},
		CompanyCode: "COMP001",
	}

	interceptor := GRPCAuthInterceptor(func(token string) (*AuthContext, error) {
		assert.Equal(t, "valid-token", token)
		expectedAuth.Token = token
		return expectedAuth, nil
	}, map[string]AuthConfig{
		"/user.UserApi/ListUsers": {NeedAuth: true, AllowedRoles: []string{"admin", "superadmin"}},
	})

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer valid-token"))
	resp, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{
		FullMethod: "/user.UserApi/ListUsers",
	}, func(ctx context.Context, req interface{}) (interface{}, error) {
		auth, ok := GetGRPCAuthContext(ctx)
		require.True(t, ok)
		assert.Equal(t, expectedAuth, auth)
		return "ok", nil
	})

	require.NoError(t, err)
	assert.Equal(t, "ok", resp)
}
