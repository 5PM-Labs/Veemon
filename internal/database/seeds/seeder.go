// Package seeds provides database seeders.
package seeds

import (
	"context"
	"errors"
	"fmt"

	"github.com/5PM-Labs/veemon/internal/domain"

	"github.com/5PM-Labs/veemon/internal/infra/database"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

// Seeder handles database seeding.
type Seeder struct {
	pool *pgxpool.Pool
}

// New creates a new Seeder instance.
func New(pool *pgxpool.Pool) *Seeder {
	return &Seeder{pool: pool}
}

// SeedAll runs all seeders.
func (s *Seeder) SeedAll(ctx context.Context) error {
	seeders := []func(context.Context) error{
		s.SeedUsers,
	}

	for _, seeder := range seeders {
		if err := seeder(ctx); err != nil {
			return err
		}
	}

	return nil
}

// SeedUsers seeds one account per role. Seeding is idempotent: existing emails
// are skipped, so it is safe to re-run.
func (s *Seeder) SeedUsers(ctx context.Context) error {
	fmt.Println("Seeding users...")

	users := []struct {
		Email    string
		Password string
		Name     string
		Phone    string
		Roles    []string
	}{
		{"superadmin@example.com", "SuperAdmin123!", "Super Admin", "081234567890", []string{domain.RoleSuperAdmin}},
		{"admin@example.com", "Admin123!", "Admin User", "081234567891", []string{domain.RoleAdmin}},
		{"user@example.com", "User123!", "Regular User", "081234567894", []string{domain.RoleUser}},
	}

	for _, u := range users {
		// Only a genuinely missing row means insert; any other error is real.
		var existing string
		err := s.pool.QueryRow(ctx, `SELECT id FROM users WHERE email=$1`, u.Email).Scan(&existing)
		switch {
		case err == nil:
			fmt.Printf("  User %s already exists, skipping...\n", u.Email)
			continue
		case !errors.Is(err, pgx.ErrNoRows):
			return fmt.Errorf("failed to check existing user %s: %w", u.Email, err)
		}

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("failed to hash password for %s: %w", u.Email, err)
		}

		user := domain.User{
			ID:            uuid.NewString(),
			Email:         u.Email,
			PasswordHash:  string(hashedPassword),
			Name:          u.Name,
			Phone:         u.Phone,
			Status:        domain.UserStatusActive,
			EmailVerified: true,
			AuthProvider:  domain.AuthProviderLocal,
			CompanyCode:   "COMPANY-001",
		}

		err = database.InTx(ctx, s.pool, func(tx pgx.Tx) error {
			if _, err := tx.Exec(ctx, `
				INSERT INTO users (id, email, password_hash, name, phone, status, email_verified,
				                   auth_provider, company_code)
				VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
				user.ID, user.Email, user.PasswordHash, user.Name, user.Phone,
				user.Status, user.EmailVerified, user.AuthProvider, user.CompanyCode); err != nil {
				return err
			}
			_, err := tx.Exec(ctx, `
				INSERT INTO user_roles (user_id, role_id)
				SELECT $1, id FROM roles WHERE name = ANY($2)`,
				user.ID, u.Roles)
			return err
		})
		if err != nil {
			return fmt.Errorf("failed to seed user %s: %w", u.Email, err)
		}

		fmt.Printf("  Created user: %s (%s)\n", u.Name, u.Email)
	}

	fmt.Println("Users seeded successfully!")
	return nil
}
