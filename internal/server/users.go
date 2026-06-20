package server

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/hrodrig/kui/internal/auth"
	"github.com/hrodrig/kui/internal/store"
	"github.com/hrodrig/kui/internal/ui"
)

func (s *Server) adminRoutes() {
	admin := chain(auth.RequireAuth(s.auth), auth.RequireAdmin)
	s.mux.Handle("GET /users", admin(http.HandlerFunc(s.getUsers)))
	s.mux.Handle("GET /users/new", admin(http.HandlerFunc(s.getUserNew)))
	s.mux.Handle("POST /users", admin(http.HandlerFunc(s.postUserCreate)))
	s.mux.Handle("GET /users/{id}", admin(http.HandlerFunc(s.getUserEdit)))
	s.mux.Handle("POST /users/{id}", admin(http.HandlerFunc(s.postUserUpdate)))
	s.mux.Handle("POST /users/{id}/delete", admin(http.HandlerFunc(s.postUserDelete)))
	s.mux.Handle("POST /users/{id}/2fa/reset", admin(http.HandlerFunc(s.postUser2FAReset)))
}

func (s *Server) getUsers(w http.ResponseWriter, r *http.Request) {
	s.maybeSetLocaleCookie(w, r)
	sh := s.pageShell(r)
	actor, _ := auth.UserFromContext(r.Context())
	users, err := s.store.ListUsersWithHosts(r.Context())
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = ui.UsersPage(sh, actor, users, "").Render(r.Context(), w)
}

func (s *Server) getUserNew(w http.ResponseWriter, r *http.Request) {
	s.maybeSetLocaleCookie(w, r)
	sh := s.pageShell(r)
	actor, _ := auth.UserFromContext(r.Context())
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = ui.UserFormPage(actor, ui.UserFormData{Shell: sh, IsNew: true}).Render(r.Context(), w)
}

func (s *Server) postUserCreate(w http.ResponseWriter, r *http.Request) {
	s.maybeSetLocaleCookie(w, r)
	sh := s.pageShell(r)
	actor, _ := auth.UserFromContext(r.Context())
	form, errMsg := s.parseUserForm(r, true, sh)
	if errMsg != "" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusBadRequest)
		_ = ui.UserFormPage(actor, form).Render(r.Context(), w)
		return
	}

	hash, err := auth.HashPassword(form.Password)
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	id, err := s.store.CreateUser(r.Context(), form.Email, hash, form.Role)
	if errors.Is(err, store.ErrDuplicateEmail) {
		form.Error = sh.T("error.email_exists")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusBadRequest)
		_ = ui.UserFormPage(actor, form).Render(r.Context(), w)
		return
	}
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	if err := s.store.SetUserHosts(r.Context(), id, store.ParseHostsText(form.HostsText)); err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/users", http.StatusSeeOther)
}

func (s *Server) getUserEdit(w http.ResponseWriter, r *http.Request) {
	s.maybeSetLocaleCookie(w, r)
	sh := s.pageShell(r)
	actor, _ := auth.UserFromContext(r.Context())
	id, err := userIDParam(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	u, err := s.store.UserByID(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	hosts, _ := s.store.UserHosts(r.Context(), id)
	totpOn, _ := s.store.UserTOTPEnabled(r.Context(), id)
	form := ui.UserFormData{
		Shell:       sh,
		ID:          u.ID,
		Email:       u.Email,
		Role:        u.Role,
		HostsText:   strings.Join(hosts, "\n"),
		TOTPEnabled: totpOn,
	}
	if r.URL.Query().Get("ok") == "2fa_reset" {
		form.Success = sh.T("users.reset_2fa_success")
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = ui.UserFormPage(actor, form).Render(r.Context(), w)
}

func (s *Server) postUserUpdate(w http.ResponseWriter, r *http.Request) {
	s.maybeSetLocaleCookie(w, r)
	sh := s.pageShell(r)
	actor, _ := auth.UserFromContext(r.Context())
	id, err := userIDParam(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	existing, err := s.store.UserByID(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	form, errMsg := s.parseUserForm(r, false, sh)
	form.ID = id
	form.Email = existing.Email
	if errMsg != "" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusBadRequest)
		_ = ui.UserFormPage(actor, form).Render(r.Context(), w)
		return
	}

	if existing.Role == store.RoleAdmin && form.Role == store.RoleUser {
		if err := s.ensureNotLastAdmin(r, id); err != nil {
			form.Error = sh.T("error.last_admin")
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusBadRequest)
			_ = ui.UserFormPage(actor, form).Render(r.Context(), w)
			return
		}
	}

	if err := s.store.UpdateUserRole(r.Context(), id, form.Role); err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	if form.Password != "" {
		hash, err := auth.HashPassword(form.Password)
		if err != nil {
			http.Error(w, "server error", http.StatusInternalServerError)
			return
		}
		if err := s.store.UpdateUserPassword(r.Context(), id, hash); err != nil {
			http.Error(w, "server error", http.StatusInternalServerError)
			return
		}
		_ = s.store.DeleteUserSessions(r.Context(), id)
	}
	if err := s.store.SetUserHosts(r.Context(), id, store.ParseHostsText(form.HostsText)); err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/users", http.StatusSeeOther)
}

func (s *Server) postUserDelete(w http.ResponseWriter, r *http.Request) {
	s.maybeSetLocaleCookie(w, r)
	sh := s.pageShell(r)
	actor, _ := auth.UserFromContext(r.Context())
	id, err := userIDParam(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if actor.ID == id {
		users, _ := s.store.ListUsersWithHosts(r.Context())
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusBadRequest)
		_ = ui.UsersPage(sh, actor, users, sh.T("error.cannot_delete_self")).Render(r.Context(), w)
		return
	}
	if err := s.store.DeleteUser(r.Context(), id); err != nil {
		users, _ := s.store.ListUsersWithHosts(r.Context())
		msg := sh.T("error.could_not_delete")
		if errors.Is(err, store.ErrLastAdmin) {
			msg = sh.T("error.cannot_delete_last_admin")
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusBadRequest)
		_ = ui.UsersPage(sh, actor, users, msg).Render(r.Context(), w)
		return
	}
	http.Redirect(w, r, "/users", http.StatusSeeOther)
}

func (s *Server) postUser2FAReset(w http.ResponseWriter, r *http.Request) {
	actor, _ := auth.UserFromContext(r.Context())
	id, err := userIDParam(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if actor.ID == id {
		http.Redirect(w, r, "/account/2fa", http.StatusSeeOther)
		return
	}
	enabled, err := s.store.UserTOTPEnabled(r.Context(), id)
	if err != nil || !enabled {
		http.Redirect(w, r, userEditPath(id), http.StatusSeeOther)
		return
	}
	if err := s.store.ClearUserTOTP(r.Context(), id); err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	_ = s.store.DeleteTOTPSetupPending(r.Context(), id)
	_ = s.store.DeleteUserSessions(r.Context(), id)
	http.Redirect(w, r, userEditPath(id)+"?ok=2fa_reset", http.StatusSeeOther)
}

func userEditPath(id int64) string {
	return fmt.Sprintf("/users/%d", id)
}

func (s *Server) parseUserForm(r *http.Request, isNew bool, sh ui.Shell) (ui.UserFormData, string) {
	if err := r.ParseForm(); err != nil {
		return ui.UserFormData{Shell: sh}, sh.T("error.invalid_form")
	}
	rawPassword := r.FormValue("password")
	form := ui.UserFormData{
		Shell:     sh,
		Email:     strings.TrimSpace(r.FormValue("email")),
		Password:  auth.TrimPassword(rawPassword),
		HostsText: r.FormValue("hosts"),
		IsNew:     isNew,
	}
	role, err := store.ParseRole(r.FormValue("role"))
	if err != nil {
		form.Error = sh.T("error.invalid_role")
		return form, form.Error
	}
	form.Role = role

	if isNew {
		if !auth.ValidEmail(form.Email) {
			form.Error = sh.T("error.invalid_email")
			return form, form.Error
		}
		if !auth.ValidPassword(form.Password) {
			form.Error = sh.T("error.password_policy")
			return form, form.Error
		}
	} else {
		if rawPassword != "" && form.Password == "" {
			form.Error = sh.T("error.password_blank")
			return form, form.Error
		}
		if form.Password != "" && !auth.ValidPassword(form.Password) {
			form.Error = sh.T("error.password_policy")
			return form, form.Error
		}
	}
	return form, ""
}

func (s *Server) ensureNotLastAdmin(r *http.Request, id int64) error {
	u, err := s.store.UserByID(r.Context(), id)
	if err != nil {
		return err
	}
	if u.Role != store.RoleAdmin {
		return nil
	}
	n, err := s.store.CountAdmins(r.Context())
	if err != nil {
		return err
	}
	if n <= 1 {
		return store.ErrLastAdmin
	}
	return nil
}

func userIDParam(r *http.Request) (int64, error) {
	return strconv.ParseInt(r.PathValue("id"), 10, 64)
}
