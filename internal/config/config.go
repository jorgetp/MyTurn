// Package config handles loading, validating, and atomically persisting the
// MyTurn JSON configuration file.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const DateLayout = "2006-01-02"

// Config is the on-disk configuration for a MyTurn group.
type Config struct {
	GroupName string   `json:"group_name"`
	Timezone  string   `json:"timezone"`
	StartDate string   `json:"start_date"`
	Members   []string `json:"members"`
	SkipDates []string `json:"skip_dates,omitempty"`
}

// Load reads and parses the configuration file at path.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	if err := Validate(&cfg); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}
	return &cfg, nil
}

// Save atomically writes cfg to path by writing a temp file and renaming it
// over the original, avoiding partial/corrupted writes.
func Save(path string, cfg *Config) error {
	if err := Validate(cfg); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding config: %w", err)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}

	tmp, err := os.CreateTemp(dir, ".config-*.json.tmp")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath) // no-op once renamed

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("writing temp file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("syncing temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("closing temp file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("renaming temp file: %w", err)
	}
	return nil
}

// Validate checks that cfg is internally consistent and safe to use.
func Validate(cfg *Config) error {
	if cfg.GroupName == "" {
		return fmt.Errorf("group_name must not be empty")
	}
	if len(cfg.Members) == 0 {
		return fmt.Errorf("members must contain at least one person")
	}
	seen := make(map[string]bool, len(cfg.Members))
	for _, m := range cfg.Members {
		if m == "" {
			return fmt.Errorf("member names must not be empty")
		}
		if seen[m] {
			return fmt.Errorf("duplicate member name: %s", m)
		}
		seen[m] = true
	}
	if _, err := time.Parse(DateLayout, cfg.StartDate); err != nil {
		return fmt.Errorf("start_date must be in YYYY-MM-DD format: %w", err)
	}
	if _, err := time.LoadLocation(cfg.Timezone); err != nil {
		return fmt.Errorf("invalid timezone %q: %w", cfg.Timezone, err)
	}
	seenSkipDates := make(map[string]bool, len(cfg.SkipDates))
	for _, date := range cfg.SkipDates {
		if _, err := time.Parse(DateLayout, date); err != nil {
			return fmt.Errorf("invalid skip date %q: %w", date, err)
		}
		if seenSkipDates[date] {
			return fmt.Errorf("duplicate skip date: %s", date)
		}
		seenSkipDates[date] = true
	}
	return nil
}
