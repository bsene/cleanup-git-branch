// Package config defines the CLI flags and runtime configuration for
// cleanup-git-branch.
package config

import (
	"fmt"
	"strings"

	"github.com/spf13/pflag"
)

// Config holds the parsed command-line options.
type Config struct {
	Yes           bool
	AgeDays       int
	Merged        bool
	Base          string
	Exclude       []string
	PruneRemotes  bool
	Verbose       bool
}

// Default protected branch patterns.
var defaultExcludes = []string{"main", "master", "develop", "release/*"}

// Parse reads command-line flags and returns a populated Config.
// It also prints usage and returns an error when --help is requested or
// validation fails.
func Parse(args []string) (*Config, error) {
	fs := pflag.NewFlagSet("cleanup-git-branch", pflag.ContinueOnError)
	fs.SortFlags = false

	cfg := &Config{}
	fs.BoolVarP(&cfg.Yes, "yes", "y", false, "Actually delete branches (default: dry-run)")
	fs.IntVarP(&cfg.AgeDays, "age-days", "a", 30, "Minimum age in days for a branch to be considered stale")
	fs.BoolVarP(&cfg.Merged, "merged", "m", false, "Only consider branches merged into the base branch")
	fs.StringVarP(&cfg.Base, "base", "b", "", "Base branch to check merge status against (default: current branch)")
	fs.StringSliceVarP(&cfg.Exclude, "exclude", "e", defaultExcludes, "Glob patterns protecting branches from deletion")
	fs.BoolVarP(&cfg.PruneRemotes, "prune-remotes", "p", false, "Prune remote-tracking refs after local cleanup")
	fs.BoolVarP(&cfg.Verbose, "verbose", "v", false, "Show per-branch details")

	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: cleanup-git-branch [flags]")
		fmt.Fprintln(fs.Output(), "")
		fmt.Fprintln(fs.Output(), "Safely remove stale local Git branches. By default the tool performs a dry run;")
		fmt.Fprintln(fs.Output(), "pass --yes to actually delete branches.")
		fmt.Fprintln(fs.Output(), "")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	if cfg.AgeDays < 0 {
		return nil, fmt.Errorf("age-days must be >= 0")
	}

	// Merge default protected branches with user patterns so that passing
	// --exclude never unprotects the defaults.
	merged := make([]string, 0, len(defaultExcludes)+len(cfg.Exclude))
	merged = append(merged, defaultExcludes...)
	merged = append(merged, cfg.Exclude...)

	// Deduplicate and clean exclude patterns.
	seen := make(map[string]struct{}, len(merged))
	cleaned := make([]string, 0, len(merged))
	for _, p := range merged {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		cleaned = append(cleaned, p)
	}
	cfg.Exclude = cleaned

	return cfg, nil
}
