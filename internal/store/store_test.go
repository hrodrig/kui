package store_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/hrodrig/kui/internal/auth"
	"github.com/hrodrig/kui/internal/store"
)

func TestCreateUser(t *testing.T) {
	st := openTestDB(t)
	ctx := context.Background()
	hash := mustHash(t, "pw")

	id, err := st.CreateUser(ctx, "a@b", hash, store.RoleUser)
	if err != nil {
		t.Fatal(err)
	}
	if id == 0 {
		t.Fatal("expected non-zero id")
	}

	u, gotHash, err := st.UserByEmail(ctx, "a@b")
	if err != nil {
		t.Fatal(err)
	}
	if u.ID != id {
		t.Fatalf("id mismatch: %d vs %d", u.ID, id)
	}
	if gotHash != hash {
		t.Fatal("hash mismatch")
	}
}

func TestCreateUserDuplicate(t *testing.T) {
	st := openTestDB(t)
	ctx := context.Background()
	hash := mustHash(t, "pw")

	if _, err := st.CreateUser(ctx, "dup@t", hash, store.RoleUser); err != nil {
		t.Fatal(err)
	}
	_, err := st.CreateUser(ctx, "dup@t", hash, store.RoleUser)
	if !errors.Is(err, store.ErrDuplicateEmail) {
		t.Fatalf("expected ErrDuplicateEmail, got %v", err)
	}
}

func TestUserByEmailNotFound(t *testing.T) {
	st := openTestDB(t)
	_, _, err := st.UserByEmail(context.Background(), "no@no")
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestUserByID(t *testing.T) {
	st := openTestDB(t)
	ctx := context.Background()
	id := seedAdmin(t, st, "admin@t")

	u, err := st.UserByID(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if u.Email != "admin@t" {
		t.Fatalf("email = %s", u.Email)
	}
	if u.Role != store.RoleAdmin {
		t.Fatalf("role = %s", u.Role)
	}
	if u.CreatedAt.IsZero() {
		t.Fatal("created_at is zero")
	}
}

func TestUserByIDNotFound(t *testing.T) {
	st := openTestDB(t)
	_, err := st.UserByID(context.Background(), 999)
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestListUsers(t *testing.T) {
	st := openTestDB(t)
	ctx := context.Background()

	seedAdmin(t, st, "admin@t")
	hash := mustHash(t, "pw")
	st.CreateUser(ctx, "user1@t", hash, store.RoleUser)
	st.CreateUser(ctx, "user2@t", hash, store.RoleUser)

	users, err := st.ListUsers(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(users) != 3 {
		t.Fatalf("expected 3 users, got %d", len(users))
	}
}

func TestListUsersWithHosts(t *testing.T) {
	st := openTestDB(t)
	ctx := context.Background()

	adminID := seedAdmin(t, st, "admin@t")
	hash := mustHash(t, "pw")
	userID, _ := st.CreateUser(ctx, "u@t", hash, store.RoleUser)
	st.SetUserHosts(ctx, userID, []string{"h1.com"})

	list, err := st.ListUsersWithHosts(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 users, got %d", len(list))
	}
	// admin has no hosts, user has 1
	for _, u := range list {
		if u.ID == adminID {
			if u.Hosts != nil && len(u.Hosts) > 0 {
				t.Fatal("admin should have no hosts")
			}
		}
		if u.ID == userID {
			if len(u.Hosts) != 1 || u.Hosts[0] != "h1.com" {
				t.Fatalf("user hosts = %v", u.Hosts)
			}
		}
	}
}

func TestUpdateUserRole(t *testing.T) {
	st := openTestDB(t)
	ctx := context.Background()
	id := seedAdmin(t, st, "admin@t")

	if err := st.UpdateUserRole(ctx, id, store.RoleUser); err != nil {
		t.Fatal(err)
	}
	u, err := st.UserByID(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if u.Role != store.RoleUser {
		t.Fatalf("role = %s", u.Role)
	}
}

func TestUpdateUserPassword(t *testing.T) {
	st := openTestDB(t)
	ctx := context.Background()
	id := seedAdmin(t, st, "admin@t")

	newHash, err := auth.HashPassword("newpw")
	if err != nil {
		t.Fatal(err)
	}
	if err := st.UpdateUserPassword(ctx, id, newHash); err != nil {
		t.Fatal(err)
	}
	_, gotHash, err := st.UserByEmail(ctx, "admin@t")
	if err != nil {
		t.Fatal(err)
	}
	if !auth.CheckPassword(gotHash, "newpw") {
		t.Fatal("password not updated")
	}
}

func TestDeleteUser(t *testing.T) {
	st := openTestDB(t)
	ctx := context.Background()
	id1 := seedAdmin(t, st, "admin1@t")
	id2 := seedAdmin(t, st, "admin2@t")

	// delete admin2 — still have admin1
	if err := st.DeleteUser(ctx, id2); err != nil {
		t.Fatal(err)
	}
	_, err := st.UserByID(ctx, id2)
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected deleted, got %v", err)
	}

	// delete last admin → ErrLastAdmin
	err = st.DeleteUser(ctx, id1)
	if !errors.Is(err, store.ErrLastAdmin) {
		t.Fatalf("expected ErrLastAdmin, got %v", err)
	}

	// delete non-existent user
	err = st.DeleteUser(ctx, 999)
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestUserHosts(t *testing.T) {
	st := openTestDB(t)
	ctx := context.Background()
	id := seedAdmin(t, st, "admin@t")

	hosts, err := st.UserHosts(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if len(hosts) != 0 {
		t.Fatalf("expected no hosts, got %v", hosts)
	}

	st.SetUserHosts(ctx, id, []string{"a.com", "b.com"})
	hosts, err = st.UserHosts(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if len(hosts) != 2 || hosts[0] != "a.com" {
		t.Fatalf("hosts = %v", hosts)
	}
}

func TestSetUserHostsEmpty(t *testing.T) {
	st := openTestDB(t)
	ctx := context.Background()
	id := seedAdmin(t, st, "admin@t")

	st.SetUserHosts(ctx, id, []string{"x.com"})
	st.SetUserHosts(ctx, id, nil)
	hosts, _ := st.UserHosts(ctx, id)
	if len(hosts) != 0 {
		t.Fatalf("expected empty after set nil, got %v", hosts)
	}
}

func TestListDistinctHosts(t *testing.T) {
	st := openTestDB(t)
	ctx := context.Background()

	h1 := mustHash(t, "pw")
	id1, _ := st.CreateUser(ctx, "u1@t", h1, store.RoleUser)
	id2, _ := st.CreateUser(ctx, "u2@t", h1, store.RoleUser)
	st.SetUserHosts(ctx, id1, []string{"a.com", "b.com"})
	st.SetUserHosts(ctx, id2, []string{"b.com", "c.com"})

	hosts, err := st.ListDistinctHosts(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(hosts) != 3 {
		t.Fatalf("expected 3 distinct hosts, got %v", hosts)
	}
}

func TestCanAccessHost(t *testing.T) {
	st := openTestDB(t)
	ctx := context.Background()

	adminID := seedAdmin(t, st, "admin@t")
	admin, _ := st.UserByID(ctx, adminID)

	h1 := mustHash(t, "pw")
	userID, _ := st.CreateUser(ctx, "u@t", h1, store.RoleUser)
	user, _ := st.UserByID(ctx, userID)

	ok, err := st.CanAccessHost(ctx, admin, "anything")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("admin should access any host")
	}

	ok, err = st.CanAccessHost(ctx, user, "x.com")
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("user with no hosts should not access x.com")
	}

	st.SetUserHosts(ctx, userID, []string{"x.com"})
	ok2, err := st.CanAccessHost(ctx, user, "x.com")
	if err != nil {
		t.Fatal(err)
	}
	if !ok2 {
		t.Fatal("user should access assigned host")
	}
}

func TestParseRole(t *testing.T) {
	tests := []struct {
		in  string
		ok  bool
		exp store.Role
	}{
		{"admin", true, store.RoleAdmin},
		{"user", true, store.RoleUser},
		{"  admin  ", true, store.RoleAdmin},
		{"", false, ""},
		{"superuser", false, ""},
	}
	for _, tc := range tests {
		got, err := store.ParseRole(tc.in)
		if tc.ok && err != nil {
			t.Errorf("ParseRole(%q): %v", tc.in, err)
		}
		if !tc.ok && err == nil {
			t.Errorf("ParseRole(%q): expected error", tc.in)
		}
		if tc.ok && got != tc.exp {
			t.Errorf("ParseRole(%q) = %s, want %s", tc.in, got, tc.exp)
		}
	}
}

func TestCountUsersAndAdmins(t *testing.T) {
	st := openTestDB(t)
	ctx := context.Background()

	seedAdmin(t, st, "admin@t")
	hash := mustHash(t, "pw")
	st.CreateUser(ctx, "u@t", hash, store.RoleUser)

	n, err := st.CountUsers(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("CountUsers = %d", n)
	}

	na, err := st.CountAdmins(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if na != 1 {
		t.Fatalf("CountAdmins = %d", na)
	}
}

func TestCreateSessionExpired(t *testing.T) {
	st := openTestDB(t)
	ctx := context.Background()
	id := seedAdmin(t, st, "admin@t")

	past := time.Now().Add(-time.Hour)
	if err := st.CreateSession(ctx, "expired", id, past); err != nil {
		t.Fatal(err)
	}
	_, err := st.SessionUserID(ctx, "expired")
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected ErrNoRows for expired session, got %v", err)
	}
}

func TestDeleteUserSessions(t *testing.T) {
	st := openTestDB(t)
	ctx := context.Background()
	id := seedAdmin(t, st, "admin@t")

	future := time.Now().Add(time.Hour)
	st.CreateSession(ctx, "s1", id, future)
	st.CreateSession(ctx, "s2", id, future)
	if err := st.DeleteUserSessions(ctx, id); err != nil {
		t.Fatal(err)
	}
	_, err := st.SessionUserID(ctx, "s1")
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatal("sessions should be deleted")
	}
}

func TestUserTOTP(t *testing.T) {
	st := openTestDB(t)
	ctx := context.Background()
	id := seedAdmin(t, st, "admin@t")

	enabled, err := st.UserTOTPEnabled(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if enabled {
		t.Fatal("expected TOTP disabled initially")
	}

	secret := "JBSWY3DPEHPK3PXP"
	if err := st.SetUserTOTPSecret(ctx, id, secret); err != nil {
		t.Fatal(err)
	}

	got, err := st.UserTOTPSecret(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if got != secret {
		t.Fatalf("secret = %s", got)
	}

	enabled, _ = st.UserTOTPEnabled(ctx, id)
	if !enabled {
		t.Fatal("expected TOTP enabled after set")
	}

	if err := st.ClearUserTOTP(ctx, id); err != nil {
		t.Fatal(err)
	}
	enabled, _ = st.UserTOTPEnabled(ctx, id)
	if enabled {
		t.Fatal("expected TOTP disabled after clear")
	}
}

func TestCreateLoginPending(t *testing.T) {
	st := openTestDB(t)
	ctx := context.Background()
	id := seedAdmin(t, st, "admin@t")

	if err := st.CreateLoginPending(ctx, "pid", id, true, time.Now().Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	p, err := st.LoginPendingByID(ctx, "pid")
	if err != nil {
		t.Fatal(err)
	}
	if p.UserID != id || !p.Remember {
		t.Fatalf("pending = %+v", p)
	}
}

func TestLoginPendingExpired(t *testing.T) {
	st := openTestDB(t)
	ctx := context.Background()
	id := seedAdmin(t, st, "admin@t")

	past := time.Now().Add(-time.Minute)
	if err := st.CreateLoginPending(ctx, "ep", id, false, past); err != nil {
		t.Fatal(err)
	}
	_, err := st.LoginPendingByID(ctx, "ep")
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected ErrNoRows for expired pending, got %v", err)
	}
}

func TestLoginPendingConsume(t *testing.T) {
	st := openTestDB(t)
	ctx := context.Background()
	id := seedAdmin(t, st, "admin@t")

	st.CreateLoginPending(ctx, "con", id, false, time.Now().Add(time.Minute))
	p, err := st.ConsumeLoginPending(ctx, "con")
	if err != nil {
		t.Fatal(err)
	}
	if p.UserID != id {
		t.Fatalf("user id = %d", p.UserID)
	}

	// second consume fails
	_, err = st.ConsumeLoginPending(ctx, "con")
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatal("expected ErrNoRows after consume")
	}
}

func TestDeleteLoginPending(t *testing.T) {
	st := openTestDB(t)
	ctx := context.Background()
	id := seedAdmin(t, st, "admin@t")

	st.CreateLoginPending(ctx, "del", id, false, time.Now().Add(time.Minute))
	if err := st.DeleteLoginPending(ctx, "del"); err != nil {
		t.Fatal(err)
	}
	_, err := st.LoginPendingByID(ctx, "del")
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatal("expected ErrNoRows after delete")
	}
}

func TestTOTPSetupPending(t *testing.T) {
	st := openTestDB(t)
	ctx := context.Background()
	id := seedAdmin(t, st, "admin@t")
	secret := "JBSWY3DPEHPK3PXP"

	if err := st.SaveTOTPSetupPending(ctx, id, secret, time.Now().Add(time.Minute)); err != nil {
		t.Fatal(err)
	}

	got, ok, err := st.PeekTOTPSetupPending(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if !ok || got != secret {
		t.Fatalf("Peek: ok=%v secret=%s", ok, got)
	}

	consumed, err := st.ConsumeTOTPSetupPending(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if consumed != secret {
		t.Fatalf("Consume secret = %s", consumed)
	}

	// second consume fails
	_, err = st.ConsumeTOTPSetupPending(ctx, id)
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatal("expected ErrNoRows after consume")
	}
}

func TestTOTPSetupPendingExpired(t *testing.T) {
	st := openTestDB(t)
	ctx := context.Background()
	id := seedAdmin(t, st, "admin@t")

	past := time.Now().Add(-time.Minute)
	st.SaveTOTPSetupPending(ctx, id, "SECRET", past)

	_, ok, err := st.PeekTOTPSetupPending(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expected expired pending to return ok=false")
	}
}

func TestDeleteTOTPSetupPending(t *testing.T) {
	st := openTestDB(t)
	ctx := context.Background()
	id := seedAdmin(t, st, "admin@t")

	st.SaveTOTPSetupPending(ctx, id, "SECRET", time.Now().Add(time.Minute))
	if err := st.DeleteTOTPSetupPending(ctx, id); err != nil {
		t.Fatal(err)
	}
	_, ok, err := st.PeekTOTPSetupPending(ctx, id)
	if err != nil || ok {
		t.Fatal("expected not found after delete")
	}
}

func TestTOTPSetupPendingNotFound(t *testing.T) {
	st := openTestDB(t)
	_, ok, err := st.PeekTOTPSetupPending(context.Background(), 999)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expected ok=false for unknown user")
	}
}
