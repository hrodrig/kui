package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hrodrig/kui/internal/auth"
	"github.com/hrodrig/kui/internal/config"
	"github.com/hrodrig/kui/internal/i18n"
	"github.com/hrodrig/kui/internal/log"
	"github.com/hrodrig/kui/internal/store"
)

func newTestCfg() *config.Config {
	level, _ := log.ParseLevel("info")
	return &config.Config{
		Listen:         ":3000",
		LogLevel:       "info",
		DefaultLocale:  "en",
		EnabledLocales: i18n.DefaultEnabledLocales(),
		Session: config.SessionCfg{
			CookieName:    "kui_test",
			TTLHours:      168,
			ShortTTLHours: 8,
		},
		Admin: config.AdminCfg{
			Email:    "admin@test",
			Password: "admin123",
		},
		Kiko: config.KikoCfg{
			URL:    "http://127.0.0.1:1",
			APIKey: "",
		},
		Log: log.New(nil, level),
	}
}

func testServer(t *testing.T) *Server {
	t.Helper()
	dir := t.TempDir()
	st, err := store.Open(filepath.Join(dir, "kui.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	cfg := newTestCfg()
	cfg.Database.Path = filepath.Join(dir, "kui.db")

	srv, err := New(cfg, st)
	if err != nil {
		t.Fatal(err)
	}
	return srv
}

func seededTestServer(t *testing.T) (*Server, string, int64) {
	t.Helper()
	srv := testServer(t)
	ctx := context.Background()
	hash, err := auth.HashPassword("pass")
	if err != nil {
		t.Fatal(err)
	}
	id, err := srv.store.CreateUser(ctx, "user@test.com", hash, store.RoleUser)
	if err != nil {
		t.Fatal(err)
	}
	return srv, "user@test.com", id
}

func TestNewServer(t *testing.T) {
	srv := testServer(t)
	if srv.mux == nil {
		t.Fatal("mux is nil")
	}
	if srv.auth == nil {
		t.Fatal("auth service is nil")
	}
	if srv.kiko == nil {
		t.Fatal("kiko client is nil")
	}
}

func TestGetVersion(t *testing.T) {
	srv := testServer(t)
	r := httptest.NewRequest("GET", "/api/v1/version", nil)
	w := httptest.NewRecorder()
	srv.getVersion(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	ct := w.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Fatalf("content-type = %q", ct)
	}
	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["version"] == "" {
		t.Fatal("version field empty")
	}
}

func TestGetLoginRedirectsWhenAuthenticated(t *testing.T) {
	srv, email, _ := seededTestServer(t)
	ctx := context.Background()
	token, _, err := srv.auth.Login(ctx, email, "pass", false)
	if err != nil {
		t.Fatal(err)
	}

	r := httptest.NewRequest("GET", "/login", nil)
	r.AddCookie(&http.Cookie{Name: "kui_test", Value: token})
	w := httptest.NewRecorder()
	srv.getLogin(w, r)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d", w.Code)
	}
	if loc := w.Header().Get("Location"); loc != "/" {
		t.Fatalf("Location = %q", loc)
	}
}

func TestPostLogin(t *testing.T) {
	srv, email, _ := seededTestServer(t)
	body := strings.NewReader("email=" + email + "&password=pass")
	r := httptest.NewRequest("POST", "/login", body)
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	srv.postLogin(w, r)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d", w.Code)
	}
}

func TestPostLoginInvalidCredentials(t *testing.T) {
	srv, email, _ := seededTestServer(t)
	body := strings.NewReader("email=" + email + "&password=wrong")
	r := httptest.NewRequest("POST", "/login", body)
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	srv.postLogin(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestPostLogout(t *testing.T) {
	srv, email, _ := seededTestServer(t)
	ctx := context.Background()
	token, _, err := srv.auth.Login(ctx, email, "pass", false)
	if err != nil {
		t.Fatal(err)
	}

	r := httptest.NewRequest("POST", "/logout", nil)
	r.AddCookie(&http.Cookie{Name: "kui_test", Value: token})
	w := httptest.NewRecorder()
	srv.postLogout(w, r)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d", w.Code)
	}
}

func TestBootstrapAdmin(t *testing.T) {
	dir := t.TempDir()
	st, err := store.Open(filepath.Join(dir, "kui.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	ctx := context.Background()

	cfg := newTestCfg()
	cfg.Admin.Password = "initpass"

	err = BootstrapAdmin(ctx, st, cfg)
	if err != nil {
		t.Fatal(err)
	}

	// second call should be no-op
	err = BootstrapAdmin(ctx, st, cfg)
	if err != nil {
		t.Fatal(err)
	}

	n, err := st.CountUsers(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("expected 1 user, got %d", n)
	}
}

func TestBootstrapAdminMissingPassword(t *testing.T) {
	dir := t.TempDir()
	st, err := store.Open(filepath.Join(dir, "kui.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	ctx := context.Background()

	err = BootstrapAdmin(ctx, st, &config.Config{Admin: config.AdminCfg{Password: ""}})
	if err == nil {
		t.Fatal("expected error for empty password")
	}
}

func TestParseUserFormNew(t *testing.T) {
	srv := testServer(t)
	sh := srv.pageShell(httptest.NewRequest("GET", "/", nil))

	body := strings.NewReader("email=new@test.com&password=validPW1&role=user&hosts=a.com")
	r := httptest.NewRequest("POST", "/users", body)
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	form, errMsg := srv.parseUserForm(r, true, sh)
	if errMsg != "" {
		t.Fatalf("unexpected error: %s", errMsg)
	}
	if form.Email != "new@test.com" {
		t.Fatalf("email = %q", form.Email)
	}
	if form.Role != store.RoleUser {
		t.Fatalf("role = %s", form.Role)
	}
}

func TestParseUserFormNewInvalidEmail(t *testing.T) {
	srv := testServer(t)
	sh := srv.pageShell(httptest.NewRequest("GET", "/", nil))

	body := strings.NewReader("email=bad&password=validPW1&role=user")
	r := httptest.NewRequest("POST", "/users", body)
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	_, errMsg := srv.parseUserForm(r, true, sh)
	if errMsg == "" {
		t.Fatal("expected validation error")
	}
}

func TestParseUserFormNewWeakPassword(t *testing.T) {
	srv := testServer(t)
	sh := srv.pageShell(httptest.NewRequest("GET", "/", nil))

	body := strings.NewReader("email=a@b.com&password=short&role=user")
	r := httptest.NewRequest("POST", "/users", body)
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	_, errMsg := srv.parseUserForm(r, true, sh)
	if errMsg == "" {
		t.Fatal("expected validation error")
	}
}

func TestParseUserFormEditInvalidRole(t *testing.T) {
	srv := testServer(t)
	sh := srv.pageShell(httptest.NewRequest("GET", "/", nil))

	body := strings.NewReader("role=superadmin")
	r := httptest.NewRequest("POST", "/users", body)
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	_, errMsg := srv.parseUserForm(r, false, sh)
	if errMsg == "" {
		t.Fatal("expected role error")
	}
}

func TestParseUserFormBadRequest(t *testing.T) {
	srv := testServer(t)
	sh := srv.pageShell(httptest.NewRequest("GET", "/", nil))

	r := httptest.NewRequest("POST", "/users", nil)
	_, errMsg := srv.parseUserForm(r, true, sh)
	if errMsg == "" {
		t.Fatal("expected parse error")
	}
}

func TestParseUserFormEditBlankPassword(t *testing.T) {
	srv := testServer(t)
	sh := srv.pageShell(httptest.NewRequest("GET", "/", nil))

	body := strings.NewReader("role=user&password=  ")
	r := httptest.NewRequest("POST", "/users", body)
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	_, errMsg := srv.parseUserForm(r, false, sh)
	if errMsg == "" {
		t.Fatal("expected blank password error")
	}
}

func TestNormalizeLocaleConfig(t *testing.T) {
	cfg := &config.Config{
		DefaultLocale:  "pt-BR",
		EnabledLocales: []string{"en,es"},
	}
	normalizeLocaleConfig(cfg)
	if cfg.DefaultLocale != "pt-br" {
		t.Fatalf("default = %q", cfg.DefaultLocale)
	}
	if len(cfg.EnabledLocales) != 2 {
		t.Fatalf("expected 2 locales, got %v", cfg.EnabledLocales)
	}
}

func TestNormalizeLocaleConfigEmpty(t *testing.T) {
	cfg := &config.Config{
		DefaultLocale:  "",
		EnabledLocales: nil,
	}
	normalizeLocaleConfig(cfg)
	if cfg.DefaultLocale != "en" {
		t.Fatalf("default = %q", cfg.DefaultLocale)
	}
	if len(cfg.EnabledLocales) != 5 {
		t.Fatalf("expected defaults, got %v", cfg.EnabledLocales)
	}
}

func TestMaybeSetLocaleCookie(t *testing.T) {
	srv := testServer(t)

	r := httptest.NewRequest("GET", "/?lang=de", nil)
	w := httptest.NewRecorder()
	srv.maybeSetLocaleCookie(w, r)

	c := w.Header().Get("Set-Cookie")
	if c == "" {
		t.Fatal("expected Set-Cookie")
	}
}

func TestMaybeSetLocaleCookieNoop(t *testing.T) {
	srv := testServer(t)

	r := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	srv.maybeSetLocaleCookie(w, r)

	if w.Header().Get("Set-Cookie") != "" {
		t.Fatal("unexpected Set-Cookie without ?lang=")
	}
}

func TestPageShell(t *testing.T) {
	srv := testServer(t)
	r := httptest.NewRequest("GET", "/", nil)
	sh := srv.pageShell(r)

	if sh.I18n.Locale != "en" {
		t.Fatalf("locale = %q", sh.I18n.Locale)
	}
	if sh.JSI18n == "" {
		t.Fatal("JSI18n empty")
	}
}

func TestJSI18nPayload(t *testing.T) {
	payload := jsI18nPayload("en")
	if _, ok := payload["chart.page_views"]; !ok {
		t.Fatal("missing chart.page_views key")
	}
}

func TestLocaleSwitchURL(t *testing.T) {
	r := httptest.NewRequest("GET", "/users?page=1", nil)
	u := localeSwitchURL(r, "es")
	if !strings.Contains(u, "lang=es") {
		t.Fatalf("url = %q", u)
	}
}

func TestResolveKikoVersionEmptyWhenOffline(t *testing.T) {
	srv := testServer(t)
	v := srv.resolveKikoVersion()
	if v != "" {
		t.Fatalf("expected empty version when kiko unreachable, got %q", v)
	}
}

func TestDateRangeDefaults(t *testing.T) {
	since, until, key := dateRange("")
	if since == "" || until == "" {
		t.Fatal("empty date range")
	}
	if key != "7d" {
		t.Fatalf("key = %q", key)
	}
}

func TestDateRange30d(t *testing.T) {
	since, until, key := dateRange("30d")
	if key != "30d" {
		t.Fatalf("key = %q", key)
	}
	if since == "" || until == "" {
		t.Fatal("empty date range")
	}
}

func TestDateRange90d(t *testing.T) {
	_, _, key := dateRange("90d")
	if key != "90d" {
		t.Fatalf("key = %q", key)
	}
}

func TestHandlerReturnsMux(t *testing.T) {
	srv := testServer(t)
	h := srv.Handler()
	if h == nil {
		t.Fatal("handler is nil")
	}
}

func TestChainMiddlewares(t *testing.T) {
	called := 0
	mw1 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called++
			next.ServeHTTP(w, r)
		})
	}
	mw2 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called++
			next.ServeHTTP(w, r)
		})
	}
	final := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called++
	})
	chained := chain(mw1, mw2)
	chained(final).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
	if called != 3 {
		t.Fatalf("expected 3 calls, got %d", called)
	}
}

func TestAllowedHostsAdminNoHosts(t *testing.T) {
	srv := testServer(t)
	ctx := context.Background()

	// admin with no hosts assigned → ListDistinctHosts → "localhost"
	hash, _ := auth.HashPassword("pw")
	adminID, _ := srv.store.CreateUser(ctx, "admin@t", hash, store.RoleAdmin)
	admin, _ := srv.store.UserByID(ctx, adminID)

	hosts, err := srv.allowedHosts(ctx, &admin)
	if err != nil {
		t.Fatal(err)
	}
	if len(hosts) != 1 || hosts[0] != "localhost" {
		t.Fatalf("expected [localhost], got %v", hosts)
	}
}

func TestAllowedHostsAdminWithHosts(t *testing.T) {
	srv := testServer(t)
	ctx := context.Background()

	hash, _ := auth.HashPassword("pw")
	adminID, _ := srv.store.CreateUser(ctx, "admin@h", hash, store.RoleAdmin)
	srv.store.SetUserHosts(ctx, adminID, []string{"a.com", "b.com"})
	admin, _ := srv.store.UserByID(ctx, adminID)

	hosts, err := srv.allowedHosts(ctx, &admin)
	if err != nil {
		t.Fatal(err)
	}
	if len(hosts) != 2 {
		t.Fatalf("expected 2 hosts, got %v", hosts)
	}
}

func TestAllowedHostsRegularUser(t *testing.T) {
	srv := testServer(t)
	ctx := context.Background()

	hash, _ := auth.HashPassword("pw")
	userID, _ := srv.store.CreateUser(ctx, "u@t", hash, store.RoleUser)
	srv.store.SetUserHosts(ctx, userID, []string{"x.com"})
	user, _ := srv.store.UserByID(ctx, userID)

	hosts, err := srv.allowedHosts(ctx, &user)
	if err != nil {
		t.Fatal(err)
	}
	if len(hosts) != 1 || hosts[0] != "x.com" {
		t.Fatalf("expected [x.com], got %v", hosts)
	}
}

func TestBindPageLocale(t *testing.T) {
	srv := testServer(t)
	r := httptest.NewRequest("GET", "/?lang=es", nil)
	loc := bindPageLocale(r, srv.cfg)
	if loc.Locale != "es" {
		t.Fatalf("locale = %q", loc.Locale)
	}
}

func TestLocaleLinks(t *testing.T) {
	srv := testServer(t)
	r := httptest.NewRequest("GET", "/", nil)
	links := srv.localeLinks(r, "en")
	if len(links) == 0 {
		t.Fatal("no locale links")
	}
}

func TestGetLogin2FA(t *testing.T) {
	srv, _, _ := seededTestServer(t)
	r := httptest.NewRequest("GET", "/login/2fa", nil)
	w := httptest.NewRecorder()
	srv.getLogin2FA(w, r)
	// No pending cookie → redirect to /login
	if w.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d", w.Code)
	}
}

func TestPendingExpiry(t *testing.T) {
	exp := pendingExpiry()
	if exp.Before(time.Now()) {
		t.Fatal("pending expiry should be in the future")
	}
}

func TestGetUsersRequiresAuth(t *testing.T) {
	srv := testServer(t)
	r := httptest.NewRequest("GET", "/users", nil)
	w := httptest.NewRecorder()
	srv.mux.ServeHTTP(w, r)
	// no auth → redirect to /login
	if w.Code != http.StatusSeeOther {
		t.Fatalf("expected redirect, got %d", w.Code)
	}
}

func TestGetUserNewRequiresAuth(t *testing.T) {
	srv := testServer(t)
	r := httptest.NewRequest("GET", "/users/new", nil)
	w := httptest.NewRecorder()
	srv.mux.ServeHTTP(w, r)
	if w.Code != http.StatusSeeOther {
		t.Fatalf("expected redirect, got %d", w.Code)
	}
}

func TestStaticFileServer(t *testing.T) {
	srv := testServer(t)
	r := httptest.NewRequest("GET", "/static/css/kui.css", nil)
	w := httptest.NewRecorder()
	srv.mux.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("static file: %d", w.Code)
	}
}
