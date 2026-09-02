package cleaner

import (
	"errors"
	"testing"
	"time"

	"github.com/bsene/cleanup-git-branch/internal/config"
	"github.com/bsene/cleanup-git-branch/internal/git"
)

type stubClient struct {
	isRepo        bool
	current       string
	currentErr    error
	currentCalled bool
	branches      []git.Branch
	listErr       error
	deleteErr     map[string]error
	deleted       []string
	deleteForce   []bool
	pruned        bool
	pruneErr      error
}

func (s *stubClient) IsGitRepo() bool { return s.isRepo }
func (s *stubClient) CurrentBranch() (string, error) {
	s.currentCalled = true
	return s.current, s.currentErr
}
func (s *stubClient) ListBranches(base string) ([]git.Branch, error) {
	return s.branches, s.listErr
}
func (s *stubClient) DeleteBranch(name string, force bool) error {
	if err, ok := s.deleteErr[name]; ok {
		return err
	}
	s.deleted = append(s.deleted, name)
	s.deleteForce = append(s.deleteForce, force)
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

func TestRun_NotGitRepo(t *testing.T) {
	c := NewCleaner(&stubClient{isRepo: false})
	_, err := c.Run(&config.Config{AgeDays: 30})
	if err == nil {
		t.Fatal("expected error when not inside a git repository")
	}
}

func TestRun_CurrentBranchError(t *testing.T) {
	c := NewCleaner(&stubClient{
		isRepo:     true,
		currentErr: errors.New("detached HEAD"),
	})
	_, err := c.Run(&config.Config{AgeDays: 30})
	if err == nil {
		t.Fatal("expected error when current branch cannot be determined")
	}
}

func TestRun_ListBranchesError(t *testing.T) {
	c := NewCleaner(&stubClient{
		isRepo:  true,
		current: "main",
		listErr: errors.New("git failed"),
	})
	_, err := c.Run(&config.Config{AgeDays: 30})
	if err == nil {
		t.Fatal("expected error listing branches")
	}
}

func TestRun_InvalidExcludePattern(t *testing.T) {
	stub := &stubClient{
		isRepo:  true,
		current: "main",
		branches: []git.Branch{
			branch("feature/old", 40, true),
		},
	}
	c := NewCleaner(stub)
	_, err := c.Run(&config.Config{AgeDays: 30, Exclude: []string{"[bad"}})
	if err == nil {
		t.Fatal("expected error for invalid exclude pattern")
	}
}

func TestRun_NoCandidates(t *testing.T) {
	stub := &stubClient{
		isRepo:  true,
		current: "main",
		branches: []git.Branch{
			{Name: "main", LastCommit: time.Now(), Current: true},
		},
	}
	c := NewCleaner(stub)
	results, err := c.Run(&config.Config{Yes: true, AgeDays: 30, Exclude: []string{"main"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected no results, got %v", results)
	}
	if len(stub.deleted) != 0 {
		t.Fatalf("expected no deletions, got %v", stub.deleted)
	}
}

func TestRun_CurrentBranchSkipped(t *testing.T) {
	now := time.Now()
	stub := &stubClient{
		isRepo:  true,
		current: "main",
		branches: []git.Branch{
			{Name: "main", LastCommit: now.Add(-40 * 24 * time.Hour), Current: true, Merged: false},
		},
	}
	c := NewCleaner(stub)
	c.Now = now
	results, err := c.Run(&config.Config{Yes: true, AgeDays: 30})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 || len(stub.deleted) != 0 {
		t.Fatal("expected current branch to be skipped")
	}
}

func TestRun_AgeCutoff(t *testing.T) {
	now := time.Date(2024, time.January, 15, 12, 0, 0, 0, time.UTC)
	cfg := &config.Config{Yes: true, AgeDays: 30, Exclude: []string{"main"}}

	for _, tc := range []struct {
		name     string
		daysAgo  int
		merged   bool
		wantDel  bool
	}{
		{"feature/exact", 30, true, true},
		{"feature/older", 31, true, true},
		{"feature/newer", 29, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stub := &stubClient{
				isRepo:  true,
				current: "main",
				branches: []git.Branch{
					{Name: tc.name, LastCommit: now.Add(-time.Duration(tc.daysAgo) * 24 * time.Hour), Merged: tc.merged},
				},
			}
			c := NewCleaner(stub)
			c.Now = now
			_, err := c.Run(cfg)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			got := len(stub.deleted) == 1 && stub.deleted[0] == tc.name
			if got != tc.wantDel {
				t.Fatalf("expected deletion=%v, deleted=%v", tc.wantDel, stub.deleted)
			}
		})
	}
}

func TestRun_ForceDeleteFlag(t *testing.T) {
	now := time.Now()
	stub := &stubClient{
		isRepo:  true,
		current: "main",
		branches: []git.Branch{
			{Name: "feature/merged", LastCommit: now.Add(-40 * 24 * time.Hour), Merged: true},
			{Name: "feature/unmerged", LastCommit: now.Add(-40 * 24 * time.Hour), Merged: false},
		},
	}
	c := NewCleaner(stub)
	c.Now = now
	_, err := c.Run(&config.Config{Yes: true, AgeDays: 30})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(stub.deleted) != 2 || len(stub.deleteForce) != 2 {
		t.Fatalf("expected 2 deletions, got %v forces %v", stub.deleted, stub.deleteForce)
	}
	if stub.deleteForce[0] {
		t.Error("merged branch should use non-force delete")
	}
	if !stub.deleteForce[1] {
		t.Error("unmerged branch should use force delete")
	}
}

func TestRun_PruneRemotesDryRun(t *testing.T) {
	stub := &stubClient{
		isRepo:  true,
		current: "main",
		branches: []git.Branch{
			branch("feature/old", 40, true),
		},
	}
	c := NewCleaner(stub)
	_, err := c.Run(&config.Config{AgeDays: 30, PruneRemotes: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stub.pruned {
		t.Fatal("prune-remotes should not run during dry run")
	}
}

func TestRun_PruneRemotesError(t *testing.T) {
	stub := &stubClient{
		isRepo:   true,
		current:  "main",
		branches: []git.Branch{branch("feature/old", 40, true)},
		pruneErr: errors.New("prune failed"),
	}
	c := NewCleaner(stub)
	_, err := c.Run(&config.Config{Yes: true, AgeDays: 30, PruneRemotes: true})
	if err == nil {
		t.Fatal("expected error when prune-remotes fails")
	}
}

func TestRun_BasePassedToListBranches(t *testing.T) {
	stub := &stubClient{
		isRepo:  true,
		current: "main",
		branches: []git.Branch{
			{Name: "feature/old", LastCommit: time.Now().Add(-40 * 24 * time.Hour), Merged: true},
		},
	}
	c := NewCleaner(stub)
	_, err := c.Run(&config.Config{Yes: true, AgeDays: 30, Base: "develop"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stub.currentCalled {
		t.Error("expected current branch not to be consulted when base is provided")
	}
}

func TestRun_ReasonFormat(t *testing.T) {
	now := time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)
	date := time.Date(2023, time.December, 1, 0, 0, 0, 0, time.UTC)
	stub := &stubClient{
		isRepo:  true,
		current: "main",
		branches: []git.Branch{
			{Name: "feature/merged", LastCommit: date, Merged: true},
			{Name: "feature/unmerged", LastCommit: date, Merged: false},
		},
	}
	c := NewCleaner(stub)
	c.Now = now
	results, err := c.Run(&config.Config{AgeDays: 30})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].Reason != "last commit 2023-12-01, merged" {
		t.Errorf("unexpected merged reason: %q", results[0].Reason)
	}
	if results[1].Reason != "last commit 2023-12-01, not merged" {
		t.Errorf("unexpected unmerged reason: %q", results[1].Reason)
	}
}
