// Package rotation implements the deterministic daily rotation logic.
package rotation

import (
	"fmt"
	"time"

	"myturn/internal/config"
)

// Entry is a single day's rotation result.
type Entry struct {
	Date   string `json:"date"`
	Member string `json:"member"`
}

// TodayInTZ returns today's date (YYYY-MM-DD) as observed in the given IANA
// timezone, not the server's local time or UTC.
func TodayInTZ(tz string) (string, error) {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return "", fmt.Errorf("invalid timezone %q: %w", tz, err)
	}
	return time.Now().In(loc).Format(config.DateLayout), nil
}

// daysBetween returns the number of whole days from start to date, treating
// both as calendar dates independent of time-of-day or timezone.
func daysBetween(start, date time.Time) int {
	start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.UTC)
	date = time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
	return int(date.Sub(start).Hours() / 24)
}

// CalculatedMember returns the scheduled member for dateStr after applying
// all persisted skipped days.
func CalculatedMember(cfg *config.Config, dateStr string) (string, error) {
	start, err := time.Parse(config.DateLayout, cfg.StartDate)
	if err != nil {
		return "", fmt.Errorf("invalid start_date: %w", err)
	}
	date, err := time.Parse(config.DateLayout, dateStr)
	if err != nil {
		return "", fmt.Errorf("invalid date %q: %w", dateStr, err)
	}
	n := len(cfg.Members)
	if n == 0 {
		return "", fmt.Errorf("no members configured")
	}
	days := daysBetween(start, date)
	skipped := 0
	isSkipped := false
	for _, skippedDate := range cfg.SkipDates {
		parsed, err := time.Parse(config.DateLayout, skippedDate)
		if err != nil {
			return "", fmt.Errorf("invalid skip date %q: %w", skippedDate, err)
		}
		if !parsed.After(date) {
			skipped++
		}
		if parsed.Equal(date) {
			isSkipped = true
		}
	}
	if isSkipped {
		return "", nil
	}
	idx := (((days - skipped) % n) + n) % n
	return cfg.Members[idx], nil
}

// Upcoming returns n consecutive entries starting at fromDate (inclusive).
func Upcoming(cfg *config.Config, fromDate string, n int) ([]Entry, error) {
	start, err := time.Parse(config.DateLayout, fromDate)
	if err != nil {
		return nil, fmt.Errorf("invalid date %q: %w", fromDate, err)
	}
	entries := make([]Entry, 0, n)
	for i := 0; i < n; i++ {
		d := start.AddDate(0, 0, i).Format(config.DateLayout)
		member, err := CalculatedMember(cfg, d)
		if err != nil {
			return nil, err
		}
		entries = append(entries, Entry{Date: d, Member: member})
	}
	return entries, nil
}
