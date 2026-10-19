// Package validation wraps go-playground/validator with custom rules.
package validation

import (
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/go-playground/validator/v10"

	"veemon/internal/pkg/errors"
)

var (
	validate *validator.Validate
	once     sync.Once
)

// GetValidator returns the singleton validator instance
func GetValidator() *validator.Validate {
	once.Do(func() {
		validate = validator.New()

		// Use JSON tag names for field names in errors
		validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
			name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
			if name == "-" {
				return fld.Name
			}
			return name
		})

		// Register custom validators
		registerCustomValidators(validate)
	})
	return validate
}

// Validate validates a struct
func Validate(s interface{}) error {
	err := GetValidator().Struct(s)
	if err == nil {
		return nil
	}

	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		return formatValidationErrors(validationErrors)
	}
	return err
}

// ValidateVar validates a single variable
func ValidateVar(field interface{}, tag string) error {
	return GetValidator().Var(field, tag)
}

// ValidateVarWithValue validates a variable against another variable
func ValidateVarWithValue(field interface{}, other interface{}, tag string) error {
	return GetValidator().VarWithValue(field, other, tag)
}

func formatValidationErrors(errs validator.ValidationErrors) error {
	var messages []string
	for _, e := range errs {
		messages = append(messages, formatFieldError(e))
	}
	return errors.ValidationError(strings.Join(messages, "; "))
}

func formatFieldError(e validator.FieldError) string {
	field := e.Field()

	switch e.Tag() {
	case "required":
		return fmt.Sprintf("%s is required", field)
	case "email":
		return fmt.Sprintf("%s must be a valid email", field)
	case "min":
		if e.Kind() == reflect.String {
			return fmt.Sprintf("%s must be at least %s characters", field, e.Param())
		}
		return fmt.Sprintf("%s must be at least %s", field, e.Param())
	case "max":
		if e.Kind() == reflect.String {
			return fmt.Sprintf("%s must be at most %s characters", field, e.Param())
		}
		return fmt.Sprintf("%s must be at most %s", field, e.Param())
	case "len":
		return fmt.Sprintf("%s must be exactly %s characters", field, e.Param())
	case "eq":
		return fmt.Sprintf("%s must be equal to %s", field, e.Param())
	case "ne":
		return fmt.Sprintf("%s must not be equal to %s", field, e.Param())
	case "gt":
		return fmt.Sprintf("%s must be greater than %s", field, e.Param())
	case "gte":
		return fmt.Sprintf("%s must be greater than or equal to %s", field, e.Param())
	case "lt":
		return fmt.Sprintf("%s must be less than %s", field, e.Param())
	case "lte":
		return fmt.Sprintf("%s must be less than or equal to %s", field, e.Param())
	case "oneof":
		return fmt.Sprintf("%s must be one of: %s", field, e.Param())
	case "url":
		return fmt.Sprintf("%s must be a valid URL", field)
	case "uuid":
		return fmt.Sprintf("%s must be a valid UUID", field)
	case "uuid4":
		return fmt.Sprintf("%s must be a valid UUID v4", field)
	case "alphanum":
		return fmt.Sprintf("%s must contain only alphanumeric characters", field)
	case "alpha":
		return fmt.Sprintf("%s must contain only alphabetic characters", field)
	case "numeric":
		return fmt.Sprintf("%s must be a number", field)
	case "e164":
		return fmt.Sprintf("%s must be a valid phone number (E.164 format)", field)
	case "datetime":
		return fmt.Sprintf("%s must be a valid datetime in format %s", field, e.Param())
	case "eqfield":
		return fmt.Sprintf("%s must be equal to %s", field, e.Param())
	case "nefield":
		return fmt.Sprintf("%s must not be equal to %s", field, e.Param())
	case "contains":
		return fmt.Sprintf("%s must contain '%s'", field, e.Param())
	case "excludes":
		return fmt.Sprintf("%s must not contain '%s'", field, e.Param())
	case "startswith":
		return fmt.Sprintf("%s must start with '%s'", field, e.Param())
	case "endswith":
		return fmt.Sprintf("%s must end with '%s'", field, e.Param())
	default:
		return fmt.Sprintf("%s failed validation: %s", field, e.Tag())
	}
}

func registerCustomValidators(v *validator.Validate) {
	// Register phone validator (Indonesian format)
	if err := v.RegisterValidation("phone", func(fl validator.FieldLevel) bool {
		phone := fl.Field().String()
		if phone == "" {
			return true
		}
		// Simple validation: starts with + or digit, 8-20 chars
		if len(phone) < 8 || len(phone) > 20 {
			return false
		}
		for i, r := range phone {
			if i == 0 && (r == '+' || (r >= '0' && r <= '9')) {
				continue
			}
			if r >= '0' && r <= '9' || r == '-' || r == ' ' {
				continue
			}
			return false
		}
		return true
	}); err != nil {
		panic(fmt.Sprintf("failed to register phone validator: %v", err))
	}

	// Register password strength validator
	if err := v.RegisterValidation("password", func(fl validator.FieldLevel) bool {
		password := fl.Field().String()
		if len(password) < 8 {
			return false
		}
		var hasUpper, hasLower, hasNumber bool
		for _, r := range password {
			switch {
			case r >= 'A' && r <= 'Z':
				hasUpper = true
			case r >= 'a' && r <= 'z':
				hasLower = true
			case r >= '0' && r <= '9':
				hasNumber = true
			}
		}
		return hasUpper && hasLower && hasNumber
	}); err != nil {
		panic(fmt.Sprintf("failed to register password validator: %v", err))
	}

	// Register indonesian NIK (ID number) validator
	if err := v.RegisterValidation("nik", func(fl validator.FieldLevel) bool {
		nik := fl.Field().String()
		if nik == "" {
			return true
		}
		if len(nik) != 16 {
			return false
		}
		for _, r := range nik {
			if r < '0' || r > '9' {
				return false
			}
		}
		return true
	}); err != nil {
		panic(fmt.Sprintf("failed to register nik validator: %v", err))
	}
}

// Validator wraps the go-playground validator
type Validator struct {
	validate *validator.Validate
}

// NewValidator creates a new validator wrapper
func NewValidator() *Validator {
	return &Validator{validate: GetValidator()}
}

// Struct validates a struct
func (v *Validator) Struct(s interface{}) error {
	return Validate(s)
}

// Var validates a variable
func (v *Validator) Var(field interface{}, tag string) error {
	return ValidateVar(field, tag)
}

// RegisterValidation registers a custom validation
func (v *Validator) RegisterValidation(tag string, fn validator.Func) error {
	return v.validate.RegisterValidation(tag, fn)
}
