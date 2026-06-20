package i18n

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLoadAndKeyParity(t *testing.T) {
	b, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	en := b.Keys("en")
	for _, loc := range []string{"es", "fr", "de", "pt-br"} {
		if missing := diffKeys(en, b.Keys(loc)); len(missing) > 0 {
			t.Errorf("%s missing keys: %v", loc, missing)
		}
	}
}

func diffKeys(want, have []string) []string {
	set := make(map[string]struct{}, len(have))
	for _, k := range have {
		set[k] = struct{}{}
	}
	var missing []string
	for _, k := range want {
		if _, ok := set[k]; !ok {
			missing = append(missing, k)
		}
	}
	return missing
}

func TestResolveLocale(t *testing.T) {
	enabled := DefaultEnabledLocales()
	r := httptest.NewRequest("GET", "/?lang=de", nil)
	if got := ResolveLocale(r, "en", enabled); got != "de" {
		t.Fatalf("query: got %q", got)
	}
	r = httptest.NewRequest("GET", "/", nil)
	r.AddCookie(&http.Cookie{Name: CookieName, Value: "es"})
	if got := ResolveLocale(r, "en", enabled); got != "es" {
		t.Fatalf("cookie: got %q", got)
	}
	r = httptest.NewRequest("GET", "/?lang=xx", nil)
	if got := ResolveLocale(r, "en", enabled); got != "en" {
		t.Fatalf("unsupported: got %q", got)
	}
}

func TestNormalizeLocale(t *testing.T) {
	if NormalizeLocale("pt-BR") != "pt-br" {
		t.Fatal("pt-br")
	}
	if NormalizeLocale("es-ES") != "es" {
		t.Fatal("es")
	}
}
