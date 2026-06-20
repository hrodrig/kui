package server

import (
	"time"
)

// dateRange returns since/until as YYYY-MM-DD for kiko stats queries.
// kiko treats date-only until as midnight UTC (exclusive), so until is the day
// after "today" to include all hits recorded today.
func dateRange(key string) (since, until, normalized string) {
	now := time.Now().UTC()
	until = now.AddDate(0, 0, 1).Format("2006-01-02")
	days := 7
	switch key {
	case "30d":
		days = 30
	case "90d":
		days = 90
	default:
		key = "7d"
	}
	since = now.AddDate(0, 0, -days).Format("2006-01-02")
	return since, until, key
}
