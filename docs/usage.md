---
title: Usage guide
permalink: /
---

# cleanup-git-branch usage guide

Source: [github.com/bsene/cleanup-git-branch](https://github.com/bsene/cleanup-git-branch)

## Table of contents

1. [Quick start](#quick-start)
2. [Examples](#examples)
3. [How filtering works](#how-filtering-works)
4. [Exit codes](#exit-codes)
5. [Development](#development)

## Quick start

The tool is designed to be safe by default. Running it without `--yes` only
prints the branches that would be deleted:

```bash
cd my-repository
cleanup-git-branch
```

Output:

```text
BRANCH        STATUS       REASON
feature/old   would delete last commit 2026-07-10, merged
```

## Examples

### Preview before deleting

```bash
cleanup-git-branch --age-days 14 --verbose
```

The `--verbose` flag appends the short commit hash for each candidate.

### Delete only merged branches

```bash
cleanup-git-branch --yes
```

This removes merged local branches older than the default 30 days, excluding
protected patterns. Unmerged branches are always kept.

### Delete stale branches and prune remotes

```bash
cleanup-git-branch --yes --prune-remotes
```

After local cleanup, this runs `git remote prune <remote>` for every
configured remote.

### Custom protected patterns

```bash
cleanup-git-branch --yes --exclude "main,master,develop,release/*,hotfix/*"
```

Patterns are simple globs as supported by `path.Match`.

## How filtering works

A branch is selected for cleanup only when all of the following are true:

1. It is **not** the currently checked-out branch.
2. Its name does **not** match any `--exclude` pattern.
3. Its last commit is older than `--age-days` days.
4. It has already been merged into the base branch.

## Exit codes

| Code | Meaning |
|------|---------|
| 0    | Success, or no stale branches found |
| 1    | Error (invalid flags, not a git repo, or deletion failures) |

## Development

Run the test suite:

```bash
make test
```

Build the binary:

```bash
make build
```

Run the linter:

```bash
make vet
```

All CI checks are defined in `.github/workflows/ci.yml`.
