package config

import (
	"errors"
	"testing"

	"github.com/spf13/pflag"
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
	cfg, err := Parse([]string{"--yes", "--age-days", "60", "--exclude", "main,foo/*"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.Yes {
		t.Error("expected Yes true")
	}
	if cfg.AgeDays != 60 {
		t.Errorf("expected age-days 60, got %d", cfg.AgeDays)
	}
	if len(cfg.Exclude) != 5 || cfg.Exclude[0] != "main" || cfg.Exclude[1] != "master" || cfg.Exclude[2] != "develop" || cfg.Exclude[3] != "release/*" || cfg.Exclude[4] != "foo/*" {
		t.Errorf("unexpected excludes: %v", cfg.Exclude)
	}
}

func TestParseNegativeAge(t *testing.T) {
	_, err := Parse([]string{"--age-days", "-1"})
	if err == nil {
		t.Fatal("expected error for negative age-days")
	}
}

func TestParseAgeZeroAllowed(t *testing.T) {
	cfg, err := Parse([]string{"--age-days", "0"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.AgeDays != 0 {
		t.Errorf("expected age-days 0, got %d", cfg.AgeDays)
	}
}

func TestParseDefaultExcludes(t *testing.T) {
	cfg, err := Parse([]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"main", "master", "develop", "release/*"}
	if len(cfg.Exclude) != len(want) {
		t.Fatalf("expected excludes %v, got %v", want, cfg.Exclude)
	}
	for i, v := range want {
		if cfg.Exclude[i] != v {
			t.Errorf("expected exclude[%d] %q, got %q", i, v, cfg.Exclude[i])
		}
	}
}

func TestParseAllFlags(t *testing.T) {
	cfg, err := Parse([]string{
		"--yes",
		"--age-days", "60",
		"--base", "develop",
		"--exclude", "main,feature/*",
		"--prune-remotes",
		"--verbose",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.Yes {
		t.Error("expected Yes true")
	}
	if cfg.AgeDays != 60 {
		t.Errorf("expected age-days 60, got %d", cfg.AgeDays)
	}
	if cfg.Base != "develop" {
		t.Errorf("expected base develop, got %q", cfg.Base)
	}
	if !cfg.PruneRemotes {
		t.Error("expected PruneRemotes true")
	}
	if !cfg.Verbose {
		t.Error("expected Verbose true")
	}
	if len(cfg.Exclude) != 5 || cfg.Exclude[0] != "main" || cfg.Exclude[1] != "master" || cfg.Exclude[2] != "develop" || cfg.Exclude[3] != "release/*" || cfg.Exclude[4] != "feature/*" {
		t.Errorf("unexpected excludes: %v", cfg.Exclude)
	}
}

func TestParseShortFlags(t *testing.T) {
	cfg, err := Parse([]string{
		"-y", "-a", "45", "-b", "main", "-e", "foo/*", "-p", "-v",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.Yes || cfg.AgeDays != 45 || cfg.Base != "main" || !cfg.PruneRemotes || !cfg.Verbose {
		t.Errorf("unexpected config: %+v", cfg)
	}
	if len(cfg.Exclude) != 5 || cfg.Exclude[0] != "main" || cfg.Exclude[1] != "master" || cfg.Exclude[2] != "develop" || cfg.Exclude[3] != "release/*" || cfg.Exclude[4] != "foo/*" {
		t.Errorf("unexpected excludes: %v", cfg.Exclude)
	}
}

func TestParseExcludeCleaning(t *testing.T) {
	cfg, err := Parse([]string{"--exclude", " main , main ,foo/* "})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"main", "master", "develop", "release/*", "foo/*"}
	if len(cfg.Exclude) != len(want) {
		t.Fatalf("expected excludes %v, got %v", want, cfg.Exclude)
	}
	for i, v := range want {
		if cfg.Exclude[i] != v {
			t.Errorf("expected exclude[%d] %q, got %q", i, v, cfg.Exclude[i])
		}
	}
}

func TestParseHelp(t *testing.T) {
	_, err := Parse([]string{"--help"})
	if err == nil {
		t.Fatal("expected error for --help")
	}
	if !errors.Is(err, pflag.ErrHelp) {
		t.Errorf("expected pflag.ErrHelp, got %v", err)
	}
}

func TestParseInvalidFlag(t *testing.T) {
	_, err := Parse([]string{"--unknown"})
	if err == nil {
		t.Fatal("expected error for unknown flag")
	}
}
