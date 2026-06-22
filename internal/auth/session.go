package auth

import (
	"context"
	"net/http"
	"time"

	"github.com/hrodrig/kui/internal/config"
	"github.com/hrodrig/kui/internal/store"
)

type contextKey int

const userContextKey contextKey = 1

type Service struct {
	store    *store.Store
	cfg      config.SessionCfg
	cookie   string
	ttl      time.Duration
	shortTTL time.Duration
	secure   bool
}

func NewService(st *store.Store, cfg config.SessionCfg) *Service {
	ttl := time.Duration(cfg.TTLHours) * time.Hour
	if ttl <= 0 {
		ttl = 7 * 24 * time.Hour
	}
	shortTTL := time.Duration(cfg.ShortTTLHours) * time.Hour
	if shortTTL <= 0 {
		shortTTL = 8 * time.Hour
	}
	name := cfg.CookieName
	if name == "" {
		name = "kui_session"
	}
	return &Service{store: st, cfg: cfg, cookie: name, ttl: ttl, shortTTL: shortTTL, secure: cfg.Secure}
}

func (s *Service) Login(ctx context.Context, email, password string, remember bool) (string, *store.User, error) {
	u, needs2FA, err := s.Authenticate(ctx, email, password)
	if err != nil {
		return "", nil, err
	}
	if needs2FA {
		return "", nil, ErrTOTPRequired
	}
	token, err := s.CreateSession(ctx, u.ID, remember)
	if err != nil {
		return "", nil, err
	}
	return token, u, nil
}

// Authenticate verifies email/password. The bool is true when 2FA is enabled.
func (s *Service) Authenticate(ctx context.Context, email, password string) (*store.User, bool, error) {
	u, hash, err := s.store.UserByEmail(ctx, email)
	if err != nil {
		return nil, false, err
	}
	if !CheckPassword(hash, password) {
		return nil, false, store.ErrInvalidCredentials
	}
	enabled, err := s.store.UserTOTPEnabled(ctx, u.ID)
	if err != nil {
		return nil, false, err
	}
	return &u, enabled, nil
}

func (s *Service) CreateSession(ctx context.Context, userID int64, remember bool) (string, error) {
	token, err := NewSessionToken()
	if err != nil {
		return "", err
	}
	sessionTTL := s.shortTTL
	if remember {
		sessionTTL = s.ttl
	}
	if err := s.store.CreateSession(ctx, token, userID, time.Now().Add(sessionTTL)); err != nil {
		return "", err
	}
	return token, nil
}

const pendingCookie = "kui_2fa_pending"
const pendingTTL = 5 * time.Minute

func (s *Service) StartPendingLogin(ctx context.Context, userID int64, remember bool) (string, error) {
	id, err := NewSessionToken()
	if err != nil {
		return "", err
	}
	if err := s.store.CreateLoginPending(ctx, id, userID, remember, time.Now().Add(pendingTTL)); err != nil {
		return "", err
	}
	return id, nil
}

func (s *Service) CompletePendingLogin(ctx context.Context, pendingID string) (string, *store.User, error) {
	p, err := s.store.ConsumeLoginPending(ctx, pendingID)
	if err != nil {
		return "", nil, err
	}
	token, err := s.CreateSession(ctx, p.UserID, p.Remember)
	if err != nil {
		return "", nil, err
	}
	u, err := s.store.UserByID(ctx, p.UserID)
	if err != nil {
		return "", nil, err
	}
	return token, &u, nil
}

func (s *Service) SetPendingCookie(w http.ResponseWriter, pendingID string) {
	http.SetCookie(w, &http.Cookie{
		Name:     pendingCookie,
		Value:    pendingID,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   s.secure,
		MaxAge:   int(pendingTTL.Seconds()),
	})
}

func (s *Service) ClearPendingCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     pendingCookie,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   s.secure,
		MaxAge:   -1,
	})
}

func (s *Service) PendingIDFromRequest(r *http.Request) (string, error) {
	c, err := r.Cookie(pendingCookie)
	if err != nil {
		return "", err
	}
	if c.Value == "" {
		return "", http.ErrNoCookie
	}
	return c.Value, nil
}

func (s *Service) VerifyUserTOTP(ctx context.Context, userID int64, code string) (bool, error) {
	secret, err := s.store.UserTOTPSecret(ctx, userID)
	if err != nil {
		return false, err
	}
	if secret == "" {
		return false, nil
	}
	return ValidTOTPCode(secret, NormalizeTOTPCode(code)), nil
}

func (s *Service) Logout(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	return s.store.DeleteSession(ctx, token)
}

func (s *Service) UserFromRequest(ctx context.Context, r *http.Request) (*store.User, error) {
	c, err := r.Cookie(s.cookie)
	if err != nil {
		return nil, err
	}
	userID, err := s.store.SessionUserID(ctx, c.Value)
	if err != nil {
		return nil, err
	}
	u, err := s.store.UserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *Service) SetSessionCookie(w http.ResponseWriter, token string, remember bool) {
	c := &http.Cookie{
		Name:     s.cookie,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   s.secure,
	}
	if remember {
		c.MaxAge = int(s.ttl.Seconds())
	}
	http.SetCookie(w, c)
}

func (s *Service) ClearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     s.cookie,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   s.secure,
		MaxAge:   -1,
	})
}

func (s *Service) CookieName() string { return s.cookie }

func WithUser(ctx context.Context, u *store.User) context.Context {
	return context.WithValue(ctx, userContextKey, u)
}

func UserFromContext(ctx context.Context) (*store.User, bool) {
	u, ok := ctx.Value(userContextKey).(*store.User)
	return u, ok
}

func RequireAuth(svc *Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			u, err := svc.UserFromRequest(r.Context(), r)
			if err != nil {
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}
			next.ServeHTTP(w, r.WithContext(WithUser(r.Context(), u)))
		})
	}
}

func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, ok := UserFromContext(r.Context())
		if !ok || u.Role != store.RoleAdmin {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}
