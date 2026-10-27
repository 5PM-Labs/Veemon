// Package handler implements the gRPC/HTTP request handlers.
package handler

import (
	"context"
	"time"

	"github.com/5PM-Labs/veemon/internal/domain"
	pb "github.com/5PM-Labs/veemon/internal/handler/grpc/user"
	"github.com/5PM-Labs/veemon/internal/infra/auth0"
	"github.com/5PM-Labs/veemon/internal/infra/logging"
	"github.com/5PM-Labs/veemon/internal/infra/monitoring/metrics"
	"github.com/5PM-Labs/veemon/internal/pkg/authguard"
	"github.com/5PM-Labs/veemon/internal/pkg/errors"
	"github.com/5PM-Labs/veemon/internal/pkg/middleware"
	"github.com/5PM-Labs/veemon/internal/pkg/token"
	"github.com/5PM-Labs/veemon/internal/usecase/user"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/emptypb"
)

type userHandler struct {
	pb.UnimplementedUserApiServer
	userUC       user.UseCase
	tokenService *token.TokenService
	guard        *authguard.Guard
	auth0        *auth0.Provider
	logger       *logging.Logger
}

// NewUserHandler builds the user-domain handler. auth0 may be nil when the
// deployment has no Auth0 tenant configured; the Auth0Login route then answers
// 400 "not configured" instead of failing startup.
func NewUserHandler(userUC user.UseCase, tokenService *token.TokenService, guard *authguard.Guard, auth0Provider *auth0.Provider, logger *logging.Logger) pb.UserApiServer {
	if logger == nil {
		logger = &logging.Logger{Logger: zap.NewNop()}
	}
	return &userHandler{
		userUC:       userUC,
		tokenService: tokenService,
		guard:        guard,
		auth0:        auth0Provider,
		logger:       logger,
	}
}

// internal logs the underlying cause of a 5xx (which is never sent to clients)
// and returns a sanitized error carrying only a stable code and public message.
// Logging through logger.WithContext(ctx) attaches trace_id/span_id when a
// span is active (both the REST and gRPC paths carry one), so a 500 traced
// here is findable in the OTel backend by trace — this is the single
// error-logging chokepoint for every handler method below.
func (h *userHandler) internal(ctx context.Context, code int, publicMsg string, cause error) error {
	h.logger.WithContext(ctx).Error("internal handler error",
		zap.Int("code", code),
		zap.String("message", publicMsg),
		zap.Error(cause),
	)
	return errors.Internal(code, publicMsg)
}

// Register creates a new user account.
func (h *userHandler) Register(ctx context.Context, req *pb.RegisterReq) (*pb.RegisterRes, error) {
	if err := pb.ValidateRequest(req); err != nil {
		return nil, err
	}

	result, err := h.userUC.Register(ctx, user.RegisterInput{
		Email:    req.Email,
		Password: req.Password,
		Name:     req.Name,
		Phone:    req.Phone,
	})
	if err != nil {
		if err == user.ErrEmailExists {
			return nil, errors.Conflict(40901, "email already registered")
		}
		return nil, h.internal(ctx, 50001, "failed to register user", err)
	}

	if m := metrics.Get(); m != nil {
		m.RecordUserRegistered()
	}

	return &pb.RegisterRes{
		Id:    result.ID,
		Email: result.Email,
		Name:  result.Name,
	}, nil
}

// Login authenticates a user and returns a PASETO token.
func (h *userHandler) Login(ctx context.Context, req *pb.LoginReq) (*pb.LoginRes, error) {
	if err := pb.ValidateRequest(req); err != nil {
		return nil, err
	}

	// Reject early if the account is locked out from repeated failures.
	if h.guard.IsLocked(ctx, req.Email) {
		return nil, errors.TooManyRequests("too many failed login attempts; try again later")
	}

	userEntity, err := h.userUC.Login(ctx, req.Email, req.Password)
	if err != nil {
		switch {
		case err == user.ErrInvalidCreds:
			h.guard.RecordFailure(ctx, req.Email)
			return nil, errors.Unauthorized("invalid email or password")
		case err == user.ErrUserNotActive:
			return nil, errors.Forbidden("account is not active")
		case err == user.ErrLocked:
			return nil, errors.TooManyRequests("too many failed login attempts; try again later")
		default:
			return nil, h.internal(ctx, 50002, "failed to login", err)
		}
	}

	// Successful login clears any accumulated failure/lock state.
	h.guard.Reset(ctx, req.Email)
	_ = h.userUC.RecordLogin(ctx, userEntity.ID)

	if m := metrics.Get(); m != nil {
		m.RecordUserLogin()
	}

	// Generate PASETO token
	accessToken, err := h.tokenService.GenerateToken(userEntity.ID, userEntity.Email, userEntity.CompanyCode)
	if err != nil {
		return nil, h.internal(ctx, 50003, "failed to generate token", err)
	}

	return &pb.LoginRes{
		Token: accessToken,
		User:  toProfile(userEntity),
	}, nil
}

// RefreshToken rotates a valid token: it reloads the user from the database so
// role/status changes take effect, revokes the presented token, and issues a
// new one. Note: because the refresh route itself requires a non-expired token,
// this cannot refresh an already-expired token — a separate long-lived refresh
// token (a proto change) would be needed for that.
func (h *userHandler) RefreshToken(ctx context.Context, req *pb.RefreshTokenReq) (*pb.RefreshTokenRes, error) {
	authCtx, err := requireAuth(ctx)
	if err != nil {
		return nil, err
	}

	// Reload the user so a deactivated or deleted account cannot keep
	// refreshing, and so fresh roles are embedded in the new token.
	profile, err := h.userUC.GetProfile(ctx, authCtx.UserID)
	if err != nil {
		if err == user.ErrNotFound {
			return nil, errors.Unauthorized("user no longer exists")
		}
		return nil, h.internal(ctx, 50010, "failed to refresh token", err)
	}
	if string(profile.Status) != "active" {
		return nil, errors.Forbidden("account is not active")
	}

	// Rotate: revoke the presented token so it cannot be reused.
	if authCtx.TokenID != "" {
		_ = h.guard.Revoke(ctx, authCtx.TokenID, time.Until(authCtx.ExpiresAt))
	}

	newToken, err := h.tokenService.GenerateToken(profile.ID, profile.Email, profile.CompanyCode)
	if err != nil {
		return nil, h.internal(ctx, 50010, "failed to refresh token", err)
	}

	return &pb.RefreshTokenRes{
		Token: newToken,
	}, nil
}

// Auth0Login verifies an Auth0 access token and issues a first-party token.
// The account is provisioned on first login (provider=auth0, subject=sub) and
// granted the default "user" role.
func (h *userHandler) Auth0Login(ctx context.Context, req *pb.Auth0LoginReq) (*pb.LoginRes, error) {
	if err := pb.ValidateRequest(req); err != nil {
		return nil, err
	}
	if h.auth0 == nil {
		return nil, errors.BadRequest(40010, "Auth0 login is not configured on this deployment")
	}

	claims, err := h.auth0.Verify(ctx, req.AccessToken)
	if err != nil {
		return nil, errors.Unauthorized("invalid auth0 token")
	}

	subject := auth0.Subject(claims)
	if subject == "" {
		return nil, errors.Unauthorized("invalid auth0 token")
	}

	userEntity, err := h.userUC.FindByAuth0Subject(ctx, subject)
	if err == user.ErrNotFound {
		userEntity, err = h.userUC.ProvisionAuth0User(ctx, subject, auth0.Email(claims))
	}
	if err != nil {
		return nil, h.internal(ctx, 50011, "failed to resolve auth0 account", err)
	}

	if userEntity.Status != domain.UserStatusActive {
		return nil, errors.Forbidden("account is not active")
	}

	_ = h.userUC.RecordLogin(ctx, userEntity.ID)

	accessToken, err := h.tokenService.GenerateToken(userEntity.ID, userEntity.Email, userEntity.CompanyCode)
	if err != nil {
		return nil, h.internal(ctx, 50003, "failed to generate token", err)
	}

	if m := metrics.Get(); m != nil {
		m.RecordUserLogin()
	}

	return &pb.LoginRes{Token: accessToken, User: toProfile(userEntity)}, nil
}

// GetMe returns the profile of the currently authenticated user.
func (h *userHandler) GetMe(ctx context.Context, req *emptypb.Empty) (*pb.UserProfile, error) {
	authCtx, err := requireAuth(ctx)
	if err != nil {
		return nil, err
	}

	profile, err := h.userUC.GetProfile(ctx, authCtx.UserID)
	if err != nil {
		if err == user.ErrNotFound {
			return nil, errors.NotFound("user not found")
		}
		return nil, h.internal(ctx, 50004, "failed to get profile", err)
	}

	// The caller's own effective permissions come from the auth context (the
	// same set the middleware authorized against), not from a second query.
	out := toProfile(profile)
	out.Permissions = authCtx.Permissions
	return out, nil
}

// Logout revokes the presented token so it can no longer be used, even before
// its natural expiry. Requires a Redis-backed guard; without Redis it degrades
// to a client-side-only logout.
func (h *userHandler) Logout(ctx context.Context, req *emptypb.Empty) (*pb.LogoutRes, error) {
	authCtx, err := requireAuth(ctx)
	if err != nil {
		return nil, err
	}

	if authCtx.TokenID != "" {
		_ = h.guard.Revoke(ctx, authCtx.TokenID, time.Until(authCtx.ExpiresAt))
	}

	return &pb.LogoutRes{
		Message: "successfully logged out",
	}, nil
}

// ListUsers returns a paginated list of all users (admin only).
func (h *userHandler) ListUsers(ctx context.Context, req *pb.ListUsersReq) (*pb.ListUsersRes, error) {
	if err := pb.ValidateRequest(req); err != nil {
		return nil, err
	}

	users, total, err := h.userUC.ListAll(ctx, user.ListInput{
		Page:      int(req.Page),
		Size:      int(req.Size),
		Search:    req.Search,
		SortBy:    req.SortBy,
		SortOrder: req.SortOrder,
	})
	if err != nil {
		return nil, h.internal(ctx, 50005, "failed to list users", err)
	}

	pbUsers := make([]*pb.UserProfile, len(users))
	for i, u := range users {
		pbUsers[i] = toProfile(&u)
	}

	// Echo the page/size actually served: ValidateRequest/defaulting keeps them
	// in bound, and the usecase would clamp anything else (see CurrentPageSize).
	size := int32(user.CurrentPageSize(int(req.Size))) // #nosec G115 -- bounded by MaxPageSize (100)
	totalPages := (total + int64(size) - 1) / int64(size)

	return &pb.ListUsersRes{
		Users: pbUsers,
		Pagination: &pb.Pagination{
			Page:       req.Page,
			Size:       size,
			Total:      total,
			TotalPages: int32(totalPages), // #nosec G115 -- totalPages is bounded by pagination
		},
	}, nil
}

// GetUser returns a single user by ID (admin only).
func (h *userHandler) GetUser(ctx context.Context, req *pb.GetUserReq) (*pb.UserProfile, error) {
	if err := validateUserID(req.Id); err != nil {
		return nil, err
	}

	userEntity, err := h.userUC.GetUser(ctx, req.Id)
	if err != nil {
		if err == user.ErrNotFound {
			return nil, errors.NotFound("user not found")
		}
		return nil, h.internal(ctx, 50006, "failed to get user", err)
	}

	return toProfile(userEntity), nil
}

// UpdateUser updates a user by ID (admin only).
func (h *userHandler) UpdateUser(ctx context.Context, req *pb.UpdateUserReq) (*pb.UserProfile, error) {
	if err := validateUserID(req.Id); err != nil {
		return nil, err
	}
	if err := pb.ValidateRequest(req); err != nil {
		return nil, err
	}

	userEntity, err := h.userUC.UpdateUser(ctx, req.Id, user.UpdateInput{
		Name:   req.Name,
		Phone:  req.Phone,
		Status: req.Status,
	})
	if err != nil {
		if err == user.ErrNotFound {
			return nil, errors.NotFound("user not found")
		}
		return nil, h.internal(ctx, 50007, "failed to update user", err)
	}

	return toProfile(userEntity), nil
}

// DeleteUser soft-deletes a user by ID (admin only).
func (h *userHandler) DeleteUser(ctx context.Context, req *pb.DeleteUserReq) (*pb.DeleteUserRes, error) {
	if err := validateUserID(req.Id); err != nil {
		return nil, err
	}

	err := h.userUC.DeleteUser(ctx, req.Id)
	if err != nil {
		if err == user.ErrNotFound {
			return nil, errors.NotFound("user not found")
		}
		return nil, h.internal(ctx, 50008, "failed to delete user", err)
	}

	return &pb.DeleteUserRes{
		Message: "user deleted successfully",
	}, nil
}

// requireAuth returns the authenticated user or a 401 when the request did not
// pass through an auth middleware. Never default to "allow" on a missing
// context: a route accidentally registered without auth must fail closed.
func requireAuth(ctx context.Context) (*middleware.AuthContext, error) {
	a, ok := middleware.AuthFromContext(ctx)
	if !ok || a == nil {
		return nil, errors.Unauthorized("authentication required")
	}
	return a, nil
}

// validateUserID rejects malformed identifiers before they reach the database,
// where an invalid UUID would surface as a 500 instead of a 400.
func validateUserID(id string) error {
	if _, err := uuid.Parse(id); err != nil {
		return errors.BadRequest(40002, "invalid user id")
	}
	return nil
}

// toProfile maps a domain user to its wire representation. Roles and
// permissions are enough for clients to render UI affordances; permission
// decisions stay server-side.
func toProfile(u *domain.User) *pb.UserProfile {
	p := &pb.UserProfile{
		Id:            u.ID,
		Email:         u.Email,
		Name:          u.Name,
		Phone:         u.Phone,
		Status:        string(u.Status),
		CreatedAt:     u.CreatedAt.Format(time.RFC3339),
		AvatarUrl:     u.AvatarURL,
		Locale:        u.Locale,
		Timezone:      u.Timezone,
		EmailVerified: u.EmailVerified,
		MfaEnabled:    u.MFAEnabled,
		AuthProvider:  string(u.AuthProvider),
		CompanyCode:   u.CompanyCode,
		Roles:         u.Roles,
	}
	if u.LastLoginAt != nil {
		p.LastLoginAt = u.LastLoginAt.Format(time.RFC3339)
	}
	return p
}
