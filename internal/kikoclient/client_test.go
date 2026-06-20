package kikoclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func testServer(t *testing.T, apiKey string, handler http.HandlerFunc) (*Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	c := New(srv.URL, apiKey)
	return c, srv
}

func TestNewDefaults(t *testing.T) {
	c := New("http://example.com", "key")
	if c.base != "http://example.com" {
		t.Fatalf("base = %q", c.base)
	}
	if c.apiKey != "key" {
		t.Fatalf("api key = %q", c.apiKey)
	}
}

func TestNewStripsTrailingSlash(t *testing.T) {
	c := New("http://example.com/", "key")
	if c.base != "http://example.com" {
		t.Fatalf("base = %q", c.base)
	}
}

func TestSummary(t *testing.T) {
	c, _ := testServer(t, "key", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-API-Key") != "key" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if r.URL.Path != "/api/v1/stats/summary" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if r.URL.Query().Get("host") != "example.com" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		json.NewEncoder(w).Encode(Summary{
			Host: "example.com", Hits: 100, Uniques: 50,
			TopPath: "/", TopPathHits: 40,
			Since: "2026-01-01", Until: "2026-01-31",
		})
	})

	ctx := context.Background()
	s, err := c.Summary(ctx, "example.com", "2026-01-01", "2026-01-31")
	if err != nil {
		t.Fatal(err)
	}
	if s.Hits != 100 || s.Uniques != 50 {
		t.Fatalf("summary = %+v", s)
	}
}

func TestSummaryAuthError(t *testing.T) {
	c, _ := testServer(t, "wrong", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})
	_, err := c.Summary(context.Background(), "h", "s", "u")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestTimeline(t *testing.T) {
	c, _ := testServer(t, "", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/stats/timeline" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode([]TimelinePoint{
			{Period: "2026-01-01", Hits: 10, Uniques: 5},
		})
	})
	tl, err := c.Timeline(context.Background(), "h", "s", "u", "day")
	if err != nil {
		t.Fatal(err)
	}
	if len(tl) != 1 || tl[0].Hits != 10 {
		t.Fatalf("timeline = %+v", tl)
	}
}

func TestPaths(t *testing.T) {
	c, _ := testServer(t, "", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]PathRow{
			{Path: "/blog", Hits: 30, Uniques: 15},
		})
	})
	paths, err := c.Paths(context.Background(), "h", "s", "u", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 1 || paths[0].Path != "/blog" {
		t.Fatalf("paths = %+v", paths)
	}
}

func TestRefs(t *testing.T) {
	c, _ := testServer(t, "", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]RefRow{
			{Referrer: "https://x.com", Hits: 20, Uniques: 10},
		})
	})
	refs, err := c.Refs(context.Background(), "h", "s", "u", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(refs) != 1 || refs[0].Referrer != "https://x.com" {
		t.Fatalf("refs = %+v", refs)
	}
}

func TestChannels(t *testing.T) {
	c, _ := testServer(t, "", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]Row{
			{Label: "organic", Hits: 50, Uniques: 25},
		})
	})
	ch, err := c.Channels(context.Background(), "h", "s", "u", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(ch) != 1 || ch[0].Label != "organic" {
		t.Fatalf("channels = %+v", ch)
	}
}

func TestBuildInfo(t *testing.T) {
	c, _ := testServer(t, "", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(BuildInfo{
			Version: "1.0.0", Commit: "abc123",
			BuildDate: "today", Branch: "main",
		})
	})
	bi, err := c.BuildInfo(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if bi.Version != "1.0.0" {
		t.Fatalf("version = %q", bi.Version)
	}
}

func TestBuildInfoError(t *testing.T) {
	c := New("http://127.0.0.1:1", "")
	_, err := c.BuildInfo(context.Background())
	if err == nil {
		t.Fatal("expected connection error")
	}
}

func TestGetJSONInvalidURL(t *testing.T) {
	c := New("http://[::1]:99999", "")
	_, err := c.Summary(context.Background(), "h", "s", "u")
	if err == nil {
		t.Fatal("expected URL parse error")
	}
}

func TestGetJSONNon200(t *testing.T) {
	c, _ := testServer(t, "", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
		w.Write([]byte("short and stout"))
	})
	_, err := c.Summary(context.Background(), "h", "s", "u")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGetJSONBadBody(t *testing.T) {
	c, _ := testServer(t, "", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not json"))
	})
	_, err := c.Summary(context.Background(), "h", "s", "u")
	if err == nil {
		t.Fatal("expected decode error")
	}
}

func TestGetJSONEmptyAPIKey(t *testing.T) {
	c, _ := testServer(t, "", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-API-Key") != "" {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		w.Write([]byte("{}"))
	})
	s, err := c.Summary(context.Background(), "h", "s", "u")
	if err != nil {
		t.Fatal(err)
	}
	if s == nil {
		t.Fatal("expected empty summary")
	}
}
