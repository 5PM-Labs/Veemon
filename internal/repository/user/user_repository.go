// Package user_repository provides data access for users.
package user_repository

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/5PM-Labs/veemon/internal/domain"
	"github.com/5PM-Labs/veemon/internal/infra/database"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ListParams struct {
	Page      int
	Size      int
	Search    string
	SortBy    string
	SortOrder string
}

type Repository interface {
	Create(ctx context.Context, user *domain.User) error
	FindByID(ctx context.Context, id string) (*domain.User, error)
	FindByEmail(ctx context.Context, email string) (*domain.User, error)
	FindByAuth0Subject(ctx context.Context, subject string) (*domain.User, error)
	FindAll(ctx context.Context, params ListParams) ([]domain.User, int64, error)
	// UpdateFields applies a partial update to only the given columns. It
	// returns domain.ErrNotFound if no live row matches. Column-scoped updates
	// (instead of Save on a previously-read struct) avoid clobbering columns
	// changed concurrently.
	UpdateFields(ctx context.Context, id string, fields map[string]any) (*domain.User, error)
	Delete(ctx context.Context, id string) error
	// AssignRoles replaces the user's role assignments in one transaction.
	AssignRoles(ctx context.Context, userID string, roles []string) error
	// RecordLogin stamps last_login_at and clears lockout state on success.
	RecordLogin(ctx context.Context, id string) error
	// RecordFailedLogin increments the failed counter and locks the account at
	// max failures. Returns the new count and the lock state.
	RecordFailedLogin(ctx context.Context, id string, maxAttempts int, lockout time.Duration) (int, bool, error)
}

var allowedSortColumns = map[string]bool{
	"created_at": true,
	"updated_at": true,
	"name":       true,
	"email":      true,
	"status":     true,
}

// selectColumns is shared by every read so the scan order matches exactly.
// Roles come from user_roles (the single source of truth) as a subquery, so no
// read needs a follow-up query or a join that would duplicate rows.
const selectColumns = `SELECT u.id, u.email, u.password_hash, u.name, u.phone, u.avatar_url, u.locale,
	u.timezone, u.status, u.email_verified, u.mfa_enabled, u.auth_provider, u.auth0_subject,
	u.company_code, coalesce((SELECT array_agg(r.name ORDER BY r.name) FROM user_roles ur
	    JOIN roles r ON r.id = ur.role_id WHERE ur.user_id = u.id), '{}'),
	u.metadata, u.failed_login_count, u.last_login_at, u.locked_until,
	u.created_at, u.updated_at, u.deleted_at`

type repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) Repository {
	return &repository{pool: pool}
}

func (r *repository) Create(ctx context.Context, user *domain.User) error {
	// metadata is NOT NULL DEFAULT '{}'; a nil map would bind SQL NULL.
	if user.Metadata == nil {
		user.Metadata = map[string]any{}
	}
	// The insert and the role assignment are one unit: a user row without its
	// roles would silently lose authorization.
	err := database.InTx(ctx, r.pool, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, `
			INSERT INTO users (email, password_hash, name, phone, avatar_url, locale, timezone,
			                   status, email_verified, mfa_enabled, auth_provider, auth0_subject,
			                   company_code, metadata)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
			RETURNING id, created_at, updated_at`,
			user.Email, user.PasswordHash, user.Name, user.Phone,
			user.AvatarURL, user.Locale, user.Timezone,
			user.Status, user.EmailVerified, user.MFAEnabled,
			user.AuthProvider, user.Auth0Subject, user.CompanyCode,
			user.Metadata,
		).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt); err != nil {
			return err
		}
		if len(user.Roles) == 0 {
			return nil
		}
		_, err := tx.Exec(ctx, `
			INSERT INTO user_roles (user_id, role_id)
			SELECT $1, id FROM roles WHERE name = ANY($2)`, user.ID, user.Roles)
		return err
	})
	if err != nil {
		if database.IsUniqueViolation(err) {
			return domain.ErrDuplicateEntry
		}
		return err
	}
	return nil
}

// scanUser scans one User. Callers pass a pgx.Row (QueryRow/UpdateFields) or
// pgx.Rows (FindAll); both implement Scan(...any) error.
func scanUser(row interface{ Scan(...any) error }) (*domain.User, error) {
	var u domain.User
	// Nullable columns scan into pointers first; pgx refuses to put NULL in a
	// plain string field.
	var passwordHash, phone, avatarURL, locale, timezone, auth0Subject, companyCode *string
	var lastLoginAt, lockedUntil, deletedAt *time.Time

	err := row.Scan(
		&u.ID, &u.Email, &passwordHash, &u.Name, &phone,
		&avatarURL, &locale, &timezone, &u.Status,
		&u.EmailVerified, &u.MFAEnabled, &u.AuthProvider, &auth0Subject,
		&companyCode, &u.Roles, &u.Metadata, &u.FailedLoginCount,
		&lastLoginAt, &lockedUntil, &u.CreatedAt, &u.UpdatedAt, &deletedAt,
	)
	if err != nil {
		return nil, err
	}
	u.PasswordHash = deref(passwordHash)
	u.Phone = deref(phone)
	u.AvatarURL = deref(avatarURL)
	u.Locale = deref(locale)
	u.Timezone = deref(timezone)
	u.Auth0Subject = deref(auth0Subject)
	u.CompanyCode = deref(companyCode)
	u.LastLoginAt = lastLoginAt
	u.LockedUntil = lockedUntil
	u.DeletedAt = deletedAt
	return &u, nil
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func (r *repository) FindByID(ctx context.Context, id string) (*domain.User, error) {
	u, err := scanUser(r.pool.QueryRow(ctx, selectColumns+` FROM users u WHERE u.id=$1 AND u.deleted_at IS NULL`, id))
	return u, translate(err)
}

func (r *repository) FindByAuth0Subject(ctx context.Context, subject string) (*domain.User, error) {
	u, err := scanUser(r.pool.QueryRow(ctx, selectColumns+` FROM users u WHERE u.auth0_subject=$1 AND u.deleted_at IS NULL`, subject))
	return u, translate(err)
}

func (r *repository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	u, err := scanUser(r.pool.QueryRow(ctx, selectColumns+` FROM users u WHERE u.email=$1 AND u.deleted_at IS NULL`, email))
	return u, translate(err)
}

func (r *repository) FindAll(ctx context.Context, params ListParams) ([]domain.User, int64, error) {
	// Whitelist sort column and direction: they are concatenated into ORDER BY
	// and cannot be parameterized, so they must never come straight from input.
	sortColumn := "created_at"
	if allowedSortColumns[params.SortBy] {
		sortColumn = params.SortBy
	}
	sortOrder := "desc"
	if params.SortOrder == "asc" {
		sortOrder = "asc"
	}

	where := "u.deleted_at IS NULL"
	args := []any{}
	if params.Search != "" {
		args = append(args, "%"+params.Search+"%")
		where += ` AND (u.name ILIKE $1 OR u.email ILIKE $1)`
	}

	var total int64
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM users u WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	offset := (params.Page - 1) * params.Size
	args = append(args, params.Size, offset)
	rows, err := r.pool.Query(ctx,
		selectColumns+` FROM users u WHERE `+where+` ORDER BY u.`+sortColumn+` `+sortOrder+
			` LIMIT $`+strconv.Itoa(len(args)-1)+` OFFSET $`+strconv.Itoa(len(args)),
		args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	users := make([]domain.User, 0)
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, 0, err
		}
		users = append(users, *u)
	}
	return users, total, rows.Err()
}

func (r *repository) UpdateFields(ctx context.Context, id string, fields map[string]any) (*domain.User, error) {
	if len(fields) == 0 {
		return r.FindByID(ctx, id)
	}

	set := make([]string, 0, len(fields))
	args := make([]any, 0, len(fields)+1)
	for k, v := range fields {
		// Column names come from the usecase's fixed allowlist, never from input.
		args = append(args, v)
		set = append(set, k+"=$"+strconv.Itoa(len(args)))
	}
	args = append(args, id)
	tag, err := r.pool.Exec(ctx, `UPDATE users SET `+strings.Join(set, ", ")+`, updated_at = now() WHERE id=$`+
		strconv.Itoa(len(args))+` AND deleted_at IS NULL`, args...)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, domain.ErrNotFound
	}
	return r.FindByID(ctx, id)
}

func (r *repository) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `UPDATE users SET deleted_at = now() WHERE id=$1 AND deleted_at IS NULL`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// AssignRoles replaces the user's role assignments. Unknown role names are
// ignored (the FK would reject them anyway) so a stale name cannot fail a
// legitimate assignment of the remaining roles.
func (r *repository) AssignRoles(ctx context.Context, userID string, roles []string) error {
	return database.InTx(ctx, r.pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `DELETE FROM user_roles WHERE user_id=$1`, userID); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `
			INSERT INTO user_roles (user_id, role_id)
			SELECT $1, id FROM roles WHERE name = ANY($2)`, userID, roles)
		return err
	})
}

// RecordLogin stamps last_login_at (and clears lockout counters) on success.
func (r *repository) RecordLogin(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET last_login_at = now(), failed_login_count = 0, locked_until = NULL WHERE id=$1`,
		id)
	return err
}

// RecordFailedLogin increments the failed counter and locks the account once
// the threshold is reached. Returns the new failed count and whether the
// account is now locked.
func (r *repository) RecordFailedLogin(ctx context.Context, id string, max int, lockout time.Duration) (int, bool, error) {
	var failed int
	var locked bool
	err := r.pool.QueryRow(ctx, `
		UPDATE users
		SET failed_login_count = failed_login_count + 1,
		    locked_until = CASE WHEN failed_login_count + 1 >= $2
		                        THEN now() + make_interval(secs => $3) ELSE NULL END
		WHERE id=$1
		RETURNING failed_login_count, locked_until IS NOT NULL`,
		id, max, lockout.Seconds()).Scan(&failed, &locked)
	return failed, locked, err
}

// translate maps pgx's sentinel errors to domain sentinels so callers above the
// repository never import the driver.
func translate(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrNotFound
	}
	return err
}
