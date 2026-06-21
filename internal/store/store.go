package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type Role string

const (
	RoleAdmin Role = "admin"
	RoleUser  Role = "user"
)

type User struct {
	ID        int64
	Email     string
	Role      Role
	CreatedAt time.Time
}

type UserWithHosts struct {
	User
	Hosts       []string
	TOTPEnabled bool
}

type Store struct {
	db *sql.DB
}

func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) PingContext(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

func (s *Store) migrate() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			email TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			role TEXT NOT NULL CHECK (role IN ('admin', 'user')),
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS sessions (
			id TEXT PRIMARY KEY,
			user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			expires_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS user_hosts (
			user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			host TEXT NOT NULL,
			PRIMARY KEY (user_id, host)
		)`,
	}
	for _, q := range stmts {
		if _, err := s.db.Exec(q); err != nil {
			return fmt.Errorf("migrate: %w", err)
		}
	}
	// additive migrations (ignore duplicate column)
	_, _ = s.db.Exec(`ALTER TABLE users ADD COLUMN totp_secret TEXT NOT NULL DEFAULT ''`)
	pending := []string{
		`CREATE TABLE IF NOT EXISTS login_pending (
			id TEXT PRIMARY KEY,
			user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			remember INTEGER NOT NULL DEFAULT 0,
			expires_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS totp_setup_pending (
			user_id INTEGER PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
			secret TEXT NOT NULL,
			expires_at TEXT NOT NULL
		)`,
	}
	for _, q := range pending {
		if _, err := s.db.Exec(q); err != nil {
			return fmt.Errorf("migrate pending: %w", err)
		}
	}
	return nil
}

func (s *Store) CountUsers(ctx context.Context) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&n)
	return n, err
}

func (s *Store) CountAdmins(ctx context.Context) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users WHERE role = ?`, RoleAdmin).Scan(&n)
	return n, err
}

func (s *Store) CreateUser(ctx context.Context, email, passwordHash string, role Role) (int64, error) {
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO users (email, password_hash, role, created_at) VALUES (?, ?, ?, ?)`,
		email, passwordHash, role, time.Now().UTC().Format(time.RFC3339),
	)
	if err != nil && strings.Contains(err.Error(), "UNIQUE") {
		return 0, ErrDuplicateEmail
	}
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) UserByEmail(ctx context.Context, email string) (User, string, error) {
	var u User
	var hash string
	var created string
	err := s.db.QueryRowContext(ctx,
		`SELECT id, email, password_hash, role, created_at FROM users WHERE email = ?`, email,
	).Scan(&u.ID, &u.Email, &hash, &u.Role, &created)
	if err != nil {
		return User{}, "", err
	}
	u.CreatedAt, _ = time.Parse(time.RFC3339, created)
	return u, hash, nil
}

func (s *Store) UserByID(ctx context.Context, id int64) (User, error) {
	var u User
	var created string
	err := s.db.QueryRowContext(ctx,
		`SELECT id, email, role, created_at FROM users WHERE id = ?`, id,
	).Scan(&u.ID, &u.Email, &u.Role, &created)
	if err != nil {
		return User{}, err
	}
	u.CreatedAt, _ = time.Parse(time.RFC3339, created)
	return u, nil
}

func (s *Store) ListUsers(ctx context.Context) ([]User, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, email, role, created_at FROM users ORDER BY email`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []User
	for rows.Next() {
		var u User
		var created string
		if err := rows.Scan(&u.ID, &u.Email, &u.Role, &created); err != nil {
			return nil, err
		}
		u.CreatedAt, _ = time.Parse(time.RFC3339, created)
		out = append(out, u)
	}
	return out, rows.Err()
}

func (s *Store) ListUsersWithHosts(ctx context.Context) ([]UserWithHosts, error) {
	users, err := s.ListUsers(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]UserWithHosts, len(users))
	for i, u := range users {
		hosts, err := s.UserHosts(ctx, u.ID)
		if err != nil {
			return nil, err
		}
		totpOn, err := s.UserTOTPEnabled(ctx, u.ID)
		if err != nil {
			return nil, err
		}
		out[i] = UserWithHosts{User: u, Hosts: hosts, TOTPEnabled: totpOn}
	}
	return out, nil
}

func (s *Store) UpdateUserRole(ctx context.Context, id int64, role Role) error {
	_, err := s.db.ExecContext(ctx, `UPDATE users SET role = ? WHERE id = ?`, role, id)
	return err
}

func (s *Store) UpdateUserPassword(ctx context.Context, id int64, passwordHash string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE users SET password_hash = ? WHERE id = ?`, passwordHash, id)
	return err
}

func (s *Store) DeleteUser(ctx context.Context, id int64) error {
	u, err := s.UserByID(ctx, id)
	if err != nil {
		return err
	}
	if u.Role == RoleAdmin {
		n, err := s.CountAdmins(ctx)
		if err != nil {
			return err
		}
		if n <= 1 {
			return ErrLastAdmin
		}
	}
	_, err = s.db.ExecContext(ctx, `DELETE FROM users WHERE id = ?`, id)
	return err
}

func (s *Store) CreateSession(ctx context.Context, id string, userID int64, expires time.Time) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO sessions (id, user_id, expires_at) VALUES (?, ?, ?)`,
		id, userID, expires.UTC().Format(time.RFC3339),
	)
	return err
}

func (s *Store) DeleteSession(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE id = ?`, id)
	return err
}

func (s *Store) DeleteUserSessions(ctx context.Context, userID int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE user_id = ?`, userID)
	return err
}

func (s *Store) SessionUserID(ctx context.Context, id string) (int64, error) {
	var userID int64
	var expires string
	err := s.db.QueryRowContext(ctx,
		`SELECT user_id, expires_at FROM sessions WHERE id = ?`, id,
	).Scan(&userID, &expires)
	if err != nil {
		return 0, err
	}
	t, err := time.Parse(time.RFC3339, expires)
	if err != nil || time.Now().After(t) {
		_, _ = s.db.ExecContext(ctx, `DELETE FROM sessions WHERE id = ?`, id)
		return 0, sql.ErrNoRows
	}
	return userID, nil
}

func (s *Store) UserHosts(ctx context.Context, userID int64) ([]string, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT host FROM user_hosts WHERE user_id = ? ORDER BY host`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hosts []string
	for rows.Next() {
		var h string
		if err := rows.Scan(&h); err != nil {
			return nil, err
		}
		hosts = append(hosts, h)
	}
	return hosts, rows.Err()
}

func (s *Store) ListDistinctHosts(ctx context.Context) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT DISTINCT host FROM user_hosts ORDER BY host`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hosts []string
	for rows.Next() {
		var h string
		if err := rows.Scan(&h); err != nil {
			return nil, err
		}
		hosts = append(hosts, h)
	}
	return hosts, rows.Err()
}

func (s *Store) SetUserHosts(ctx context.Context, userID int64, hosts []string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `DELETE FROM user_hosts WHERE user_id = ?`, userID); err != nil {
		return err
	}
	for _, host := range hosts {
		host = strings.TrimSpace(host)
		if host == "" {
			continue
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO user_hosts (user_id, host) VALUES (?, ?)`, userID, host); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) CanAccessHost(ctx context.Context, user User, host string) (bool, error) {
	if user.Role == RoleAdmin {
		return true, nil
	}
	hosts, err := s.UserHosts(ctx, user.ID)
	if err != nil {
		return false, err
	}
	for _, h := range hosts {
		if h == host {
			return true, nil
		}
	}
	return false, nil
}

func ParseRole(s string) (Role, error) {
	switch Role(strings.TrimSpace(s)) {
	case RoleAdmin:
		return RoleAdmin, nil
	case RoleUser:
		return RoleUser, nil
	default:
		return "", errors.New("invalid role")
	}
}

func ParseHostsText(text string) []string {
	lines := strings.Split(text, "\n")
	var out []string
	seen := make(map[string]struct{})
	for _, line := range lines {
		for _, part := range strings.Split(line, ",") {
			h := strings.TrimSpace(part)
			if h == "" {
				continue
			}
			if _, ok := seen[h]; ok {
				continue
			}
			seen[h] = struct{}{}
			out = append(out, h)
		}
	}
	return out
}
