// Package user contains user business logic (the usecase layer).
package user

import (
	"context"
	"errors"
	"time"

	"github.com/5PM-Labs/veemon/internal/domain"
	"github.com/5PM-Labs/veemon/internal/repository/user"

	"golang.org/x/crypto/bcrypt"
)

// ErrNotFound is the not-found sentinel surfaced to handlers. It aliases the
// repository-facing domain sentinel, so errors.Is matches either way.
var ErrNotFound = domain.ErrNotFound

var (
	ErrEmailExists   = errors.New("email already registered")
	ErrInvalidCreds  = errors.New("invalid credentials")
	ErrUserNotActive = errors.New("user account is not active")
	// ErrLocked is returned while a DB-side lockout is still in effect.
	ErrLocked = errors.New("account temporarily locked")
)

// Lockout policy for the durable (DB) counters. The Redis guard in the handler
// is the fast path across replicas; these columns survive a Redis flush.
const (
	maxFailedLogins = 5
	lockoutDuration = 15 * time.Minute
)

// dummyPasswordHash is a valid bcrypt hash of an arbitrary value, compared
// against on the user-not-found login path to equalize timing and mitigate
// user enumeration. Generated once at package init at the same cost as real
// password hashing.
var dummyPasswordHash []byte

func init() {
	h, err := bcrypt.GenerateFromPassword([]byte("timing-equalization-placeholder"), bcrypt.DefaultCost)
	if err != nil {
		panic("failed to initialize dummy password hash: " + err.Error())
	}
	dummyPasswordHash = h
}

type UseCase interface {
	Register(ctx context.Context, input RegisterInput) (*RegisterOutput, error)
	Login(ctx context.Context, email, password string) (*domain.User, error)
	GetProfile(ctx context.Context, userID string) (*domain.User, error)
	ListAll(ctx context.Context, input ListInput) ([]domain.User, int64, error)
	GetUser(ctx context.Context, userID string) (*domain.User, error)
	UpdateUser(ctx context.Context, userID string, input UpdateInput) (*domain.User, error)
	DeleteUser(ctx context.Context, userID string) error
	// Federated (Auth0) identity + login tracking.
	FindByAuth0Subject(ctx context.Context, subject string) (*domain.User, error)
	ProvisionAuth0User(ctx context.Context, subject, email string) (*domain.User, error)
	RecordLogin(ctx context.Context, userID string) error
}

type RegisterInput struct {
	Email    string
	Password string
	Name     string
	Phone    string
}

type RegisterOutput struct {
	ID    string
	Email string
	Name  string
}

// FindByAuth0Subject resolves a federated user by their Auth0 subject.
func (uc *useCase) FindByAuth0Subject(ctx context.Context, subject string) (*domain.User, error) {
	u, err := uc.userRepo.FindByAuth0Subject(ctx, subject)
	if errors.Is(err, domain.ErrNotFound) {
		return nil, ErrNotFound
	}
	return u, err
}

// ProvisionAuth0User creates the first-party account for a federated identity.
// Email comes from the verified Auth0 claim; accounts start active because the
// email was verified by the identity provider. PasswordHash stays empty.
func (uc *useCase) ProvisionAuth0User(ctx context.Context, subject, email string) (*domain.User, error) {
	u := &domain.User{
		Email:         email,
		Name:          email, // display name is updated by the user later
		Status:        domain.UserStatusActive,
		EmailVerified: true,
		AuthProvider:  domain.AuthProviderAuth0,
		Auth0Subject:  subject,
		Roles:         []string{domain.RoleUser},
	}
	if err := uc.userRepo.Create(ctx, u); err != nil {
		if errors.Is(err, domain.ErrDuplicateEntry) {
			// Lost a race with a concurrent first login: fetch the winner.
			return uc.FindByAuth0Subject(ctx, subject)
		}
		return nil, err
	}
	return u, nil
}

// RecordLogin forwards the last-login stamp to the repository.
func (uc *useCase) RecordLogin(ctx context.Context, userID string) error {
	return uc.userRepo.RecordLogin(ctx, userID)
}

// Pagination bounds for ListAll. proto3 cannot distinguish "page omitted" from
// page=0, so a zero value means "use the default" rather than "no rows".
const (
	DefaultPageSize = 10
	MaxPageSize     = 100
)

type ListInput struct {
	Page      int
	Size      int
	Search    string
	SortBy    string
	SortOrder string
}

type UpdateInput struct {
	Name   string
	Phone  string
	Status string
}

type useCase struct {
	userRepo user_repository.Repository
}

func NewUseCase(userRepo user_repository.Repository) UseCase {
	return &useCase{userRepo: userRepo}
}

func (uc *useCase) Register(ctx context.Context, input RegisterInput) (*RegisterOutput, error) {
	// Check if email exists
	existing, err := uc.userRepo.FindByEmail(ctx, input.Email)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return nil, err
	}
	if existing != nil {
		return nil, ErrEmailExists
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &domain.User{
		Email:         input.Email,
		PasswordHash:  string(hashedPassword),
		Name:          input.Name,
		Phone:         input.Phone,
		Status:        domain.UserStatusPending, // pending until email verified
		AuthProvider:  domain.AuthProviderLocal,
		EmailVerified: false,
		Roles:         []string{domain.RoleUser},
	}
	if err := uc.userRepo.Create(ctx, user); err != nil {
		// Closes the check-then-insert race: two concurrent registrations pass
		// the FindByEmail check, but the unique index rejects the second insert.
		if errors.Is(err, domain.ErrDuplicateEntry) {
			return nil, ErrEmailExists
		}
		return nil, err
	}

	return &RegisterOutput{
		ID:    user.ID,
		Email: user.Email,
		Name:  user.Name,
	}, nil
}

func (uc *useCase) Login(ctx context.Context, email, password string) (*domain.User, error) {
	user, err := uc.userRepo.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			// Perform a dummy hash comparison so the not-found path takes
			// roughly the same time as the wrong-password path, mitigating
			// user enumeration via response timing.
			_ = bcrypt.CompareHashAndPassword(dummyPasswordHash, []byte(password))
			return nil, ErrInvalidCreds
		}
		return nil, err
	}

	// Durable lockout, checked before the password comparison: a locked
	// account is rejected even with the right password.
	if user.LockedUntil != nil && user.LockedUntil.After(time.Now()) {
		return nil, ErrLocked
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		// Count the failure at the source outliving Redis. Best-effort: a
		// counter write must not turn a bad password into a 500.
		_, _, _ = uc.userRepo.RecordFailedLogin(context.WithoutCancel(ctx), user.ID, maxFailedLogins, lockoutDuration)
		return nil, ErrInvalidCreds
	}

	// Only active accounts may authenticate. Deactivated (inactive) or
	// not-yet-activated (pending) users are rejected even with valid credentials.
	if user.Status != domain.UserStatusActive {
		return nil, ErrUserNotActive
	}

	return user, nil
}

func (uc *useCase) GetProfile(ctx context.Context, userID string) (*domain.User, error) {
	user, err := uc.userRepo.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return user, nil
}

func (uc *useCase) ListAll(ctx context.Context, input ListInput) ([]domain.User, int64, error) {
	return uc.userRepo.FindAll(ctx, user_repository.ListParams{
		Page:      CurrentPage(input.Page),
		Size:      CurrentPageSize(input.Size),
		Search:    input.Search,
		SortBy:    input.SortBy,
		SortOrder: input.SortOrder,
	})
}

// CurrentPage maps 0 (field omitted) to page 1 and floors negatives at 1.
func CurrentPage(page int) int {
	if page < 1 {
		return 1
	}
	return page
}

// CurrentPageSize maps 0 (field omitted) to DefaultPageSize and caps oversized
// requests at MaxPageSize.
func CurrentPageSize(size int) int {
	switch {
	case size < 1:
		return DefaultPageSize
	case size > MaxPageSize:
		return MaxPageSize
	default:
		return size
	}
}

func (uc *useCase) GetUser(ctx context.Context, userID string) (*domain.User, error) {
	user, err := uc.userRepo.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return user, nil
}

func (uc *useCase) UpdateUser(ctx context.Context, userID string, input UpdateInput) (*domain.User, error) {
	// Build a column-scoped update so only changed fields are written; this
	// avoids the lost-update hazard of read-modify-write with Save.
	fields := map[string]interface{}{}
	if input.Name != "" {
		fields["name"] = input.Name
	}
	if input.Phone != "" {
		fields["phone"] = input.Phone
	}
	if input.Status != "" {
		fields["status"] = input.Status
	}

	// No changes requested: return the current record (or not-found).
	if len(fields) == 0 {
		return uc.GetUser(ctx, userID)
	}

	user, err := uc.userRepo.UpdateFields(ctx, userID, fields)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return user, nil
}

func (uc *useCase) DeleteUser(ctx context.Context, userID string) error {
	_, err := uc.userRepo.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return ErrNotFound
		}
		return err
	}

	return uc.userRepo.Delete(ctx, userID)
}
