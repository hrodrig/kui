package server

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"io/fs"
	"net/http"
	"strings"

	"github.com/hrodrig/kui/internal/auth"
	"github.com/hrodrig/kui/internal/config"
	"github.com/hrodrig/kui/internal/i18n"
	"github.com/hrodrig/kui/internal/kikoclient"
	"github.com/hrodrig/kui/internal/log"
	"github.com/hrodrig/kui/internal/store"
	"github.com/hrodrig/kui/internal/ui"
)

//go:embed static
var staticEmbed embed.FS

type Server struct {
	cfg    *config.Config
	store  *store.Store
	auth   *auth.Service
	kiko   *kikoclient.Client
	mux    *http.ServeMux
	log    *log.Logger
	static fs.FS
}

func New(cfg *config.Config, st *store.Store) (*Server, error) {
	static, err := fs.Sub(staticEmbed, "static")
	if err != nil {
		return nil, err
	}
	s := &Server{
		cfg:    cfg,
		store:  st,
		auth:   auth.NewService(st, cfg.Session),
		kiko:   kikoclient.New(cfg.Kiko.URL, cfg.Kiko.APIKey),
		mux:    http.NewServeMux(),
		log:    cfg.Log,
		static: static,
	}
	normalizeLocaleConfig(cfg)
	i18n.MustLoad()
	ui.SetKikoVersion(s.resolveKikoVersion())
	s.routes()
	return s, nil
}

func (s *Server) resolveKikoVersion() string {
	info, err := s.kiko.BuildInfo(context.Background())
	if err != nil {
		s.log.Warn("kiko version: %v", err)
		return ""
	}
	return info.Version
}

func (s *Server) Handler() http.Handler {
	return s.mux
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET "+VersionPath, s.getVersion)
	s.mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(s.static))))
	s.mux.HandleFunc("GET /login", s.getLogin)
	s.mux.HandleFunc("POST /login", s.postLogin)
	s.mux.HandleFunc("GET /login/2fa", s.getLogin2FA)
	s.mux.HandleFunc("POST /login/2fa", s.postLogin2FA)
	s.mux.HandleFunc("POST /logout", s.postLogout)
	s.mux.Handle("GET /{$}", auth.RequireAuth(s.auth)(http.HandlerFunc(s.getDashboard)))
	s.accountRoutes()
	s.adminRoutes()
}

func chain(mw ...func(http.Handler) http.Handler) func(http.Handler) http.Handler {
	return func(final http.Handler) http.Handler {
		h := final
		for i := len(mw) - 1; i >= 0; i-- {
			h = mw[i](h)
		}
		return h
	}
}

func (s *Server) getLogin(w http.ResponseWriter, r *http.Request) {
	if u, err := s.auth.UserFromRequest(r.Context(), r); err == nil && u != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	s.maybeSetLocaleCookie(w, r)
	sh := s.pageShell(r)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = ui.LoginPage(ui.LoginFormData{Shell: sh, Remember: true}).Render(r.Context(), w)
}

func (s *Server) postLogin(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	s.maybeSetLocaleCookie(w, r)
	sh := s.pageShell(r)
	email := strings.TrimSpace(r.FormValue("email"))
	password := auth.TrimPassword(r.FormValue("password"))
	remember := r.FormValue("remember") == "1"
	if !auth.ValidEmail(email) || password == "" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusUnauthorized)
		_ = ui.LoginPage(ui.LoginFormData{Shell: sh, Error: sh.T("error.invalid_credentials"), Email: email, Remember: remember}).Render(r.Context(), w)
		return
	}
	u, needs2FA, err := s.auth.Authenticate(r.Context(), email, password)
	if err != nil {
		msg := sh.T("error.invalid_credentials")
		if errors.Is(err, store.ErrInvalidCredentials) || errors.Is(err, sql.ErrNoRows) {
			msg = sh.T("error.invalid_credentials")
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusUnauthorized)
		_ = ui.LoginPage(ui.LoginFormData{Shell: sh, Error: msg, Email: email, Remember: remember}).Render(r.Context(), w)
		return
	}
	if needs2FA {
		pendingID, err := s.auth.StartPendingLogin(r.Context(), u.ID, remember)
		if err != nil {
			http.Error(w, "server error", http.StatusInternalServerError)
			return
		}
		s.auth.SetPendingCookie(w, pendingID)
		http.Redirect(w, r, "/login/2fa", http.StatusSeeOther)
		return
	}
	token, err := s.auth.CreateSession(r.Context(), u.ID, remember)
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	s.auth.SetSessionCookie(w, token, remember)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *Server) postLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(s.auth.CookieName()); err == nil {
		_ = s.auth.Logout(r.Context(), c.Value)
	}
	s.auth.ClearSessionCookie(w)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (s *Server) getDashboard(w http.ResponseWriter, r *http.Request) {
	s.maybeSetLocaleCookie(w, r)
	sh := s.pageShell(r)
	u, _ := auth.UserFromContext(r.Context())
	hosts, err := s.allowedHosts(r.Context(), u)
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	host := strings.TrimSpace(r.URL.Query().Get("host"))
	if host == "" && len(hosts) > 0 {
		host = hosts[0]
	}
	rangeKey := strings.TrimSpace(r.URL.Query().Get("range"))
	since, until, rangeKey := dateRange(rangeKey)

	data := ui.DashboardData{
		Shell: sh,
		Hosts: hosts,
		Host:  host,
		Range: rangeKey,
	}
	if host != "" {
		ok, err := s.store.CanAccessHost(r.Context(), *u, host)
		if err != nil || !ok {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		s.fillDashboardKiko(r.Context(), &data, host, since, until)
	}

	data.ChartJSON = ui.DashboardChartJSON(data, sh)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = ui.DashboardPage(u, data).Render(r.Context(), w)
}

func (s *Server) allowedHosts(ctx context.Context, u *store.User) ([]string, error) {
	if u.Role == store.RoleAdmin {
		hosts, err := s.store.UserHosts(ctx, u.ID)
		if err != nil {
			return nil, err
		}
		if len(hosts) > 0 {
			return hosts, nil
		}
		hosts, err = s.store.ListDistinctHosts(ctx)
		if err != nil {
			return nil, err
		}
		if len(hosts) > 0 {
			return hosts, nil
		}
		return []string{"localhost"}, nil
	}
	hosts, err := s.store.UserHosts(ctx, u.ID)
	if err != nil {
		return nil, err
	}
	return hosts, nil
}

func BootstrapAdmin(ctx context.Context, st *store.Store, cfg *config.Config) error {
	n, err := st.CountUsers(ctx)
	if err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	if cfg.Admin.Password == "" {
		return errors.New("no users exist: set KUI_ADMIN_PASSWORD (or admin.password) to seed the first admin")
	}
	hash, err := auth.HashPassword(cfg.Admin.Password)
	if err != nil {
		return err
	}
	email := cfg.Admin.Email
	if email == "" {
		email = "admin@localhost"
	}
	_, err = st.CreateUser(ctx, email, hash, store.RoleAdmin)
	return err
}
