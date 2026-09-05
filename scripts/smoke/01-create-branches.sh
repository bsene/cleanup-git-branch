#!/usr/bin/env bash
# Smoke test part 1: create a `git-test` worktree with derived branches from a
# `dummy-base` branch.
#
#   feature-1  merged normally (detected by `git branch --merged`)
#   feature-2  squash-merged (NOT detected by `git branch --merged`; caught
#              by the squash-merge check)
#   feature-3  left unmerged
#
# Records the worktree path in a state file so part 2 (check-cleanup.sh) can
# run the tool against it.

set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
project_root="$(cd "$script_dir/../.." && pwd)"
state_file="${TMPDIR:-/tmp}/cleanup-git-branch-smoke-state"
worktree="/tmp/git-test"

# Build a fresh binary.
(cd "$project_root" && go build -o cleanup-git-branch ./cmd/cleanup-git-branch)

# Reset any leftover worktree from a previous run so this is re-runnable.
if [[ -d "$worktree" ]]; then
  git -C "$project_root" worktree remove --force "$worktree"
fi
for b in dummy-base feature-1 feature-2 feature-3; do
  git -C "$project_root" branch -D "$b" >/dev/null 2>&1 || true
done

# Create a worktree on a new dummy-base branch.
git -C "$project_root" worktree add -b dummy-base "$worktree"
cd "$worktree"
git config user.email "smoke@example.com"
git config user.name "Smoke Test"

# Initial commit on dummy-base.
echo "initial" > file.txt
git add file.txt
git commit -q -m "initial commit"

# feature-1: merged normally.
git checkout -q -b feature-1
echo "feature 1 work" > feature-1.txt
git add feature-1.txt
git commit -q -m "feature 1 commit"
git checkout -q dummy-base

# feature-2: squash-merged.
git checkout -q -b feature-2
echo "feature 2 work" > feature-2.txt
git add feature-2.txt
git commit -q -m "feature 2 commit"
git checkout -q dummy-base

# feature-3: left unmerged.
git checkout -q -b feature-3
echo "feature 3 work" > feature-3.txt
git add feature-3.txt
git commit -q -m "feature 3 commit"
git checkout -q dummy-base

# Merge feature-1 normally and squash-merge feature-2.
git merge -q --no-ff feature-1 -m "merge feature-1"
git merge --squash feature-2 -q
git commit -q -m "squash merge feature-2"

echo ""
echo "=== Branches created ==="
git branch -v

# Record the worktree path for part 2.
echo "$worktree" > "$state_file"
