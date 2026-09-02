package cleaner

import (
	"errors"
	"testing"
	"time"

	"github.com/bsene/cleanup-git-branch/internal/config"
	"github.com/bsene/cleanup-git-branch/internal/git"
)

type stubClient struct {
	isRepo    bool
	current   string
	branches  []git.Branch
	deleteErr map[string]error
	deleted   []string
	pruned    bool
	pruneErr  error
}

func (s *stubClient) IsGitRepo() bool                 { return s.isRepo }
func (s *stubClient) CurrentBranch() (string, error) { return s.current, nil }
func (s *stubClient) ListBranches(base string) ([]git.Branch, error) {
	return s.branches, nil
}
func (s *stubClient) DeleteBranch(name string, force bool) error {
	if err, ok := s.deleteErr[name]; ok {
		return err
	}
	s.deleted = append(s.deleted, name)
	return nil
}
func (s *stubClient) PruneRemotes() error {
	s.pruned = true
	return s.pruneErr
}

func branch(name string, daysAgo int, merged bool) git.Branch {
	return git.Branch{
		Name:       name,
		LastCommit: time.Now().Add(-time.Duration(daysAgo) * 24 * time.Hour),
		Merged:     merged,
	}
}

func TestRun_DryRunReportsCandidates(t *testing.T) {
	stub := &stubClient{
		isRepo:  true,
		current: "main",
		branches: []git.Branch{
			{Name: "main", LastCommit: time.Now(), Current: true},
			branch("feature/old", 40, true),
			branch("feature/recent", 5, true),
			branch("release/1.0", 60, true),
		},
	}
	c := NewCleaner(stub)

	cfg := &config.Config{AgeDays: 30, Exclude: []string{"main", "master", "develop", "release/*"}}
	results, err := c.Run(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(stub.deleted) != 0 {
		t.Fatalf("dry run deleted branches: %v", stub.deleted)
	}
	if len(results) != 1 || results[0].Branch.Name != "feature/old" {
		t.Fatalf("expected one candidate 'feature/old', got %+v", results)
	}
}

func TestRun_YesDeletesStaleBranches(t *testing.T) {
	stub := &stubClient{
		isRepo:  true,
		current: "main",
		branches: []git.Branch{
			{Name: "main", LastCommit: time.Now(), Current: true},
			branch("feature/old", 40, true),
			branch("feature/older", 50, false),
		},
	}
	c := NewCleaner(stub)

	cfg := &config.Config{Yes: true, AgeDays: 30, Exclude: []string{"main"}}
	results, err := c.Run(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if len(stub.deleted) != 2 {
		t.Fatalf("expected 2 deleted branches, got %v", stub.deleted)
	}
}

func TestRun_MergedOnly(t *testing.T) {
	stub := &stubClient{
		isRepo:  true,
		current: "main",
		branches: []git.Branch{
			{Name: "main", LastCommit: time.Now(), Current: true},
			branch("feature/merged", 40, true),
			branch("feature/unmerged", 40, false),
		},
	}
	c := NewCleaner(stub)

	cfg := &config.Config{Yes: true, Merged: true, AgeDays: 30, Exclude: []string{"main"}}
	_, err := c.Run(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(stub.deleted) != 1 || stub.deleted[0] != "feature/merged" {
		t.Fatalf("expected only merged branch deleted, got %v", stub.deleted)
	}
}

func TestRun_PruneRemotes(t *testing.T) {
	stub := &stubClient{
		isRepo:  true,
		current: "main",
		branches: []git.Branch{
			{Name: "main", LastCommit: time.Now(), Current: true},
			branch("feature/old", 40, true),
		},
	}
	c := NewCleaner(stub)

	cfg := &config.Config{Yes: true, AgeDays: 30, Exclude: []string{"main"}, PruneRemotes: true}
	_, err := c.Run(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stub.pruned {
		t.Fatal("expected remotes to be pruned")
	}
}

func TestRun_DeleteErrorCollected(t *testing.T) {
	stub := &stubClient{
		isRepo:    true,
		current:   "main",
		branches:  []git.Branch{branch("feature/broken", 40, true)},
		deleteErr: map[string]error{"feature/broken": errors.New("locked")},
	}
	c := NewCleaner(stub)

	cfg := &config.Config{Yes: true, AgeDays: 30}
	_, err := c.Run(cfg)
	if err == nil {
		t.Fatal("expected error reporting delete failure")
	}
}

func TestIsProtected(t *testing.T) {
	c := NewCleaner(nil)
	patterns := []string{"main", "master", "release/*"}
	for _, tc := range []struct {
		name    string
		want    bool
	}{
		{"main", true},
		{"master", true},
		{"release/1.0", true},
		{"feature/foo", false},
	} {
		got, err := c.isProtected(tc.name, patterns)
		if err != nil {
			t.Fatalf("isProtected(%q) unexpected error: %v", tc.name, err)
		}
		if got != tc.want {
			t.Errorf("isProtected(%q) = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestIsProtectedInvalidPattern(t *testing.T) {
	c := NewCleaner(nil)
	_, err := c.isProtected("main", []string{"[invalid"})
	if err == nil {
		t.Fatal("expected error for invalid glob pattern")
	}
}
