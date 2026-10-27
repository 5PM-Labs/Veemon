package middleware

import (
	"context"
	"strings"

	"github.com/5PM-Labs/veemon/internal/pkg/errors"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func GRPCAuthInterceptor(validator TokenValidator, authConfig map[string]AuthConfig) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		config, ok := authConfig[info.FullMethod]
		if !ok {
			// Fail closed: a method with no explicit auth policy is denied
			// rather than served without authentication.
			return nil, errors.Unauthorized("no auth policy configured for method").GRPCStatus().Err()
		}
		if !config.NeedAuth {
			return handler(ctx, req)
		}

		token, err := extractBearerToken(ctx)
		if err != nil {
			return nil, errors.Unauthorized("missing authorization").GRPCStatus().Err()
		}

		authCtx, err := validator(token)
		if err != nil {
			return nil, errors.Unauthorized("invalid token").GRPCStatus().Err()
		}

		if !authorized(authCtx, config) {
			return nil, errors.Forbidden("insufficient permissions").GRPCStatus().Err()
		}

		authCtx.Token = token
		return handler(WithAuthContext(ctx, authCtx), req)
	}
}

func GetGRPCAuthContext(ctx context.Context) (*AuthContext, bool) {
	return AuthFromContext(ctx)
}

func extractBearerToken(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", errors.Unauthorized("missing metadata")
	}

	authHeaders := md.Get("authorization")
	if len(authHeaders) == 0 {
		return "", errors.Unauthorized("missing authorization header")
	}

	parts := strings.SplitN(authHeaders[0], " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" || parts[1] == "" {
		return "", errors.Unauthorized("invalid authorization format")
	}

	return parts[1], nil
}
