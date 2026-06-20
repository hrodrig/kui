package ui

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/hrodrig/kui/internal/kikoclient"
)

func formatInt(n int64) string {
	return fmt.Sprintf("%d", n)
}

func emailInputPattern() string {
	return `[^@\s]+@(localhost|[^@\s]+\.[^@\s]+)`
}

func formatFloat(n float64) string {
	return fmt.Sprintf("%.1f", n)
}

func pagesPerVisit(hits, uniques int64) string {
	if uniques <= 0 {
		return "—"
	}
	return formatFloat(float64(hits) / float64(uniques))
}

func refLabel(sh Shell, r kikoclient.RefRow) string {
	if r.Source != "" {
		return r.Source
	}
	if r.Referrer == "" {
		return channelLabel(sh, "direct")
	}
	return r.Referrer
}

func channelLabel(sh Shell, label string) string {
	key := "channel." + strings.ToLower(strings.TrimSpace(label))
	if t := sh.T(key); t != key {
		return t
	}
	return label
}

func parseStatTime(iso string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, iso); err == nil {
		return t.UTC(), nil
	}
	return time.Parse("2006-01-02", iso)
}

func formatStatDate(locale, iso string) string {
	t, err := parseStatTime(iso)
	if err != nil {
		return iso
	}
	switch locale {
	case "es", "pt-br", "de", "fr":
		return t.Format("02.01.2006")
	default:
		return t.Format("Jan 2, 2006")
	}
}

func formatChartDate(sh Shell, period string) string {
	t, err := time.Parse("2006-01-02", period)
	if err != nil {
		return period
	}
	switch sh.Locale {
	case "es":
		return fmt.Sprintf("%d %s", t.Day(), spanishMonth(t.Month()))
	case "pt-br":
		return fmt.Sprintf("%d %s", t.Day(), portugueseMonth(t.Month()))
	case "fr":
		return fmt.Sprintf("%d %s", t.Day(), frenchMonth(t.Month()))
	case "de":
		return fmt.Sprintf("%d. %s", t.Day(), germanMonth(t.Month()))
	default:
		return t.Format("Jan 2")
	}
}

func spanishMonth(m time.Month) string {
	return [...]string{"", "ene", "feb", "mar", "abr", "may", "jun", "jul", "ago", "sep", "oct", "nov", "dic"}[m]
}

func portugueseMonth(m time.Month) string {
	return [...]string{"", "jan", "fev", "mar", "abr", "mai", "jun", "jul", "ago", "set", "out", "nov", "dez"}[m]
}

func frenchMonth(m time.Month) string {
	return [...]string{"", "janv.", "févr.", "mars", "avr.", "mai", "juin", "juil.", "août", "sept.", "oct.", "nov.", "déc."}[m]
}

func germanMonth(m time.Month) string {
	return [...]string{"", "Jan", "Feb", "Mär", "Apr", "Mai", "Jun", "Jul", "Aug", "Sep", "Okt", "Nov", "Dez"}[m]
}

func displayUntil(locale, untilISO string) string {
	t, err := parseStatTime(untilISO)
	if err != nil {
		return untilISO
	}
	// kiko until is exclusive (midnight UTC); show the previous calendar day.
	if t.Hour() == 0 && t.Minute() == 0 && t.Second() == 0 {
		t = t.AddDate(0, 0, -1)
	}
	return formatStatDate(locale, t.Format(time.RFC3339))
}

func pathLabel(p kikoclient.PathRow) string {
	if p.Path != "" {
		return p.Path
	}
	return "/"
}

func toJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "[]"
	}
	return string(b)
}

func timelineLabels(points []kikoclient.TimelinePoint) string {
	labels := make([]string, len(points))
	for i, p := range points {
		labels[i] = p.Period
	}
	return toJSON(labels)
}

func timelineHits(points []kikoclient.TimelinePoint) string {
	vals := make([]int64, len(points))
	for i, p := range points {
		vals[i] = p.Hits
	}
	return toJSON(vals)
}

func timelineUniques(points []kikoclient.TimelinePoint) string {
	vals := make([]int64, len(points))
	for i, p := range points {
		vals[i] = p.Uniques
	}
	return toJSON(vals)
}

func channelLabels(rows []kikoclient.Row) string {
	labels := make([]string, len(rows))
	for i, r := range rows {
		labels[i] = r.Label
	}
	return toJSON(labels)
}

func channelHits(rows []kikoclient.Row) string {
	vals := make([]int64, len(rows))
	for i, r := range rows {
		vals[i] = r.Hits
	}
	return toJSON(vals)
}

func DashboardChartJSON(data DashboardData, sh Shell) string {
	type chartCfg struct {
		Labels   []string `json:"labels"`
		Hits     []int64  `json:"hits"`
		Uniques  []int64  `json:"uniques"`
		ChLabels []string `json:"chLabels"`
		ChHits   []int64  `json:"chHits"`
	}
	cfg := chartCfg{
		Labels:   make([]string, len(data.Timeline)),
		Hits:     make([]int64, len(data.Timeline)),
		Uniques:  make([]int64, len(data.Timeline)),
		ChLabels: make([]string, len(data.Channels)),
		ChHits:   make([]int64, len(data.Channels)),
	}
	for i, p := range data.Timeline {
		cfg.Labels[i] = formatChartDate(sh, p.Period)
		cfg.Hits[i] = p.Hits
		cfg.Uniques[i] = p.Uniques
	}
	for i, c := range data.Channels {
		cfg.ChLabels[i] = channelLabel(sh, c.Label)
		cfg.ChHits[i] = c.Hits
	}
	return toJSON(cfg)
}

func deleteUserBody(sh Shell, email string) string {
	return sh.Tfmt("users.delete_body", map[string]string{"email": email})
}

func reset2FABody(sh Shell, email string) string {
	return sh.Tfmt("users.reset_2fa_body", map[string]string{"email": email})
}

func rangeMeta(sh Shell, since, until string) string {
	return sh.Tfmt("dashboard.range_meta", map[string]string{
		"since": formatStatDate(sh.Locale, since),
		"until": displayUntil(sh.Locale, until),
	})
}

func formTitle(sh Shell, form UserFormData) string {
	if form.IsNew {
		return sh.T("users.new_user")
	}
	return sh.T("users.edit_user")
}
