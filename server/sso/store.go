package sso

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
)

var ErrUserNotFound = errors.New("user not found")

type SQLStore struct {
	db     *sql.DB
	driver string
}

func NewSQLStore(db *sql.DB, driver string) *SQLStore {
	return &SQLStore{db: db, driver: driver}
}

const userColumns = `id, sso_subject, name, nickname, email, phone, avatar, created_at, updated_at, last_login_at`

// Upsert links the SSO identity to a local account, refreshing the profile
// fields and the login timestamp on every sign-in.
func (s *SQLStore) Upsert(ctx context.Context, info UserInfo) (User, error) {
	if s.driver == "postgres" {
		var user User
		err := s.db.QueryRowContext(ctx, `
			INSERT INTO users (sso_subject, name, nickname, email, phone, avatar, last_login_at)
			VALUES ($1, $2, $3, $4, $5, $6, CURRENT_TIMESTAMP)
			ON CONFLICT (sso_subject) DO UPDATE SET
				name = EXCLUDED.name,
				nickname = EXCLUDED.nickname,
				email = EXCLUDED.email,
				phone = EXCLUDED.phone,
				avatar = EXCLUDED.avatar,
				updated_at = CURRENT_TIMESTAMP,
				last_login_at = CURRENT_TIMESTAMP
			RETURNING `+userColumns,
			info.Subject, info.Name, info.Nickname, info.Email, info.Phone, info.Avatar).
			Scan(scanTargets(&user)...)
		if err != nil {
			return User{}, fmt.Errorf("upserting SSO user: %w", err)
		}
		return user, nil
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO users (sso_subject, name, nickname, email, phone, avatar, last_login_at)
		VALUES (?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON DUPLICATE KEY UPDATE
			name = VALUES(name),
			nickname = VALUES(nickname),
			email = VALUES(email),
			phone = VALUES(phone),
			avatar = VALUES(avatar),
			updated_at = CURRENT_TIMESTAMP,
			last_login_at = CURRENT_TIMESTAMP`,
		info.Subject, info.Name, info.Nickname, info.Email, info.Phone, info.Avatar)
	if err != nil {
		return User{}, fmt.Errorf("upserting SSO user: %w", err)
	}
	// LastInsertId is unreliable for ON DUPLICATE KEY UPDATE, so look the row
	// up by the subject we just wrote.
	return s.LookupBySubject(ctx, info.Subject)
}

func (s *SQLStore) LookupBySubject(ctx context.Context, subject string) (User, error) {
	query := "SELECT " + userColumns + " FROM users WHERE sso_subject = ?"
	if s.driver == "postgres" {
		query = "SELECT " + userColumns + " FROM users WHERE sso_subject = $1"
	}
	return s.queryUser(ctx, query, subject)
}

func (s *SQLStore) Lookup(ctx context.Context, id int64) (User, error) {
	query := "SELECT " + userColumns + " FROM users WHERE id = ?"
	if s.driver == "postgres" {
		query = "SELECT " + userColumns + " FROM users WHERE id = $1"
	}
	return s.queryUser(ctx, query, id)
}

func (s *SQLStore) queryUser(ctx context.Context, query string, arg any) (User, error) {
	var user User
	if err := s.db.QueryRowContext(ctx, query, arg).Scan(scanTargets(&user)...); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return User{}, ErrUserNotFound
		}
		return User{}, fmt.Errorf("looking up user: %w", err)
	}
	return user, nil
}

// scanTargets keeps the column order in one place. Nullable text columns scan
// through sql.NullString so a missing email or phone becomes "".
func scanTargets(user *User) []any {
	return []any{
		&user.ID,
		&user.Subject,
		nullString(&user.Name),
		nullString(&user.Nickname),
		nullString(&user.Email),
		nullString(&user.Phone),
		nullString(&user.Avatar),
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.LastLogin,
	}
}

// nullString adapts a *string field to a scan target tolerating SQL NULL.
func nullString(target *string) sql.Scanner {
	return &nullStringScanner{target: target}
}

type nullStringScanner struct {
	target *string
}

func (n *nullStringScanner) Scan(value any) error {
	switch typed := value.(type) {
	case nil:
		*n.target = ""
	case string:
		*n.target = typed
	case []byte:
		*n.target = string(typed)
	default:
		return fmt.Errorf("cannot scan %T into string", value)
	}
	return nil
}

// userIDFromSubject parses the JWT subject, which holds the local user ID.
func userIDFromSubject(subject string) (int64, error) {
	id, err := strconv.ParseInt(subject, 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("invalid session subject")
	}
	return id, nil
}
