# cleanup-git-branch

A small, safe Go CLI for removing stale local Git branches. It defaults to a
dry run so you can preview what would be deleted before anything is actually
removed.

## Features

- **Dry-run by default** — review candidates before deleting anything.
- **Age-based filtering** — only consider branches whose last commit is older
  than a configurable number of days.
- **Merged-only** — only deletes branches already merged into a base branch.
- **Protected patterns** — exclude branches by glob patterns (`main`,
  `master`, `release/*`, etc.).
- **Remote-tracking cleanup** — optionally prune stale remote-tracking refs.
- **Colored summary** — plain-text table of candidates and actions.

## Installation

### From source

```bash
go install github.com/bsene/cleanup-git-branch/cmd/cleanup-git-branch@latest
```

### Build locally

```bash
git clone https://github.com/bsene/cleanup-git-branch.git
cd cleanup-git-branch
make install
```

## Usage

Run from inside a Git repository:

```bash
# Preview stale branches (default: older than 30 days, excludes main/master/develop/release/*)
cleanup-git-branch

# Actually delete the listed branches
cleanup-git-branch --yes

# Use a 60-day threshold
cleanup-git-branch --age-days 60

# Only delete branches already merged into the current branch
cleanup-git-branch

# Use a different base branch
cleanup-git-branch --base origin/main

# Protect additional patterns
cleanup-git-branch --exclude "main,master,hotfix/*"

# Delete local branches and prune remote-tracking refs
cleanup-git-branch --yes --prune-remotes
```

## CLI flags

```
-y, --yes              Actually delete branches (default: dry-run)
-a, --age-days int     Minimum age in days for a branch to be considered stale (default 30)
-b, --base string      Base branch to check merge status against (default: current branch)
-e, --exclude strings  Glob patterns protecting branches from deletion
                       (default [main,master,develop,release/*])
-p, --prune-remotes    Prune remote-tracking refs after local cleanup
-v, --verbose          Show per-branch details
-h, --help             Help
```

## Safety notes

- The tool never deletes the currently checked-out branch.
- `--yes` is required to perform actual deletions.
- Only branches merged into the base branch are deleted. Unmerged branches are
  always kept, so unmerged work is never lost.

## Documentation

See the [`docs/usage.md`](docs/usage.md) file for more detailed examples and
development notes.

## License

MIT — see [LICENSE](LICENSE).
