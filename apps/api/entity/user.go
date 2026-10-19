// Package entity holds the domain models.
package entity

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

type UserStatus string

const (
	UserStatusActive   UserStatus = "active"
	UserStatusInactive UserStatus = "inactive"
	UserStatusPending  UserStatus = "pending"
)

// User is the users row. Indexes and column defaults are owned by the SQL
// migrations (golang-migrate is the schema source of truth), so this struct
// deliberately carries no uniqueIndex/default tags: AutoMigrate would otherwise
// install a *global* unique index on email, breaking re-registration after a
// soft delete (the migration uses a partial index WHERE deleted_at IS NULL).
type User struct {
	ID          string         `gorm:"type:uuid;primaryKey" json:"id"`
	Email       string         `gorm:"not null" json:"email"`
	Password    string         `gorm:"not null" json:"-"`
	Name        string         `gorm:"not null" json:"name"`
	Phone       string         `json:"phone"`
	Status      UserStatus     `gorm:"type:varchar(20)" json:"status"`
	Roles       pq.StringArray `gorm:"type:text[]" json:"roles"`
	CompanyCode string         `gorm:"type:varchar(50)" json:"companyCode"`
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == "" {
		u.ID = uuid.New().String()
	}
	return nil
}

func (u *User) TableName() string {
	return "users"
}
