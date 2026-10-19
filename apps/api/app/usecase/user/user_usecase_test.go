package user

import (
	"context"
	"testing"

	"veemon/entity"
	"veemon/repository/user_repository"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// MockUserRepository is a mock implementation of user_repository.Repository
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Create(ctx context.Context, user *entity.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) FindByID(ctx context.Context, id string) (*entity.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.User), args.Error(1)
}

func (m *MockUserRepository) FindByEmail(ctx context.Context, email string) (*entity.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.User), args.Error(1)
}

func (m *MockUserRepository) FindAll(ctx context.Context, params user_repository.ListParams) ([]entity.User, int64, error) {
	args := m.Called(ctx, params)
	return args.Get(0).([]entity.User), args.Get(1).(int64), args.Error(2)
}

func (m *MockUserRepository) UpdateFields(ctx context.Context, id string, fields map[string]interface{}) (*entity.User, error) {
	args := m.Called(ctx, id, fields)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.User), args.Error(1)
}

func (m *MockUserRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func TestRegister_Success(t *testing.T) {
	mockRepo := new(MockUserRepository)
	uc := NewUseCase(mockRepo)
	ctx := context.Background()

	input := RegisterInput{
		Email:    "test@example.com",
		Password: "password123",
		Name:     "Test User",
		Phone:    "081234567890",
	}

	// Mock FindByEmail returns not found
	mockRepo.On("FindByEmail", ctx, input.Email).Return(nil, gorm.ErrRecordNotFound)

	// Mock Create succeeds
	mockRepo.On("Create", ctx, mock.AnythingOfType("*entity.User")).Return(nil)

	result, err := uc.Register(ctx, input)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, input.Email, result.Email)
	assert.Equal(t, input.Name, result.Name)
	mockRepo.AssertExpectations(t)
}

func TestRegister_EmailExists(t *testing.T) {
	mockRepo := new(MockUserRepository)
	uc := NewUseCase(mockRepo)
	ctx := context.Background()

	input := RegisterInput{
		Email:    "existing@example.com",
		Password: "password123",
		Name:     "Test User",
	}

	existingUser := &entity.User{
		ID:    "existing-id",
		Email: input.Email,
	}

	// Mock FindByEmail returns existing user
	mockRepo.On("FindByEmail", ctx, input.Email).Return(existingUser, nil)

	result, err := uc.Register(ctx, input)

	assert.Error(t, err)
	assert.Equal(t, ErrEmailExists, err)
	assert.Nil(t, result)
	mockRepo.AssertExpectations(t)
}

func TestRegister_DuplicateKeyRace(t *testing.T) {
	mockRepo := new(MockUserRepository)
	uc := NewUseCase(mockRepo)
	ctx := context.Background()

	input := RegisterInput{Email: "race@example.com", Password: "Password123", Name: "Race"}

	// FindByEmail says the user does not exist (both concurrent requests pass),
	// but the unique index rejects the insert with a duplicate-key error.
	mockRepo.On("FindByEmail", ctx, input.Email).Return(nil, gorm.ErrRecordNotFound)
	mockRepo.On("Create", ctx, mock.AnythingOfType("*entity.User")).Return(gorm.ErrDuplicatedKey)

	_, err := uc.Register(ctx, input)
	assert.ErrorIs(t, err, ErrEmailExists)
	mockRepo.AssertExpectations(t)
}

func TestLogin_InactiveUserRejected(t *testing.T) {
	mockRepo := new(MockUserRepository)
	uc := NewUseCase(mockRepo)
	ctx := context.Background()

	hash, err := bcrypt.GenerateFromPassword([]byte("Password123"), bcrypt.DefaultCost)
	assert.NoError(t, err)

	mockRepo.On("FindByEmail", ctx, "inactive@example.com").Return(&entity.User{
		ID:       "u1",
		Email:    "inactive@example.com",
		Password: string(hash),
		Status:   entity.UserStatusInactive,
	}, nil)

	_, err = uc.Login(ctx, "inactive@example.com", "Password123")
	assert.ErrorIs(t, err, ErrUserNotActive)
	mockRepo.AssertExpectations(t)
}

func TestGetProfile_Success(t *testing.T) {
	mockRepo := new(MockUserRepository)
	uc := NewUseCase(mockRepo)
	ctx := context.Background()

	userID := "user-123"
	expectedUser := &entity.User{
		ID:     userID,
		Email:  "test@example.com",
		Name:   "Test User",
		Status: entity.UserStatusActive,
	}

	mockRepo.On("FindByID", ctx, userID).Return(expectedUser, nil)

	result, err := uc.GetProfile(ctx, userID)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, expectedUser.ID, result.ID)
	assert.Equal(t, expectedUser.Email, result.Email)
	mockRepo.AssertExpectations(t)
}

func TestGetProfile_NotFound(t *testing.T) {
	mockRepo := new(MockUserRepository)
	uc := NewUseCase(mockRepo)
	ctx := context.Background()

	userID := "non-existent"

	mockRepo.On("FindByID", ctx, userID).Return(nil, gorm.ErrRecordNotFound)

	result, err := uc.GetProfile(ctx, userID)

	assert.Error(t, err)
	assert.Equal(t, ErrNotFound, err)
	assert.Nil(t, result)
	mockRepo.AssertExpectations(t)
}

func TestListAll_Success(t *testing.T) {
	mockRepo := new(MockUserRepository)
	uc := NewUseCase(mockRepo)
	ctx := context.Background()

	input := ListInput{
		Page:      1,
		Size:      10,
		Search:    "",
		SortBy:    "created_at",
		SortOrder: "desc",
	}

	expectedUsers := []entity.User{
		{ID: "user-1", Email: "user1@example.com", Name: "User 1"},
		{ID: "user-2", Email: "user2@example.com", Name: "User 2"},
	}
	expectedTotal := int64(2)

	mockRepo.On("FindAll", ctx, mock.AnythingOfType("user_repository.ListParams")).Return(expectedUsers, expectedTotal, nil)

	users, total, err := uc.ListAll(ctx, input)

	assert.NoError(t, err)
	assert.Equal(t, expectedTotal, total)
	assert.Len(t, users, 2)
	mockRepo.AssertExpectations(t)
}

// A zero page/size (the proto3 default when the client omits the field) must
// become page 1 / DefaultPageSize. Passing 0 through produced LIMIT 0 and a
// division by zero when computing total pages.
func TestListAll_DefaultsAndClampsPagination(t *testing.T) {
	tests := []struct {
		name     string
		page     int
		size     int
		wantPage int
		wantSize int
	}{
		{"omitted", 0, 0, 1, DefaultPageSize},
		{"negative", -3, -1, 1, DefaultPageSize},
		{"oversized", 2, 10_000, 2, MaxPageSize},
		{"valid", 3, 25, 3, 25},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockUserRepository)
			uc := NewUseCase(mockRepo)
			ctx := context.Background()

			mockRepo.On("FindAll", ctx, mock.MatchedBy(func(p user_repository.ListParams) bool {
				return p.Page == tt.wantPage && p.Size == tt.wantSize
			})).Return([]entity.User{}, int64(0), nil)

			_, _, err := uc.ListAll(ctx, ListInput{Page: tt.page, Size: tt.size})

			assert.NoError(t, err)
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestUpdateUser_Success(t *testing.T) {
	mockRepo := new(MockUserRepository)
	uc := NewUseCase(mockRepo)
	ctx := context.Background()

	userID := "user-123"
	updatedUser := &entity.User{
		ID:     userID,
		Email:  "test@example.com",
		Name:   "New Name",
		Phone:  "089876543210",
		Status: entity.UserStatusActive,
	}

	mockRepo.On("UpdateFields", ctx, userID, mock.AnythingOfType("map[string]interface {}")).Return(updatedUser, nil)

	result, err := uc.UpdateUser(ctx, userID, UpdateInput{
		Name:  "New Name",
		Phone: "089876543210",
	})

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "New Name", result.Name)
	assert.Equal(t, "089876543210", result.Phone)
	mockRepo.AssertExpectations(t)
}

func TestDeleteUser_Success(t *testing.T) {
	mockRepo := new(MockUserRepository)
	uc := NewUseCase(mockRepo)
	ctx := context.Background()

	userID := "user-123"
	existingUser := &entity.User{ID: userID}

	mockRepo.On("FindByID", ctx, userID).Return(existingUser, nil)
	mockRepo.On("Delete", ctx, userID).Return(nil)

	err := uc.DeleteUser(ctx, userID)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestDeleteUser_NotFound(t *testing.T) {
	mockRepo := new(MockUserRepository)
	uc := NewUseCase(mockRepo)
	ctx := context.Background()

	userID := "non-existent"

	mockRepo.On("FindByID", ctx, userID).Return(nil, gorm.ErrRecordNotFound)

	err := uc.DeleteUser(ctx, userID)

	assert.Error(t, err)
	assert.Equal(t, ErrNotFound, err)
	mockRepo.AssertExpectations(t)
}
