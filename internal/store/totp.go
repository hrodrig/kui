package store

import (
	"context"
	"database/sql"
	"time"
)

func (s *Store) UserTOTPSecret(ctx context.Context, userID int64) (string, error) {
	var secret string
	err := s.db.QueryRowContext(ctx,
		`SELECT totp_secret FROM users WHERE id = ?`, userID,
	).Scan(&secret)
	return secret, err
}

func (s *Store) UserTOTPEnabled(ctx context.Context, userID int64) (bool, error) {
	secret, err := s.UserTOTPSecret(ctx, userID)
	if err != nil {
		return false, err
	}
	return secret != "", nil
}

func (s *Store) SetUserTOTPSecret(ctx context.Context, userID int64, secret string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE users SET totp_secret = ? WHERE id = ?`, secret, userID,
	)
	return err
}

func (s *Store) ClearUserTOTP(ctx context.Context, userID int64) error {
	return s.SetUserTOTPSecret(ctx, userID, "")
}

func (s *Store) CreateLoginPending(ctx context.Context, id string, userID int64, remember bool, expires time.Time) error {
	rem := 0
	if remember {
		rem = 1
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO login_pending (id, user_id, remember, expires_at) VALUES (?, ?, ?, ?)`,
		id, userID, rem, expires.UTC().Format(time.RFC3339),
	)
	return err
}

type LoginPending struct {
	UserID   int64
	Remember bool
}

func (s *Store) LoginPendingByID(ctx context.Context, id string) (LoginPending, error) {
	var p LoginPending
	var remember int
	var expires string
	err := s.db.QueryRowContext(ctx,
		`SELECT user_id, remember, expires_at FROM login_pending WHERE id = ?`, id,
	).Scan(&p.UserID, &remember, &expires)
	if err != nil {
		return LoginPending{}, err
	}
	t, err := time.Parse(time.RFC3339, expires)
	if err != nil || time.Now().After(t) {
		_, _ = s.db.ExecContext(ctx, `DELETE FROM login_pending WHERE id = ?`, id)
		return LoginPending{}, sql.ErrNoRows
	}
	p.Remember = remember == 1
	return p, nil
}

func (s *Store) ConsumeLoginPending(ctx context.Context, id string) (LoginPending, error) {
	p, err := s.LoginPendingByID(ctx, id)
	if err != nil {
		return LoginPending{}, err
	}
	_, _ = s.db.ExecContext(ctx, `DELETE FROM login_pending WHERE id = ?`, id)
	return p, nil
}

func (s *Store) DeleteLoginPending(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM login_pending WHERE id = ?`, id)
	return err
}

func (s *Store) SaveTOTPSetupPending(ctx context.Context, userID int64, secret string, expires time.Time) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO totp_setup_pending (user_id, secret, expires_at) VALUES (?, ?, ?)
		 ON CONFLICT(user_id) DO UPDATE SET secret = excluded.secret, expires_at = excluded.expires_at`,
		userID, secret, expires.UTC().Format(time.RFC3339),
	)
	return err
}

func (s *Store) PeekTOTPSetupPending(ctx context.Context, userID int64) (secret string, ok bool, err error) {
	var expires string
	err = s.db.QueryRowContext(ctx,
		`SELECT secret, expires_at FROM totp_setup_pending WHERE user_id = ?`, userID,
	).Scan(&secret, &expires)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	t, err := time.Parse(time.RFC3339, expires)
	if err != nil || time.Now().After(t) {
		_, _ = s.db.ExecContext(ctx, `DELETE FROM totp_setup_pending WHERE user_id = ?`, userID)
		return "", false, nil
	}
	return secret, true, nil
}

func (s *Store) ConsumeTOTPSetupPending(ctx context.Context, userID int64) (string, error) {
	var secret, expires string
	err := s.db.QueryRowContext(ctx,
		`SELECT secret, expires_at FROM totp_setup_pending WHERE user_id = ?`, userID,
	).Scan(&secret, &expires)
	if err != nil {
		return "", err
	}
	t, err := time.Parse(time.RFC3339, expires)
	if err != nil || time.Now().After(t) {
		_, _ = s.db.ExecContext(ctx, `DELETE FROM totp_setup_pending WHERE user_id = ?`, userID)
		return "", sql.ErrNoRows
	}
	_, _ = s.db.ExecContext(ctx, `DELETE FROM totp_setup_pending WHERE user_id = ?`, userID)
	return secret, nil
}

func (s *Store) DeleteTOTPSetupPending(ctx context.Context, userID int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM totp_setup_pending WHERE user_id = ?`, userID)
	return err
}
