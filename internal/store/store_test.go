package store_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/hrodrig/kui/internal/auth"
	"github.com/hrodrig/kui/internal/store"
)

func TestStoreUsersAndSessions(t *testing.T) {
	dir := t.TempDir()
	st, err := store.Open(filepath.Join(dir, "kui.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	ctx := context.Background()
	hash, err := auth.HashPassword("admin123")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.CreateUser(ctx, "admin@test", hash, store.RoleAdmin); err != nil {
		t.Fatal(err)
	}

	u, gotHash, err := st.UserByEmail(ctx, "admin@test")
	if err != nil {
		t.Fatal(err)
	}
	if !auth.CheckPassword(gotHash, "admin123") {
		t.Fatal("password hash mismatch")
	}

	if err := st.CreateSession(ctx, "sess1", u.ID, time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	uid, err := st.SessionUserID(ctx, "sess1")
	if err != nil || uid != u.ID {
		t.Fatalf("session user = %d err=%v", uid, err)
	}

	ok, err := st.CanAccessHost(ctx, u, "example.com")
	if err != nil || !ok {
		t.Fatalf("admin should access any host: ok=%v err=%v", ok, err)
	}
}
