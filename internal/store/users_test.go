package store_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/hrodrig/kui/internal/auth"
	"github.com/hrodrig/kui/internal/store"
)

func openTestDB(t *testing.T) *store.Store {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "kui.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}

func seedAdmin(t *testing.T, st *store.Store, email string) int64 {
	t.Helper()
	hash, err := auth.HashPassword("password123")
	if err != nil {
		t.Fatal(err)
	}
	id, err := st.CreateUser(context.Background(), email, hash, store.RoleAdmin)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func TestUserHostsAndDistinct(t *testing.T) {
	st := openTestDB(t)
	ctx := context.Background()

	id, err := st.CreateUser(ctx, "viewer@test", mustHash(t, "password123"), store.RoleUser)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.SetUserHosts(ctx, id, []string{"a.com", "b.com"}); err != nil {
		t.Fatal(err)
	}
	hosts, err := st.ListDistinctHosts(ctx)
	if err != nil || len(hosts) != 2 {
		t.Fatalf("distinct hosts = %v err=%v", hosts, err)
	}
}

func TestDeleteUserLastAdmin(t *testing.T) {
	st := openTestDB(t)
	ctx := context.Background()
	id := seedAdmin(t, st, "only-admin@test")

	err := st.DeleteUser(ctx, id)
	if !errors.Is(err, store.ErrLastAdmin) {
		t.Fatalf("expected ErrLastAdmin, got %v", err)
	}
}

func TestDuplicateEmail(t *testing.T) {
	st := openTestDB(t)
	ctx := context.Background()
	hash := mustHash(t, "password123")
	if _, err := st.CreateUser(ctx, "dup@test", hash, store.RoleUser); err != nil {
		t.Fatal(err)
	}
	_, err := st.CreateUser(ctx, "dup@test", hash, store.RoleUser)
	if !errors.Is(err, store.ErrDuplicateEmail) {
		t.Fatalf("expected duplicate email, got %v", err)
	}
}

func TestParseHostsText(t *testing.T) {
	got := store.ParseHostsText(" a.com \n b.com, c.com \na.com")
	if len(got) != 3 || got[0] != "a.com" || got[2] != "c.com" {
		t.Fatalf("unexpected hosts: %v", got)
	}
}

func mustHash(t *testing.T, pw string) string {
	t.Helper()
	h, err := auth.HashPassword(pw)
	if err != nil {
		t.Fatal(err)
	}
	return h
}
