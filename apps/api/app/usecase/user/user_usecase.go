// Package user contains user business logic (the usecase layer).
package user

import (
	"context"
	"errors"

	"veemon/entity"
	"veemon/repository/user_repository"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var (
	ErrEmailExists   = errors.New("email already registered")
	ErrNotFound      = errors.New("user not found")
	ErrInvalidCreds  = errors.New("invalid credentials")
	ErrUserNotActive = errors.New("user account is not active")
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
	Login(ctx context.Context, email, password string) (*entity.User, error)
	GetProfile(ctx context.Context, userID string) (*entity.User, error)
	ListAll(ctx context.Context, input ListInput) ([]entity.User, int64, error)
	GetUser(ctx context.Context, userID string) (*entity.User, error)
	UpdateUser(ctx context.Context, userID string, input UpdateInput) (*entity.User, error)
	DeleteUser(ctx context.Context, userID string) error
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
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
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

	user := &entity.User{
		Email:    input.Email,
		Password: string(hashedPassword),
		Name:     input.Name,
		Phone:    input.Phone,
		Status:   entity.UserStatusActive,
	}

	if err := uc.userRepo.Create(ctx, user); err != nil {
		// Closes the check-then-insert race: two concurrent registrations pass
		// the FindByEmail check, but the unique index rejects the second insert.
		if errors.Is(err, gorm.ErrDuplicatedKey) {
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

func (uc *useCase) Login(ctx context.Context, email, password string) (*entity.User, error) {
	user, err := uc.userRepo.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Perform a dummy hash comparison so the not-found path takes
			// roughly the same time as the wrong-password path, mitigating
			// user enumeration via response timing.
			_ = bcrypt.CompareHashAndPassword(dummyPasswordHash, []byte(password))
			return nil, ErrInvalidCreds
		}
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, ErrInvalidCreds
	}

	// Only active accounts may authenticate. Deactivated (inactive) or
	// not-yet-activated (pending) users are rejected even with valid credentials.
	if user.Status != entity.UserStatusActive {
		return nil, ErrUserNotActive
	}

	return user, nil
}

func (uc *useCase) GetProfile(ctx context.Context, userID string) (*entity.User, error) {
	user, err := uc.userRepo.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return user, nil
}

func (uc *useCase) ListAll(ctx context.Context, input ListInput) ([]entity.User, int64, error) {
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

func (uc *useCase) GetUser(ctx context.Context, userID string) (*entity.User, error) {
	user, err := uc.userRepo.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return user, nil
}

func (uc *useCase) UpdateUser(ctx context.Context, userID string, input UpdateInput) (*entity.User, error) {
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
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return user, nil
}

func (uc *useCase) DeleteUser(ctx context.Context, userID string) error {
	_, err := uc.userRepo.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		return err
	}

	return uc.userRepo.Delete(ctx, userID)
}
