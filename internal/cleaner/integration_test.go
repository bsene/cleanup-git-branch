package cleaner

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/bsene/cleanup-git-branch/internal/config"
	"github.com/bsene/cleanup-git-branch/internal/git"
)

// runGit runs a git command in dir, failing the test on error.
func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func branchExists(t *testing.T, dir, name string) bool {
	t.Helper()
	cmd := exec.Command("git", "rev-parse", "--verify", name)
	cmd.Dir = dir
	return cmd.Run() == nil
}

// TestRun_Integration_UnmergedBranchKept verifies against a real git repo that
// only merged branches are deleted and unmerged work is never lost.
func TestRun_Integration_UnmergedBranchKept(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init", "-q")
	runGit(t, dir, "config", "user.email", "test@example.com")
	runGit(t, dir, "config", "user.name", "Test")

	writeFile(t, dir, "file.txt", "initial")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-q", "-m", "initial")

	// merged branch: merged into main.
	runGit(t, dir, "checkout", "-q", "-b", "merged")
	writeFile(t, dir, "merged.txt", "merged work")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-q", "-m", "merged commit")
	runGit(t, dir, "checkout", "-q", "main")
	runGit(t, dir, "merge", "-q", "--no-ff", "merged", "-m", "merge merged")

	// unmerged branch: never merged.
	runGit(t, dir, "checkout", "-q", "-b", "unmerged")
	writeFile(t, dir, "unmerged.txt", "unmerged work")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-q", "-m", "unmerged commit")
	runGit(t, dir, "checkout", "-q", "main")

	// The real client runs git in the process cwd, so chdir into the repo.
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldwd)

	c := NewCleaner(git.NewClient())
	if _, err := c.Run(&config.Config{Yes: true, AgeDays: 0}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if branchExists(t, dir, "merged") {
		t.Error("merged branch should have been deleted")
	}
	if !branchExists(t, dir, "unmerged") {
		t.Error("unmerged branch should have been kept")
	}
}
