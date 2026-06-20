package auth_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/hrodrig/kui/internal/auth"
	"github.com/hrodrig/kui/internal/config"
	"github.com/hrodrig/kui/internal/store"
)

func newTestStore(t *testing.T) *store.Store {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "kui.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	return st
}

func seededService(t *testing.T) (*store.Store, *auth.Service, int64) {
	t.Helper()
	st := newTestStore(t)
	svc := auth.NewService(st, config.SessionCfg{CookieName: "kui_test", Secure: false})
	ctx := context.Background()
	hash, err := auth.HashPassword("pass123")
	if err != nil {
		t.Fatal(err)
	}
	u, err := st.CreateUser(ctx, "user@test", hash, store.RoleUser)
	if err != nil {
		t.Fatal(err)
	}
	return st, svc, u
}

func TestNewServiceDefaults(t *testing.T) {
	cfg := config.SessionCfg{} // all zero
	svc := auth.NewService(newTestStore(t), cfg)
	if c := svc.CookieName(); c != "kui_session" {
		t.Errorf("default cookie = %q, want kui_session", c)
	}
}

func TestLogin(t *testing.T) {
	_, svc, _ := seededService(t)
	ctx := context.Background()

	t.Run("valid", func(t *testing.T) {
		token, u, err := svc.Login(ctx, "user@test", "pass123", false)
		if err != nil {
			t.Fatal(err)
		}
		if token == "" {
			t.Fatal("expected non-empty token")
		}
		if u.Email != "user@test" {
			t.Fatalf("got email %s", u.Email)
		}
	})

	t.Run("wrong password", func(t *testing.T) {
		_, _, err := svc.Login(ctx, "user@test", "wrong", false)
		if err != store.ErrInvalidCredentials {
			t.Fatalf("got %v, want ErrInvalidCredentials", err)
		}
	})

	t.Run("unknown email", func(t *testing.T) {
		_, _, err := svc.Login(ctx, "nobody@test", "pass123", false)
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestLoginRemember(t *testing.T) {
	_, svc, _ := seededService(t)
	_, _, err := svc.Login(context.Background(), "user@test", "pass123", true)
	if err != nil {
		t.Fatal(err)
	}
}

func TestUserFromRequest(t *testing.T) {
	st, svc, uid := seededService(t)
	ctx := context.Background()

	token, _, err := svc.Login(ctx, "user@test", "pass123", false)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("valid cookie", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/", nil)
		r.AddCookie(&http.Cookie{Name: "kui_test", Value: token})
		u, err := svc.UserFromRequest(ctx, r)
		if err != nil {
			t.Fatal(err)
		}
		if u.ID != uid {
			t.Fatalf("user ID = %d, want %d", u.ID, uid)
		}
	})

	t.Run("no cookie", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/", nil)
		_, err := svc.UserFromRequest(ctx, r)
		if err == nil {
			t.Fatal("expected error without cookie")
		}
	})

	t.Run("bad token", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/", nil)
		r.AddCookie(&http.Cookie{Name: "kui_test", Value: "invalid"})
		_, err := svc.UserFromRequest(ctx, r)
		if err == nil {
			t.Fatal("expected error with bad token")
		}
	})

	t.Run("deleted session", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/", nil)
		r.AddCookie(&http.Cookie{Name: "kui_test", Value: token})
		_ = st.DeleteSession(ctx, token)
		_, err := svc.UserFromRequest(ctx, r)
		if err == nil {
			t.Fatal("expected error after session deleted")
		}
	})
}

func TestLogout(t *testing.T) {
	_, svc, _ := seededService(t)
	ctx := context.Background()

	t.Run("empty token", func(t *testing.T) {
		if err := svc.Logout(ctx, ""); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("valid token", func(t *testing.T) {
		token, _, err := svc.Login(ctx, "user@test", "pass123", false)
		if err != nil {
			t.Fatal(err)
		}
		if err := svc.Logout(ctx, token); err != nil {
			t.Fatal(err)
		}
	})
}

func TestSetAndClearSessionCookie(t *testing.T) {
	svc := auth.NewService(newTestStore(t), config.SessionCfg{Secure: false})
	w := httptest.NewRecorder()

	svc.SetSessionCookie(w, "tok123", false)
	c := w.Header().Get("Set-Cookie")
	if c == "" {
		t.Fatal("expected Set-Cookie header")
	}

	w2 := httptest.NewRecorder()
	svc.ClearSessionCookie(w2)
	c2 := w2.Header().Get("Set-Cookie")
	if c2 == "" {
		t.Fatal("expected Set-Cookie on clear")
	}
}

func TestSetAndClearPendingCookie(t *testing.T) {
	svc := auth.NewService(newTestStore(t), config.SessionCfg{Secure: false})

	w := httptest.NewRecorder()
	svc.SetPendingCookie(w, "pending-id")
	c := w.Header().Get("Set-Cookie")
	if c == "" {
		t.Fatal("expected Set-Cookie")
	}

	w2 := httptest.NewRecorder()
	svc.ClearPendingCookie(w2)
	c2 := w2.Header().Get("Set-Cookie")
	if c2 == "" {
		t.Fatal("expected Set-Cookie on clear")
	}
}

func TestPendingIDFromRequest(t *testing.T) {
	svc := auth.NewService(newTestStore(t), config.SessionCfg{Secure: false})

	t.Run("has cookie", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/", nil)
		r.AddCookie(&http.Cookie{Name: "kui_2fa_pending", Value: "abc"})
		id, err := svc.PendingIDFromRequest(r)
		if err != nil {
			t.Fatal(err)
		}
		if id != "abc" {
			t.Fatalf("got %s", id)
		}
	})

	t.Run("empty value", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/", nil)
		r.AddCookie(&http.Cookie{Name: "kui_2fa_pending", Value: ""})
		_, err := svc.PendingIDFromRequest(r)
		if err == nil {
			t.Fatal("expected error for empty value")
		}
	})

	t.Run("no cookie", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/", nil)
		_, err := svc.PendingIDFromRequest(r)
		if err == nil {
			t.Fatal("expected error for missing cookie")
		}
	})
}

func TestRequireAuthRedirects(t *testing.T) {
	svc := auth.NewService(newTestStore(t), config.SessionCfg{Secure: false})
	handler := auth.RequireAuth(svc)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d", w.Code)
	}
	loc := w.Header().Get("Location")
	if loc != "/login" {
		t.Fatalf("Location = %s", loc)
	}
}

func TestRequireAuthPasses(t *testing.T) {
	_, svc, _ := seededService(t)
	ctx := context.Background()
	token, _, err := svc.Login(ctx, "user@test", "pass123", false)
	if err != nil {
		t.Fatal(err)
	}

	handler := auth.RequireAuth(svc)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest("GET", "/", nil)
	r.AddCookie(&http.Cookie{Name: "kui_test", Value: token})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestRequireAdmin(t *testing.T) {
	t.Run("admin passes", func(t *testing.T) {
		u := &store.User{Role: store.RoleAdmin}
		ctx := auth.WithUser(context.Background(), u)
		r := httptest.NewRequest("GET", "/", nil).WithContext(ctx)
		w := httptest.NewRecorder()
		auth.RequireAdmin(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		})).ServeHTTP(w, r)
		if w.Code != http.StatusOK {
			t.Fatalf("admin: got %d", w.Code)
		}
	})

	t.Run("user forbidden", func(t *testing.T) {
		u := &store.User{Role: store.RoleUser}
		ctx := auth.WithUser(context.Background(), u)
		r := httptest.NewRequest("GET", "/", nil).WithContext(ctx)
		w := httptest.NewRecorder()
		auth.RequireAdmin(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		})).ServeHTTP(w, r)
		if w.Code != http.StatusForbidden {
			t.Fatalf("user: got %d", w.Code)
		}
	})

	t.Run("no user", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()
		auth.RequireAdmin(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		})).ServeHTTP(w, r)
		if w.Code != http.StatusForbidden {
			t.Fatalf("no user: got %d", w.Code)
		}
	})
}

func TestWithUserAndFromContext(t *testing.T) {
	u := &store.User{Email: "ctx@test"}
	ctx := auth.WithUser(context.Background(), u)
	got, ok := auth.UserFromContext(ctx)
	if !ok {
		t.Fatal("UserFromContext returned false")
	}
	if got.Email != "ctx@test" {
		t.Fatalf("email = %s", got.Email)
	}
}

func TestAuthenticate(t *testing.T) {
	st, svc, uid := seededService(t)
	ctx := context.Background()

	// enable TOTP on user
	secret := "JBSWY3DPEHPK3PXP"
	if err := st.SetUserTOTPSecret(ctx, uid, secret); err != nil {
		t.Fatal(err)
	}

	t.Run("needs 2FA", func(t *testing.T) {
		u, needs2FA, err := svc.Authenticate(ctx, "user@test", "pass123")
		if err != nil {
			t.Fatal(err)
		}
		if !needs2FA {
			t.Fatal("expected needs2FA=true")
		}
		if u.ID != uid {
			t.Fatalf("user ID = %d", u.ID)
		}
	})

	t.Run("Login returns ErrTOTPRequired", func(t *testing.T) {
		_, _, err := svc.Login(ctx, "user@test", "pass123", false)
		if err != auth.ErrTOTPRequired {
			t.Fatalf("got %v, want ErrTOTPRequired", err)
		}
	})
}

func TestStartAndCompletePendingLogin(t *testing.T) {
	st, svc, uid := seededService(t)
	ctx := context.Background()

	_ = st.SetUserTOTPSecret(ctx, uid, "JBSWY3DPEHPK3PXP")

	pendingID, err := svc.StartPendingLogin(ctx, uid, true)
	if err != nil {
		t.Fatal(err)
	}
	if pendingID == "" {
		t.Fatal("empty pending ID")
	}

	_ = st.SetUserTOTPSecret(ctx, uid, "JBSWY3DPEHPK3PXP")

	token, u2, err := svc.CompletePendingLogin(ctx, pendingID)
	if err != nil {
		t.Fatal(err)
	}
	if token == "" {
		t.Fatal("empty session token after completion")
	}
	if u2.ID != uid {
		t.Fatalf("user ID = %d", u2.ID)
	}

	// ConsumeLoginPending on same ID should fail (already consumed)
	_, _, err = svc.CompletePendingLogin(ctx, pendingID)
	if err == nil {
		t.Fatal("expected error on second consume")
	}
}
