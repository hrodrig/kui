package kikoclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	base   string
	apiKey string
	http   *http.Client
}

func New(baseURL, apiKey string) *Client {
	return &Client{
		base:   strings.TrimRight(baseURL, "/"),
		apiKey: apiKey,
		http:   &http.Client{Timeout: 15 * time.Second},
	}
}

type Summary struct {
	Host        string `json:"host"`
	Hits        int64  `json:"hits"`
	Uniques     int64  `json:"uniques"`
	TopPath     string `json:"top_path"`
	TopPathHits int64  `json:"top_path_hits"`
	Since       string `json:"since"`
	Until       string `json:"until"`
}

type PathRow struct {
	Path    string `json:"path"`
	Title   string `json:"title,omitempty"`
	Hits    int64  `json:"hits"`
	Uniques int64  `json:"uniques"`
}

type RefRow struct {
	Referrer string `json:"referrer"`
	Source   string `json:"source,omitempty"`
	Hits     int64  `json:"hits"`
	Uniques  int64  `json:"uniques"`
}

type Row struct {
	Label   string `json:"label"`
	Hits    int64  `json:"hits"`
	Uniques int64  `json:"uniques"`
}

type TimelinePoint struct {
	Period  string `json:"period"`
	Hits    int64  `json:"hits"`
	Uniques int64  `json:"uniques"`
}

// BuildInfo matches GET /api/v1/version on the kiko server.
type BuildInfo struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildDate string `json:"build_date"`
	Branch    string `json:"branch"`
}

func (c *Client) Summary(ctx context.Context, host, since, until string) (*Summary, error) {
	var out Summary
	if err := c.getJSON(ctx, "/api/v1/stats/summary", map[string]string{
		"host": host, "since": since, "until": until,
	}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) Timeline(ctx context.Context, host, since, until, interval string) ([]TimelinePoint, error) {
	var out []TimelinePoint
	if err := c.getJSON(ctx, "/api/v1/stats/timeline", map[string]string{
		"host": host, "since": since, "until": until, "interval": interval,
	}, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) Paths(ctx context.Context, host, since, until string, limit int) ([]PathRow, error) {
	var out []PathRow
	if err := c.getJSON(ctx, "/api/v1/stats/paths", map[string]string{
		"host": host, "since": since, "until": until, "limit": fmt.Sprintf("%d", limit),
	}, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) Refs(ctx context.Context, host, since, until string, limit int) ([]RefRow, error) {
	var out []RefRow
	if err := c.getJSON(ctx, "/api/v1/stats/refs", map[string]string{
		"host": host, "since": since, "until": until, "limit": fmt.Sprintf("%d", limit),
	}, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) Channels(ctx context.Context, host, since, until string, limit int) ([]Row, error) {
	var out []Row
	if err := c.getJSON(ctx, "/api/v1/stats/channels", map[string]string{
		"host": host, "since": since, "until": until, "limit": fmt.Sprintf("%d", limit),
	}, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// BuildInfo returns kiko build metadata from GET /api/v1/version.
func (c *Client) BuildInfo(ctx context.Context) (BuildInfo, error) {
	var out BuildInfo
	if err := c.getJSON(ctx, "/api/v1/version", nil, &out); err != nil {
		return BuildInfo{}, err
	}
	out.Version = strings.TrimSpace(out.Version)
	return out, nil
}

func (c *Client) getJSON(ctx context.Context, path string, params map[string]string, dest any) error {
	u, err := url.Parse(c.base + path)
	if err != nil {
		return err
	}
	q := u.Query()
	for k, v := range params {
		if v != "" {
			q.Set(k, v)
		}
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return err
	}
	if c.apiKey != "" {
		req.Header.Set("X-API-Key", c.apiKey)
	}

	res, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("kiko %s: %s", res.Status, strings.TrimSpace(string(body)))
	}
	if err := json.Unmarshal(body, dest); err != nil {
		return fmt.Errorf("decode kiko response: %w", err)
	}
	return nil
}
