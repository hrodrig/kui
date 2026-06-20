package server

import (
	"context"
	"net/http"
	"time"

	"github.com/hrodrig/kui/internal/auth"
	"github.com/hrodrig/kui/internal/store"
	"github.com/hrodrig/kui/internal/ui"
)

func pendingExpiry() time.Time {
	return time.Now().UTC().Add(10 * time.Minute)
}

func (s *Server) accountRoutes() {
	s.mux.Handle("GET /account/2fa", auth.RequireAuth(s.auth)(http.HandlerFunc(s.getAccount2FA)))
	s.mux.Handle("GET /account/2fa/cancel", auth.RequireAuth(s.auth)(http.HandlerFunc(s.getAccount2FACancel)))
	s.mux.Handle("POST /account/2fa/begin", auth.RequireAuth(s.auth)(http.HandlerFunc(s.postAccount2FABegin)))
	s.mux.Handle("POST /account/2fa/confirm", auth.RequireAuth(s.auth)(http.HandlerFunc(s.postAccount2FAConfirm)))
	s.mux.Handle("POST /account/2fa/disable", auth.RequireAuth(s.auth)(http.HandlerFunc(s.postAccount2FADisable)))
}

func (s *Server) getAccount2FA(w http.ResponseWriter, r *http.Request) {
	s.maybeSetLocaleCookie(w, r)
	sh := s.pageShell(r)
	u, _ := auth.UserFromContext(r.Context())
	data, err := s.account2FAData(r.Context(), u, "")
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	data.Shell = sh
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = ui.Account2FAPage(u, data).Render(r.Context(), w)
}

func (s *Server) postAccount2FABegin(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFromContext(r.Context())
	enabled, err := s.store.UserTOTPEnabled(r.Context(), u.ID)
	if err != nil || enabled {
		http.Redirect(w, r, "/account/2fa", http.StatusSeeOther)
		return
	}
	secret, _, err := auth.NewTOTPSecret(u.Email)
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	if err := s.store.SaveTOTPSetupPending(r.Context(), u.ID, secret, pendingExpiry()); err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/account/2fa", http.StatusSeeOther)
}

func (s *Server) postAccount2FAConfirm(w http.ResponseWriter, r *http.Request) {
	s.maybeSetLocaleCookie(w, r)
	sh := s.pageShell(r)
	u, _ := auth.UserFromContext(r.Context())
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	code := auth.NormalizeTOTPCode(r.FormValue("code"))
	secret, err := s.store.ConsumeTOTPSetupPending(r.Context(), u.ID)
	if err != nil {
		data, _ := s.account2FAData(r.Context(), u, sh.T("error.setup_expired"))
		data.Shell = sh
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusBadRequest)
		_ = ui.Account2FAPage(u, data).Render(r.Context(), w)
		return
	}
	if !auth.ValidTOTPCode(secret, code) {
		_ = s.store.SaveTOTPSetupPending(r.Context(), u.ID, secret, pendingExpiry())
		data, _ := s.account2FAData(r.Context(), u, sh.T("error.invalid_totp"))
		data.Shell = sh
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusBadRequest)
		_ = ui.Account2FAPage(u, data).Render(r.Context(), w)
		return
	}
	if err := s.store.SetUserTOTPSecret(r.Context(), u.ID, secret); err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/account/2fa", http.StatusSeeOther)
}

func (s *Server) postAccount2FADisable(w http.ResponseWriter, r *http.Request) {
	s.maybeSetLocaleCookie(w, r)
	sh := s.pageShell(r)
	u, _ := auth.UserFromContext(r.Context())
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	code := auth.NormalizeTOTPCode(r.FormValue("code"))
	ok, err := s.auth.VerifyUserTOTP(r.Context(), u.ID, code)
	if err != nil || !ok {
		data, _ := s.account2FAData(r.Context(), u, sh.T("error.invalid_code"))
		data.Shell = sh
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusBadRequest)
		_ = ui.Account2FAPage(u, data).Render(r.Context(), w)
		return
	}
	if err := s.store.ClearUserTOTP(r.Context(), u.ID); err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	_ = s.store.DeleteUserSessions(r.Context(), u.ID)
	s.auth.ClearSessionCookie(w)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (s *Server) getAccount2FACancel(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFromContext(r.Context())
	_ = s.store.DeleteTOTPSetupPending(r.Context(), u.ID)
	http.Redirect(w, r, "/account/2fa", http.StatusSeeOther)
}

func (s *Server) account2FAData(ctx context.Context, u *store.User, errMsg string) (ui.Account2FAData, error) {
	enabled, err := s.store.UserTOTPEnabled(ctx, u.ID)
	if err != nil {
		return ui.Account2FAData{}, err
	}
	data := ui.Account2FAData{Enabled: enabled, Error: errMsg}
	if !enabled {
		secret, ok, err := s.store.PeekTOTPSetupPending(ctx, u.ID)
		if err != nil {
			return ui.Account2FAData{}, err
		}
		if ok {
			data.SetupPending = true
			data.ManualSecret = secret
			data.QRDataURI, err = auth.QRCodeDataURI(auth.TOTPURL(u.Email, secret))
			if err != nil {
				return ui.Account2FAData{}, err
			}
		}
	}
	return data, nil
}

func (s *Server) getLogin2FA(w http.ResponseWriter, r *http.Request) {
	if _, err := s.auth.PendingIDFromRequest(r); err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	s.maybeSetLocaleCookie(w, r)
	sh := s.pageShell(r)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = ui.Login2FAPage(sh, "").Render(r.Context(), w)
}

func (s *Server) postLogin2FA(w http.ResponseWriter, r *http.Request) {
	pendingID, err := s.auth.PendingIDFromRequest(r)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	s.maybeSetLocaleCookie(w, r)
	sh := s.pageShell(r)
	pending, err := s.store.LoginPendingByID(r.Context(), pendingID)
	if err != nil {
		s.auth.ClearPendingCookie(w)
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	ok, err := s.auth.VerifyUserTOTP(r.Context(), pending.UserID, auth.NormalizeTOTPCode(r.FormValue("code")))
	if err != nil || !ok {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusUnauthorized)
		_ = ui.Login2FAPage(sh, sh.T("error.invalid_code")).Render(r.Context(), w)
		return
	}
	_ = s.store.DeleteLoginPending(r.Context(), pendingID)
	token, err := s.auth.CreateSession(r.Context(), pending.UserID, pending.Remember)
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	s.auth.ClearPendingCookie(w)
	s.auth.SetSessionCookie(w, token, pending.Remember)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
