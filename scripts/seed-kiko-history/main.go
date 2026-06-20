// Seed kiko SQLite with backdated pageview history for kui dashboard demos.
package main

import (
	"database/sql"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type hitRow struct {
	host, path, referrer, visitor, title, browser, os, channel, source string
	width                                                              int
	at                                                                 time.Time
}

type hostProfile struct {
	name  string
	paths []struct {
		path   string
		title  string
		weight int
	}
	sources []struct {
		referrer string
		channel  string
		source   string
		weight   int
	}
}

func main() {
	dbPath := flag.String("db", envOr("KIKO_DB", "../kiko/data/kiko.db"), "path to kiko SQLite database")
	days := flag.Int("days", envIntOr("SEED_DAYS", 90), "days of history to generate")
	hostsRaw := flag.String("hosts", envOr("KIKO_SEED_HOSTS", "gghstats.com,localhost,kzero.dev"), "comma-separated hosts")
	reset := flag.Bool("reset", false, "delete existing hits for target hosts before seeding")
	flag.Parse()

	hosts := parseHosts(*hostsRaw)
	if len(hosts) == 0 {
		fatal("no hosts")
	}
	if *days < 60 {
		fatal("days must be at least 60 for README captures")
	}

	db, err := sql.Open("sqlite", *dbPath)
	if err != nil {
		fatal("open db: %v", err)
	}
	defer db.Close()

	if *reset {
		for _, h := range hosts {
			if _, err := db.Exec(`DELETE FROM kiko_hits WHERE host = ?`, h); err != nil {
				fatal("reset %s: %v", h, err)
			}
		}
		fmt.Printf("cleared hits for: %s\n", strings.Join(hosts, ", "))
	}

	profiles := buildProfiles(hosts)
	visitors := visitorPool(48)
	rng := rand.New(rand.NewSource(42))
	now := time.Now().UTC().Truncate(24 * time.Hour)

	var rows []hitRow
	for day := *days; day >= 0; day-- {
		dayStart := now.AddDate(0, 0, -day)
		for _, p := range profiles {
			n := hitsForDay(rng, day, *days)
			for i := 0; i < n; i++ {
				rows = append(rows, synthesizeHit(rng, p, visitors, dayStart))
			}
		}
	}

	if err := insertHits(db, rows); err != nil {
		fatal("insert: %v", err)
	}

	fmt.Printf("inserted %d hits across %d days for %s\n", len(rows), *days+1, strings.Join(hosts, ", "))
	for _, h := range hosts {
		var cnt int
		var minAt, maxAt string
		_ = db.QueryRow(`SELECT COUNT(*), MIN(created_at), MAX(created_at) FROM kiko_hits WHERE host = ?`, h).Scan(&cnt, &minAt, &maxAt)
		fmt.Printf("  %s: %d hits (%s .. %s)\n", h, cnt, trimDate(minAt), trimDate(maxAt))
	}
}

func buildProfiles(hosts []string) []hostProfile {
	var out []hostProfile
	for _, h := range hosts {
		switch h {
		case "gghstats.com", "www.gghstats.com":
			out = append(out, hostProfile{
				name: h,
				paths: []struct {
					path, title string
					weight      int
				}{
					{"/", "GGH Stats", 14},
					{"/games/elden-ring", "Elden Ring stats", 22},
					{"/games/hollow-knight", "Hollow Knight", 16},
					{"/leaderboard", "Leaderboard", 18},
					{"/players/me", "My profile", 8},
					{"/blog/patch-1-2", "Patch notes", 10},
					{"/about", "About", 6},
				},
				sources: defaultSources(),
			})
		case "localhost", "127.0.0.1":
			out = append(out, hostProfile{
				name: h,
				paths: []struct {
					path, title string
					weight      int
				}{
					{"/", "Home", 16},
					{"/blog/kiko-launch", "Kiko launch", 14},
					{"/blog/htmx-dashboard", "HTMX dashboard", 12},
					{"/docs/getting-started", "Getting started", 15},
					{"/docs/api", "API reference", 10},
					{"/pricing", "Pricing", 11},
					{"/about", "About", 7},
					{"/contact", "Contact", 5},
				},
				sources: defaultSources(),
			})
		case "kzero.dev", "www.kzero.dev":
			out = append(out, hostProfile{
				name: h,
				paths: []struct {
					path, title string
					weight      int
				}{
					{"/", "kzero", 14},
					{"/docs/install", "Install guide", 18},
					{"/docs/cli", "CLI reference", 14},
					{"/pricing", "Pricing", 10},
					{"/changelog", "Changelog", 12},
					{"/changelog/v2", "v2 release", 9},
					{"/blog/launch", "Launch post", 11},
					{"/status", "Status", 6},
				},
				sources: defaultSources(),
			})
		default:
			out = append(out, hostProfile{
				name: h,
				paths: []struct {
					path, title string
					weight      int
				}{
					{"/", "Home", 20},
					{"/about", "About", 10},
					{"/blog", "Blog", 12},
					{"/docs", "Docs", 14},
					{"/pricing", "Pricing", 8},
				},
				sources: defaultSources(),
			})
		}
	}
	return out
}

func defaultSources() []struct {
	referrer, channel, source string
	weight                    int
} {
	return []struct {
		referrer, channel, source string
		weight                    int
	}{
		{"https://www.google.com/", "organic", "Google", 28},
		{"https://news.ycombinator.com/", "referral", "Hacker News", 8},
		{"https://twitter.com/", "social", "Twitter/X", 14},
		{"https://www.reddit.com/", "social", "Reddit", 12},
		{"https://www.linkedin.com/", "referral", "LinkedIn", 7},
		{"https://github.com/", "referral", "GitHub", 10},
		{"https://dev.to/", "referral", "DEV", 6},
		{"https://www.bing.com/", "organic", "Bing", 5},
		{"", "direct", "", 20},
	}
}

func visitorPool(n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = fmt.Sprintf("seed-visitor-%03d", i+1)
	}
	return out
}

func hitsForDay(rng *rand.Rand, daysAgo, totalDays int) int {
	day := time.Now().UTC().AddDate(0, 0, -daysAgo)
	weekend := 1.0
	if day.Weekday() == time.Saturday || day.Weekday() == time.Sunday {
		weekend = 0.72
	}
	// gentle growth toward present + occasional spikes
	progress := 1.0 - float64(daysAgo)/float64(totalDays)
	growth := 0.55 + progress*0.9
	spike := 1.0
	if rng.Float64() < 0.06 {
		spike = 1.8 + rng.Float64()*0.7
	}
	base := 10.0 + rng.Float64()*18.0
	return int(base * weekend * growth * spike)
}

func synthesizeHit(rng *rand.Rand, p hostProfile, visitors []string, day time.Time) hitRow {
	path := weightedPick(rng, p.paths, func(i int) int { return p.paths[i].weight })
	src := weightedPick(rng, p.sources, func(i int) int { return p.sources[i].weight })

	hour := rng.Intn(24)
	min := rng.Intn(60)
	sec := rng.Intn(60)
	at := day.Add(time.Duration(hour)*time.Hour + time.Duration(min)*time.Minute + time.Duration(sec)*time.Second)

	browsers := [][2]string{
		{"Chrome", "macOS"}, {"Firefox", "Linux"}, {"Safari", "iOS"},
		{"Edge", "Windows"}, {"Chrome", "Android"},
	}
	b := browsers[rng.Intn(len(browsers))]
	widths := []int{1920, 1440, 1366, 390, 412}

	return hitRow{
		host:     p.name,
		path:     path.path,
		title:    path.title,
		referrer: src.referrer,
		visitor:  visitors[rng.Intn(len(visitors))],
		browser:  b[0],
		os:       b[1],
		channel:  src.channel,
		source:   src.source,
		width:    widths[rng.Intn(len(widths))],
		at:       at,
	}
}

func weightedPick[T any](rng *rand.Rand, items []T, weight func(int) int) T {
	total := 0
	for i := range items {
		total += weight(i)
	}
	r := rng.Intn(total)
	for i := range items {
		r -= weight(i)
		if r < 0 {
			return items[i]
		}
	}
	return items[len(items)-1]
}

func insertHits(db *sql.DB, rows []hitRow) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`INSERT INTO kiko_hits (
		host, path, referrer, visitor_hash, screen_width, title,
		browser, os, channel, source,
		utm_source, utm_medium, utm_campaign, utm_term, utm_content, created_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NULL, NULL, NULL, NULL, NULL, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, r := range rows {
		ts := r.at.UTC().Format("2006-01-02T15:04:05.000Z")
		_, err := stmt.Exec(
			r.host, r.path, nullStr(r.referrer), r.visitor, r.width, nullStr(r.title),
			nullStr(r.browser), nullStr(r.os), nullStr(r.channel), nullStr(r.source), ts,
		)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

func nullStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func parseHosts(raw string) []string {
	var out []string
	for _, part := range strings.FieldsFunc(raw, func(r rune) bool { return r == ',' || r == ' ' || r == '\n' }) {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envIntOr(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		var n int
		if _, err := fmt.Sscanf(v, "%d", &n); err == nil {
			return n
		}
	}
	return def
}

func trimDate(iso string) string {
	if len(iso) >= 10 {
		return iso[:10]
	}
	return iso
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "seed-kiko-history: "+format+"\n", args...)
	os.Exit(1)
}
