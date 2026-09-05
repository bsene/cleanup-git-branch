#!/usr/bin/env bash
# Smoke test part 2: run cleanup-git-branch against the repo created by
# create-branches.sh and verify feature-1 and feature-2 were deleted while
# feature-3 was kept.

set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
project_root="$(cd "$script_dir/../.." && pwd)"
binary="${project_root}/cleanup-git-branch"
state_file="${TMPDIR:-/tmp}/cleanup-git-branch-smoke-state"

if [[ ! -f "$state_file" ]]; then
  echo "FAIL: state file $state_file not found. Run create-branches.sh first."
  exit 1
fi

repo="$(cat "$state_file")"
if [[ ! -d "$repo" ]]; then
  echo "FAIL: repo $repo from state file does not exist."
  exit 1
fi

cd "$repo"

echo ""
echo "=== Branches before cleanup ==="
git branch -v

echo ""
echo "=== Run cleanup-git-branch --age-days 0 --yes ==="
"$binary" --age-days 0 --yes

echo ""
echo "=== Branches after cleanup ==="
git branch -v

failed=0
for b in feature-1 feature-2; do
  if git rev-parse --verify "$b" >/dev/null 2>&1; then
    echo "FAIL: $b should have been deleted"
    failed=1
  fi
done
if ! git rev-parse --verify feature-3 >/dev/null 2>&1; then
  echo "FAIL: feature-3 should have been kept"
  failed=1
fi

if [[ "$failed" -ne 0 ]]; then
  exit 1
fi

echo ""
echo "PASS: feature-1 and feature-2 deleted, feature-3 kept."
