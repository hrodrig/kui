package server

import (
	"testing"
	"time"
)

func TestDateRangeUntilIncludesToday(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC()
	_, until, _ := dateRange("30d")
	untilDay, err := time.Parse("2006-01-02", until)
	if err != nil {
		t.Fatal(err)
	}
	tomorrow := now.AddDate(0, 0, 1)
	if untilDay.Year() != tomorrow.Year() || untilDay.YearDay() != tomorrow.YearDay() {
		t.Fatalf("until = %s, want start of day after today (%s)", until, tomorrow.Format("2006-01-02"))
	}
}
