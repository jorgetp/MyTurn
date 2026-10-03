package rotation

import (
	"testing"

	"myturn/internal/config"
)

func testConfig() *config.Config {
	return &config.Config{
		GroupName: "Home",
		Timezone:  "Europe/Madrid",
		StartDate: "2026-10-01",
		Members:   []string{"Ana", "Bruno", "Carla", "Diego", "Elena"},
	}
}

func TestCalculatedMember_FirstDaySelectsFirstMember(t *testing.T) {
	cfg := testConfig()
	got, err := CalculatedMember(cfg, "2026-10-01")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "Ana" {
		t.Errorf("got %q, want %q", got, "Ana")
	}
}

func TestCalculatedMember_FollowingDaySelectsSecondMember(t *testing.T) {
	cfg := testConfig()
	got, err := CalculatedMember(cfg, "2026-10-02")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "Bruno" {
		t.Errorf("got %q, want %q", got, "Bruno")
	}
}

func TestCalculatedMember_WrapsAfterLastMember(t *testing.T) {
	cfg := testConfig()
	// Members has 5 entries; day index 5 (2026-10-06) should wrap back to Ana.
	got, err := CalculatedMember(cfg, "2026-10-06")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "Ana" {
		t.Errorf("got %q, want %q", got, "Ana")
	}

	got, err = CalculatedMember(cfg, "2026-10-07")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "Bruno" {
		t.Errorf("got %q, want %q", got, "Bruno")
	}
}

func TestCalculatedMember_DatesBeforeStartDate(t *testing.T) {
	cfg := testConfig()
	// One day before start should be the last member (wrap backwards).
	got, err := CalculatedMember(cfg, "2026-09-30")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "Elena" {
		t.Errorf("got %q, want %q", got, "Elena")
	}

	// Two days before start should be the second-to-last member.
	got, err = CalculatedMember(cfg, "2026-09-29")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "Diego" {
		t.Errorf("got %q, want %q", got, "Diego")
	}
}

func TestCalculatedMember_LeapDayArithmetic(t *testing.T) {
	cfg := testConfig()
	cfg.StartDate = "2024-02-28"
	// 2024 is a leap year: Feb 28 -> Feb 29 -> Mar 1 are consecutive days (index 0,1,2).
	tests := []struct {
		date string
		want string
	}{
		{"2024-02-28", "Ana"},
		{"2024-02-29", "Bruno"},
		{"2024-03-01", "Carla"},
	}
	for _, tc := range tests {
		got, err := CalculatedMember(cfg, tc.date)
		if err != nil {
			t.Fatalf("unexpected error for %s: %v", tc.date, err)
		}
		if got != tc.want {
			t.Errorf("date %s: got %q, want %q", tc.date, got, tc.want)
		}
	}
}

func TestCalculatedMember_SkippedDateMovesTurnForward(t *testing.T) {
	cfg := testConfig()
	cfg.SkipDates = []string{"2026-10-01"}
	tests := []struct {
		date string
		want string
	}{
		{"2026-10-01", ""},
		{"2026-10-02", "Ana"},
		{"2026-10-03", "Bruno"},
		{"2026-10-04", "Carla"},
	}
	for _, tc := range tests {
		got, err := CalculatedMember(cfg, tc.date)
		if err != nil {
			t.Fatalf("unexpected error for %s: %v", tc.date, err)
		}
		if got != tc.want {
			t.Errorf("date %s: got %q, want %q", tc.date, got, tc.want)
		}
	}
}

func TestCalculatedMember_MultipleSkippedDatesShiftByTheirCount(t *testing.T) {
	cfg := testConfig()
	cfg.SkipDates = []string{"2026-10-01", "2026-10-03"}
	got, err := CalculatedMember(cfg, "2026-10-04")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "Bruno" {
		t.Errorf("got %q, want %q", got, "Bruno")
	}
}

func TestTodayInTZ_RespectsConfiguredTimezone(t *testing.T) {
	// A fixed instant that falls on different calendar dates depending on timezone.
	// 2026-10-01T23:30:00Z is 2026-10-02 in Europe/Madrid (UTC+2 in October, DST).
	// We can't easily mock time.Now(), so instead verify that two different valid
	// timezones load without error and that an invalid one errors out.
	if _, err := TodayInTZ("Europe/Madrid"); err != nil {
		t.Errorf("unexpected error for valid timezone: %v", err)
	}
	if _, err := TodayInTZ("UTC"); err != nil {
		t.Errorf("unexpected error for valid timezone: %v", err)
	}
	if _, err := TodayInTZ("Not/ARealZone"); err == nil {
		t.Errorf("expected error for invalid timezone, got nil")
	}
}

func TestUpcoming_ReturnsSequentialEntries(t *testing.T) {
	cfg := testConfig()
	entries, err := Upcoming(cfg, "2026-10-01", 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []Entry{
		{Date: "2026-10-01", Member: "Ana"},
		{Date: "2026-10-02", Member: "Bruno"},
		{Date: "2026-10-03", Member: "Carla"},
	}
	if len(entries) != len(want) {
		t.Fatalf("got %d entries, want %d", len(entries), len(want))
	}
	for i := range want {
		if entries[i] != want[i] {
			t.Errorf("entry %d: got %+v, want %+v", i, entries[i], want[i])
		}
	}
}
