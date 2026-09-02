// Package git wraps the Git CLI commands used by cleanup-git-branch.
package git

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Branch holds metadata about a local Git branch.
type Branch struct {
	Name      string
	Ref       string
	LastCommit time.Time
	Merged    bool
	Current   bool
}

// Runner abstracts running git commands so tests can stub it out.
type Runner interface {
	Run(args ...string) (string, error)
}

// ExecRunner runs real git commands in the current working directory.
type ExecRunner struct{}

func (r ExecRunner) Run(args ...string) (string, error) {
	// #nosec G204 - argv list, no shell interpreter; caller-side values are
	// validated (e.g. base branch via check-ref-format) before reaching git.
	cmd := exec.Command("git", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s: %w\n%s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}

// Client provides high-level git operations.
type Client struct {
	Runner Runner
}

// NewClient creates a Client backed by real git commands.
func NewClient() *Client {
	return &Client{Runner: ExecRunner{}}
}

// CurrentBranch returns the name of the currently checked-out branch.
func (c *Client) CurrentBranch() (string, error) {
	out, err := c.Runner.Run("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// ListBranches returns all local branches with their last commit time and
// merge status relative to the supplied base branch.
func (c *Client) ListBranches(base string) ([]Branch, error) {
	format := "%(refname:short)%00%(objectname:short)%00%(committerdate:iso8601)%00%(HEAD)%00"
	out, err := c.Runner.Run("for-each-ref", "--format="+format, "refs/heads/")
	if err != nil {
		return nil, err
	}

	mergedSet, err := c.mergedBranches(base)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(out), "\n")
	branches := make([]Branch, 0, len(lines))
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\x00")
		if len(parts) < 4 {
			continue
		}
		name := parts[0]
		hash := parts[1]
		dateStr := parts[2]
		current := parts[3] == "*"

		t, err := time.Parse(time.RFC3339, dateStr)
		if err != nil {
			// git iso8601 may contain a space instead of 'T' in some locales.
			t, err = time.Parse("2006-01-02 15:04:05 -0700", dateStr)
			if err != nil {
				return nil, fmt.Errorf("parsing commit date for %s: %w", name, err)
			}
		}

		_, merged := mergedSet[name]
		branches = append(branches, Branch{
			Name:       name,
			Ref:        hash,
			LastCommit: t,
			Merged:     merged,
			Current:    current,
		})
	}

	return branches, nil
}

// validateRefName checks that ref looks like a valid branch/ref name.
func (c *Client) validateRefName(ref string) error {
	_, err := c.Runner.Run("check-ref-format", "--branch", ref)
	return err
}

// mergedBranches returns the set of local branches already merged into base.
func (c *Client) mergedBranches(base string) (map[string]struct{}, error) {
	if err := c.validateRefName(base); err != nil {
		return nil, fmt.Errorf("invalid base branch %q: %w", base, err)
	}
	out, err := c.Runner.Run("branch", "--merged", base)
	if err != nil {
		return nil, err
	}
	merged := make(map[string]struct{})
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Active branch is prefixed with "* ".
		line = strings.TrimPrefix(line, "* ")
		merged[line] = struct{}{}
	}
	return merged, nil
}

// DeleteBranch removes a local branch. If force is true it uses -D.
func (c *Client) DeleteBranch(name string, force bool) error {
	flag := "-d"
	if force {
		flag = "-D"
	}
	_, err := c.Runner.Run("branch", flag, "--", name)
	return err
}

// PruneRemotes runs "git remote prune" for every configured remote.
func (c *Client) PruneRemotes() error {
	out, err := c.Runner.Run("remote")
	if err != nil {
		return err
	}
	remotes := strings.Fields(out)
	for _, remote := range remotes {
		if _, err := c.Runner.Run("remote", "prune", remote); err != nil {
			return fmt.Errorf("pruning remote %s: %w", remote, err)
		}
	}
	return nil
}

// IsGitRepo returns true when the current directory is inside a git repository.
func (c *Client) IsGitRepo() bool {
	_, err := c.Runner.Run("rev-parse", "--git-dir")
	return err == nil
}
