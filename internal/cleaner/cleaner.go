// Package cleaner implements the branch selection and deletion logic for
// cleanup-git-branch.
package cleaner

import (
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/bsene/cleanup-git-branch/internal/config"
	"github.com/bsene/cleanup-git-branch/internal/git"
)

// GitClient abstracts the git operations the cleaner needs.
type GitClient interface {
	IsGitRepo() bool
	CurrentBranch() (string, error)
	ListBranches(base string) ([]git.Branch, error)
	DeleteBranch(name string, force bool) error
	PruneRemotes() error
}

// Result describes the outcome for a single branch candidate.
type Result struct {
	Branch  git.Branch
	Reason  string
	Deleted bool
	Error   error
}

// Cleaner orchestrates branch scanning, filtering, and deletion.
type Cleaner struct {
	Client GitClient
	Now    time.Time
}

// NewCleaner returns a Cleaner using the supplied git client.
func NewCleaner(client GitClient) *Cleaner {
	return &Cleaner{Client: client, Now: time.Now()}
}

// Run scans local branches, filters stale ones, and either reports (dry-run)
// or deletes them. It returns per-branch results and a summary error if any.
func (c *Cleaner) Run(cfg *config.Config) ([]Result, error) {
	if !c.Client.IsGitRepo() {
		return nil, fmt.Errorf("not inside a git repository")
	}

	base := cfg.Base
	if base == "" {
		cur, err := c.Client.CurrentBranch()
		if err != nil {
			return nil, fmt.Errorf("unable to determine current branch: %w", err)
		}
		base = cur
	}

	branches, err := c.Client.ListBranches(base)
	if err != nil {
		return nil, fmt.Errorf("listing branches: %w", err)
	}

	cutoff := c.Now.Add(-time.Duration(cfg.AgeDays) * 24 * time.Hour)
	var results []Result
	var errs []string

	for _, b := range branches {
		if b.Current {
			continue
		}
		protected, err := c.isProtected(b.Name, cfg.Exclude)
		if err != nil {
			return nil, err
		}
		if protected {
			continue
		}
		if b.LastCommit.After(cutoff) {
			continue
		}
		if !b.Merged {
			continue
		}

		reason := fmt.Sprintf("last commit %s, merged", b.LastCommit.Format("2006-01-02"))

		res := Result{Branch: b, Reason: reason}
		if cfg.Yes {
			force := b.SquashMerged
			if err := c.Client.DeleteBranch(b.Name, force); err != nil {
				res.Error = err
				errs = append(errs, fmt.Sprintf("%s: %v", b.Name, err))
			} else {
				res.Deleted = true
			}
		}
		results = append(results, res)
	}

	if cfg.Yes && cfg.PruneRemotes {
		if err := c.Client.PruneRemotes(); err != nil {
			errs = append(errs, err.Error())
		}
	}

	if len(errs) > 0 {
		return results, fmt.Errorf("cleanup completed with errors: %s", strings.Join(errs, "; "))
	}
	return results, nil
}

// isProtected reports whether a branch name matches any of the exclude glob
// patterns. An invalid pattern returns an error so misconfigurations are
// surfaced instead of silently ignored.
func (c *Cleaner) isProtected(name string, patterns []string) (bool, error) {
	for _, p := range patterns {
		matched, err := path.Match(p, name)
		if err != nil {
			return false, fmt.Errorf("invalid exclude pattern %q: %w", p, err)
		}
		if matched {
			return true, nil
		}
	}
	return false, nil
}
