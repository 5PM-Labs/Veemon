package validation

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type TestUser struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8,max=128"`
	Name     string `json:"name" validate:"required,min=2,max=100"`
	Phone    string `json:"phone" validate:"omitempty,phone"`
	Age      int    `json:"age" validate:"omitempty,gte=0,lte=150"`
}

func TestValidate_Success(t *testing.T) {
	user := TestUser{
		Email:    "test@example.com",
		Password: "password123",
		Name:     "Test User",
		Phone:    "081234567890",
		Age:      25,
	}

	err := Validate(user)
	assert.NoError(t, err)
}

func TestValidate_RequiredFields(t *testing.T) {
	user := TestUser{
		Email: "",
		Name:  "",
	}

	err := Validate(user)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "email is required")
}

func TestValidate_InvalidEmail(t *testing.T) {
	user := TestUser{
		Email:    "not-an-email",
		Password: "password123",
		Name:     "Test User",
	}

	err := Validate(user)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "email must be a valid email")
}

func TestValidate_PasswordTooShort(t *testing.T) {
	user := TestUser{
		Email:    "test@example.com",
		Password: "short",
		Name:     "Test User",
	}

	err := Validate(user)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "password must be at least 8 characters")
}

func TestValidate_PhoneValid(t *testing.T) {
	tests := []struct {
		name  string
		phone string
		valid bool
	}{
		{"Indonesian format", "081234567890", true},
		{"International format", "+6281234567890", true},
		{"With dashes", "0812-3456-7890", true},
		{"Too short", "12345", false},
		{"Invalid chars", "abc12345678", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := TestUser{
				Email:    "test@example.com",
				Password: "password123",
				Name:     "Test User",
				Phone:    tt.phone,
			}

			err := Validate(user)
			if tt.valid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestValidateVar(t *testing.T) {
	// Test email validation
	err := ValidateVar("test@example.com", "required,email")
	assert.NoError(t, err)

	err = ValidateVar("not-an-email", "required,email")
	assert.Error(t, err)

	// Test min length
	err = ValidateVar("ab", "min=3")
	assert.Error(t, err)

	err = ValidateVar("abc", "min=3")
	assert.NoError(t, err)
}

type PasswordTest struct {
	Password string `json:"password" validate:"password"`
}

func TestCustomValidator_Password(t *testing.T) {
	tests := []struct {
		name     string
		password string
		valid    bool
	}{
		{"Valid password", "Password1", true},
		{"Missing uppercase", "password1", false},
		{"Missing lowercase", "PASSWORD1", false},
		{"Missing number", "Password", false},
		{"Too short", "Pass1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := PasswordTest{Password: tt.password}
			err := Validate(p)
			if tt.valid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

type NIKTest struct {
	NIK string `json:"nik" validate:"omitempty,nik"`
}

func TestCustomValidator_NIK(t *testing.T) {
	tests := []struct {
		name  string
		nik   string
		valid bool
	}{
		{"Valid NIK", "1234567890123456", true},
		{"Too short", "123456789012345", false},
		{"Too long", "12345678901234567", false},
		{"Contains letters", "123456789012345a", false},
		{"Empty (optional)", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := NIKTest{NIK: tt.nik}
			err := Validate(n)
			if tt.valid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}
