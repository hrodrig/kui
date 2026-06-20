package i18n

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLoad(t *testing.T) {
	b, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if b == nil {
		t.Fatal("nil bundle")
	}
	if len(b.byLocale) == 0 {
		t.Fatal("no locales loaded")
	}
}

func TestMustLoad(t *testing.T) {
	b := MustLoad()
	if b == nil {
		t.Fatal("nil bundle")
	}
	// second call returns same
	b2 := MustLoad()
	if b2 != b {
		t.Fatal("MustLoad did not return singleton")
	}
}

func TestT(t *testing.T) {
	b, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if s := b.T("en", "common.cancel"); s == "" || strings.Contains(s, "common.cancel") {
		t.Fatalf("en common.cancel = %q", s)
	}
	if s := b.T("es", "common.cancel"); s == "" || strings.Contains(s, "common.cancel") {
		t.Fatalf("es common.cancel = %q", s)
	}
	if s := b.T("missing-locale", "common.cancel"); s == "" || strings.Contains(s, "missing-locale") {
		t.Fatalf("fallback common.cancel = %q", s)
	}
	if s := b.T("en", "nonexistent.key"); s != "nonexistent.key" {
		t.Fatalf("missing key = %q", s)
	}
}

func TestTfmt(t *testing.T) {
	b, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	vars := map[string]string{"email": "test@test"}
	s := b.Tfmt("en", "login.email_title", vars)
	if s == "" || strings.Contains(s, "{{") {
		t.Fatalf("Tfmt = %q", s)
	}
}

func TestKeys(t *testing.T) {
	b, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	en := b.Keys("en")
	if len(en) == 0 {
		t.Fatal("empty en keys")
	}
	es := b.Keys("es")
	if len(es) == 0 {
		t.Fatal("empty es keys")
	}
	if b.Keys("nonexistent") != nil {
		t.Fatal("expected nil for missing locale")
	}
}

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

	t.Run("query param", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/?lang=de", nil)
		if got := ResolveLocale(r, "en", enabled); got != "de" {
			t.Fatalf("query: got %q", got)
		}
	})

	t.Run("cookie", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/", nil)
		r.AddCookie(&http.Cookie{Name: CookieName, Value: "es"})
		if got := ResolveLocale(r, "en", enabled); got != "es" {
			t.Fatalf("cookie: got %q", got)
		}
	})

	t.Run("Accept-Language", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/", nil)
		r.Header.Set("Accept-Language", "fr-CH, fr;q=0.9, en;q=0.8")
		if got := ResolveLocale(r, "en", enabled); got != "fr" {
			t.Fatalf("accept: got %q", got)
		}
	})

	t.Run("unsupported locale falls back", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/?lang=xx", nil)
		if got := ResolveLocale(r, "en", enabled); got != "en" {
			t.Fatalf("unsupported: got %q", got)
		}
	})

	t.Run("no match falls to default", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/", nil)
		if got := ResolveLocale(r, "en", []string{}); got != "en" {
			t.Fatalf("empty enabled: got %q", got)
		}
	})
}

func TestNormalizeLocale(t *testing.T) {
	tests := []struct {
		in  string
		exp string
	}{
		{"", "en"},
		{"en", "en"},
		{"pt-BR", "pt-br"},
		{"pt-br", "pt-br"},
		{"es-ES", "es"},
		{"FR", "fr"},
		{"en,es", "en"},
		{"en;q=0.9", "en"},
	}
	for _, tc := range tests {
		got := NormalizeLocale(tc.in)
		if got != tc.exp {
			t.Errorf("NormalizeLocale(%q) = %q, want %q", tc.in, got, tc.exp)
		}
	}
}

func TestLangAttr(t *testing.T) {
	tests := []struct {
		in  string
		exp string
	}{
		{"en", "en"},
		{"es", "es"},
		{"de", "de"},
		{"fr", "fr"},
		{"pt-br", "pt-BR"},
		{"pt-BR", "pt-BR"},
		{"unknown", "unknown"},
	}
	for _, tc := range tests {
		got := LangAttr(tc.in)
		if got != tc.exp {
			t.Errorf("LangAttr(%q) = %q, want %q", tc.in, got, tc.exp)
		}
	}
}

func TestDefaultEnabledLocales(t *testing.T) {
	l := DefaultEnabledLocales()
	if len(l) != 5 {
		t.Fatalf("expected 5, got %d", len(l))
	}
}

func TestParseEnabledLocales(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		got := ParseEnabledLocales("")
		if len(got) != 5 {
			t.Fatalf("expected defaults, got %v", got)
		}
	})
	t.Run("comma list", func(t *testing.T) {
		got := ParseEnabledLocales("en,es")
		if len(got) != 2 || got[0] != "en" || got[1] != "es" {
			t.Fatalf("got %v", got)
		}
	})
	t.Run("normalizes", func(t *testing.T) {
		got := ParseEnabledLocales("en,PT-BR")
		if len(got) != 2 || got[1] != "pt-br" {
			t.Fatalf("got %v", got)
		}
	})
	t.Run("all empty", func(t *testing.T) {
		got := ParseEnabledLocales(",")
		// "," splits to ["",""] → each normalizes to "en" → ["en","en"]
		if len(got) != 2 || got[0] != "en" {
			t.Fatalf("got %v", got)
		}
	})
}

func TestParseDefaultLocale(t *testing.T) {
	if got := ParseDefaultLocale(""); got != "en" {
		t.Fatalf("empty: %q", got)
	}
	if got := ParseDefaultLocale("de"); got != "de" {
		t.Fatalf("de: %q", got)
	}
	if got := ParseDefaultLocale("pt-BR"); got != "pt-br" {
		t.Fatalf("pt-BR -> pt-br: %q", got)
	}
}

func TestResolveLocaleWithUnsupportedCookie(t *testing.T) {
	// cookie "es" not in enabled ["de","en"] → falls to enabled default
	enabled := []string{"de", "en"}
	r := httptest.NewRequest("GET", "/", nil)
	r.AddCookie(&http.Cookie{Name: CookieName, Value: "es"})
	got := ResolveLocale(r, "en", enabled)
	if got != "en" {
		t.Fatalf("cookie unsupported → default: got %q", got)
	}
}

func TestSetSessionCookie(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.AddCookie(&http.Cookie{Name: CookieName, Value: "fr"})
	r.Header.Set("Accept-Language", "de")
	// cookie takes priority over Accept-Language
	if got := ResolveLocale(r, "en", DefaultEnabledLocales()); got != "fr" {
		t.Fatalf("expected cookie fr, got %q", got)
	}
}
