# Release Pipeline Plan

## Goal

Automate versioning, changelog, and GitHub Releases from conventional commits.

## How It Works

1. PRs use conventional commit titles (`feat:`, `fix:`, `chore:`, etc.)
2. Squash merge with "PR title + commit details" — the PR title becomes the
   conventional commit message on main
3. When ready to release, push a semver tag (`git tag v0.1.0 && git push --tags`)
4. Release workflow runs `make ci`, then generates CHANGELOG via
   conventional-changelog and creates a GitHub Release
5. PR title format is enforced by the semantic-pull-request action

## Files

| File | Action |
|------|--------|
| `.github/workflows/release.yml` | Create — tag-triggered release workflow |
| `.github/workflows/pr-title.yml` | Create — enforce conventional commit PR titles |
| `cmd/questcore/main.go` | Update — version vars + `--version` flag |
| `Makefile` | Update — ldflags in build target |
| `package.json` | Create — conventional-changelog dependency |

## Task List

- [ ] Add version vars and `--version` flag to `cmd/questcore/main.go`
- [ ] Update Makefile build target with ldflags
- [ ] Create `.github/workflows/pr-title.yml` (semantic-pull-request)
- [ ] Set up conventional-changelog
- [ ] Create `.github/workflows/release.yml`
- [ ] Verify `make ci` passes
- [ ] Push branch, open PR
