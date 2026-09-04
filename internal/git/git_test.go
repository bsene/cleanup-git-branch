package git

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"testing"
)

// diffExitError returns an *exec.ExitError with the given exit code for tests.
func diffExitError(exitCode int) error {
	cmd := exec.Command("sh", "-c", fmt.Sprintf("exit %d", exitCode))
	return cmd.Run()
}

type stubRunner struct {
	calls            []string
	forEachOut       string
	mergedOut        string
	mergedBase       string
	invalidBase      string
	pruneErr         error
	remoteOut        string
	squashMergedOf   map[string]bool
	filesChanged     map[string][]string
	errOn            map[string]error
}

func (s *stubRunner) Run(args ...string) (string, error) {
	s.calls = append(s.calls, strings.Join(args, " "))
	switch args[0] {
	case "for-each-ref":
		return s.forEachOut, nil
	case "check-ref-format":
		if s.invalidBase != "" && args[2] == s.invalidBase {
			return "", errors.New("invalid ref")
		}
		return "", nil
	case "branch":
		switch args[1] {
		case "--merged":
			return s.mergedOut, nil
		default:
			return "", nil
		}
	case "remote":
		if len(args) == 1 {
			return s.remoteOut, nil
		}
		if args[1] == "prune" {
			return "", s.pruneErr
		}
		return "", nil
	case "merge-base":
		if e, ok := s.errOn["merge-base"]; ok {
			return "", e
		}
		if s.invalidBase != "" && args[1] == s.invalidBase {
			return "", errors.New("invalid ref")
		}
		return "base-of-" + args[2], nil
	case "diff":
		if e, ok := s.errOn["diff"]; ok {
			return "", e
		}
		// git diff --name-only <merge-base>..<branch>
		if len(args) >= 3 && args[1] == "--name-only" {
			ref := args[2]
			branch := ref
			if i := strings.Index(ref, ".."); i >= 0 {
				branch = ref[i+2:]
			}
			if files, ok := s.filesChanged[branch]; ok {
				return strings.Join(files, "\n") + "\n", nil
			}
			return "", nil
		}
		// git diff --quiet <base> <branch> -- <files...>
		branch := args[3]
		if s.squashMergedOf != nil {
			if merged, ok := s.squashMergedOf[branch]; ok {
				if merged {
					return "", nil
				}
				return "", diffExitError(1)
			}
		}
		return "", diffExitError(1)
	case "rev-parse":
		if e, ok := s.errOn["rev-parse"]; ok {
			return "", e
		}
		if args[1] == "--git-dir" {
			return ".git", nil
		}
		return "main", nil
	}
	return "", nil
}

func TestCurrentBranch(t *testing.T) {
	stub := &stubRunner{}
	c := &Client{Runner: stub}
	got, err := c.CurrentBranch()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "main" {
		t.Errorf("expected branch main, got %q", got)
	}
}

func TestCurrentBranchError(t *testing.T) {
	stub := &stubRunner{errOn: map[string]error{"rev-parse": errors.New("not a repo")}}
	c := &Client{Runner: stub}
	_, err := c.CurrentBranch()
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestIsGitRepo(t *testing.T) {
	stub := &stubRunner{}
	c := &Client{Runner: stub}
	if !c.IsGitRepo() {
		t.Error("expected true when rev-parse succeeds")
	}
}

func TestIsGitRepoFalse(t *testing.T) {
	stub := &stubRunner{errOn: map[string]error{"rev-parse": errors.New("not a git repo")}}
	c := &Client{Runner: stub}
	if c.IsGitRepo() {
		t.Error("expected false when rev-parse fails")
	}
}

func TestDeleteBranch(t *testing.T) {
	stub := &stubRunner{}
	c := &Client{Runner: stub}

	if err := c.DeleteBranch("merged", false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := c.DeleteBranch("unmerged", true); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []string{"branch -d -- merged", "branch -D -- unmerged"}
	if len(stub.calls) != len(want) {
		t.Fatalf("expected calls %v, got %v", want, stub.calls)
	}
	for i, v := range want {
		if stub.calls[i] != v {
			t.Errorf("call[%d] = %q, want %q", i, stub.calls[i], v)
		}
	}
}

func TestPruneRemotes(t *testing.T) {
	stub := &stubRunner{remoteOut: "origin\nupstream"}
	c := &Client{Runner: stub}
	if err := c.PruneRemotes(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"remote", "remote prune origin", "remote prune upstream"}
	if len(stub.calls) != len(want) {
		t.Fatalf("expected calls %v, got %v", want, stub.calls)
	}
}

func TestPruneRemotesError(t *testing.T) {
	stub := &stubRunner{
		remoteOut: "origin",
		pruneErr:  errors.New("network"),
	}
	c := &Client{Runner: stub}
	err := c.PruneRemotes()
	if err == nil {
		t.Fatal("expected error pruning remote")
	}
}

func TestListBranches(t *testing.T) {
	out := strings.Join([]string{
		"main\x00abc1234\x002024-01-15T10:00:00+00:00\x00*",
		"feature\x00def5678\x002024-01-01 10:00:00 +0000\x00",
	}, "\n")
	stub := &stubRunner{
		forEachOut: out,
		mergedOut:  "* main\n  feature",
	}
	c := &Client{Runner: stub}
	branches, err := c.ListBranches("main")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(branches) != 2 {
		t.Fatalf("expected 2 branches, got %d", len(branches))
	}
	b := branches[0]
	if b.Name != "main" || b.Ref != "abc1234" || !b.Current || !b.Merged {
		t.Errorf("unexpected main branch: %+v", b)
	}
	b = branches[1]
	if b.Name != "feature" || b.Ref != "def5678" || b.Current || !b.Merged {
		t.Errorf("unexpected feature branch: %+v", b)
	}
}

func TestListBranchesInvalidBase(t *testing.T) {
	stub := &stubRunner{
		forEachOut:  "main\x00abc\x002024-01-15T10:00:00+00:00\x00*",
		invalidBase: "bad base",
	}
	c := &Client{Runner: stub}
	_, err := c.ListBranches("bad base")
	if err == nil {
		t.Fatal("expected error for invalid base branch")
	}
}

func TestListBranchesSquashMerge(t *testing.T) {
	out := strings.Join([]string{
		"main\x00abc1234\x002024-01-15T10:00:00+00:00\x00*",
		"feature\x00def5678\x002024-01-01 10:00:00 +0000\x00",
	}, "\n")
	stub := &stubRunner{
		forEachOut:     out,
		mergedOut:      "* main",
		squashMergedOf: map[string]bool{"feature": true},
		filesChanged:   map[string][]string{"feature": {"file.txt"}},
	}
	c := &Client{Runner: stub}
	branches, err := c.ListBranches("main")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(branches) != 2 {
		t.Fatalf("expected 2 branches, got %d", len(branches))
	}
	b := branches[1]
	if b.Name != "feature" || !b.Merged || !b.SquashMerged {
		t.Errorf("expected feature to be flagged as merged/squash-merged, got %+v", b)
	}
}

func TestListBranchesDiffError(t *testing.T) {
	out := strings.Join([]string{
		"main\x00abc1234\x002024-01-15T10:00:00+00:00\x00*",
		"feature\x00def5678\x002024-01-01 10:00:00 +0000\x00",
	}, "\n")
	stub := &stubRunner{
		forEachOut:     out,
		mergedOut:      "* main",
		squashMergedOf: map[string]bool{"feature": false},
		filesChanged:   map[string][]string{"feature": {"file.txt"}},
		errOn:          map[string]error{"diff": errors.New("diff failed")},
	}
	c := &Client{Runner: stub}
	_, err := c.ListBranches("main")
	if err == nil {
		t.Fatal("expected error when diff fails")
	}
}

func TestListBranchesMergeBaseError(t *testing.T) {
	out := strings.Join([]string{
		"main\x00abc1234\x002024-01-15T10:00:00+00:00\x00*",
		"feature\x00def5678\x002024-01-01 10:00:00 +0000\x00",
	}, "\n")
	stub := &stubRunner{
		forEachOut:   out,
		mergedOut:    "* main",
		errOn:        map[string]error{"merge-base": errors.New("merge-base failed")},
		filesChanged: map[string][]string{"feature": {"file.txt"}},
	}
	c := &Client{Runner: stub}
	_, err := c.ListBranches("main")
	if err == nil {
		t.Fatal("expected error when merge-base fails")
	}
}
