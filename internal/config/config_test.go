package config

import (
	"testing"
)

func TestParseDefaults(t *testing.T) {
	cfg, err := Parse([]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Yes {
		t.Error("expected dry-run by default")
	}
	if cfg.AgeDays != 30 {
		t.Errorf("expected default age-days 30, got %d", cfg.AgeDays)
	}
	if len(cfg.Exclude) == 0 {
		t.Error("expected default excludes")
	}
}

func TestParseFlags(t *testing.T) {
	cfg, err := Parse([]string{"--yes", "--age-days", "60", "--merged", "--exclude", "main,foo/*"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.Yes {
		t.Error("expected Yes true")
	}
	if cfg.AgeDays != 60 {
		t.Errorf("expected age-days 60, got %d", cfg.AgeDays)
	}
	if !cfg.Merged {
		t.Error("expected Merged true")
	}
	if len(cfg.Exclude) != 2 || cfg.Exclude[0] != "main" || cfg.Exclude[1] != "foo/*" {
		t.Errorf("unexpected excludes: %v", cfg.Exclude)
	}
}

func TestParseNegativeAge(t *testing.T) {
	_, err := Parse([]string{"--age-days", "-1"})
	if err == nil {
		t.Fatal("expected error for negative age-days")
	}
}
