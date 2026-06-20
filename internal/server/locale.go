package server

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"github.com/hrodrig/kui/internal/config"
	"github.com/hrodrig/kui/internal/i18n"
	"github.com/hrodrig/kui/internal/ui"
)

const localeCookieMaxAge = 365 * 24 * 3600

func bindPageLocale(r *http.Request, cfg *config.Config) ui.I18n {
	loc := i18n.ResolveLocale(r, cfg.DefaultLocale, cfg.EnabledLocales)
	bundle := i18n.MustLoad()
	return ui.I18n{
		Locale: loc,
		Lang:   i18n.LangAttr(loc),
		T:      func(key string) string { return bundle.T(loc, key) },
		Tfmt:   func(key string, vars map[string]string) string { return bundle.Tfmt(loc, key, vars) },
	}
}

func (s *Server) maybeSetLocaleCookie(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("lang") == "" {
		return
	}
	loc := i18n.ResolveLocale(r, s.cfg.DefaultLocale, s.cfg.EnabledLocales)
	http.SetCookie(w, &http.Cookie{
		Name:     i18n.CookieName,
		Value:    loc,
		Path:     "/",
		MaxAge:   localeCookieMaxAge,
		SameSite: http.SameSiteLaxMode,
		Secure:   s.cfg.Session.Secure,
	})
}

func (s *Server) localeLinks(r *http.Request, current string) []ui.LocaleLink {
	labels := map[string]string{
		"en": "EN", "es": "ES", "fr": "FR", "de": "DE", "pt-br": "PT",
	}
	var links []ui.LocaleLink
	for _, code := range s.cfg.EnabledLocales {
		code = i18n.NormalizeLocale(code)
		label := labels[code]
		if label == "" {
			label = strings.ToUpper(code)
		}
		links = append(links, ui.LocaleLink{
			Code:   code,
			Label:  label,
			URL:    localeSwitchURL(r, code),
			Active: code == current,
		})
	}
	return links
}

func localeSwitchURL(r *http.Request, lang string) string {
	q := r.URL.Query()
	q.Set("lang", lang)
	return (&url.URL{Path: r.URL.Path, RawQuery: q.Encode()}).String()
}

func (s *Server) pageShell(r *http.Request) ui.Shell {
	loc, links := s.pageLocale(r)
	raw, _ := json.Marshal(jsI18nPayload(loc.Locale))
	return ui.Shell{I18n: loc, Links: links, JSI18n: string(raw)}
}

func (s *Server) pageLocale(r *http.Request) (ui.I18n, []ui.LocaleLink) {
	loc := bindPageLocale(r, s.cfg)
	return loc, s.localeLinks(r, loc.Locale)
}

func jsI18nPayload(locale string) map[string]string {
	bundle := i18n.MustLoad()
	keys := []string{
		"common.theme_light",
		"common.theme_dark",
		"common.show_password",
		"common.hide_password",
		"chart.page_views",
		"chart.uniques",
	}
	out := make(map[string]string, len(keys))
	for _, k := range keys {
		out[k] = bundle.T(locale, k)
	}
	return out
}

func normalizeLocaleConfig(cfg *config.Config) {
	cfg.DefaultLocale = i18n.ParseDefaultLocale(cfg.DefaultLocale)
	if len(cfg.EnabledLocales) == 1 && strings.Contains(cfg.EnabledLocales[0], ",") {
		cfg.EnabledLocales = i18n.ParseEnabledLocales(cfg.EnabledLocales[0])
	} else if len(cfg.EnabledLocales) == 0 {
		cfg.EnabledLocales = i18n.DefaultEnabledLocales()
	} else {
		out := make([]string, 0, len(cfg.EnabledLocales))
		for _, e := range cfg.EnabledLocales {
			out = append(out, i18n.NormalizeLocale(e))
		}
		cfg.EnabledLocales = out
	}
}
