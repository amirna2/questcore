# Release Pipeline Plan

## Goal

Add a tag-triggered release workflow using GitHub's native release process.
Push a semver tag → CI runs → GitHub Release created with auto-generated notes.

## Approach

- `.github/release.yml` configures how GitHub categorizes PRs in release notes
- `.github/workflows/release.yml` triggers on `v*` tags, runs `make ci`, then
  creates a GitHub Release via `gh release create --generate-notes`
- Version/commit/date embedded in binary via ldflags (`--version` flag)
- Makefile `build` target updated to inject git info automatically

## Files

| File | Action |
|------|--------|
| `.github/release.yml` | Create — release notes categories |
| `.github/workflows/release.yml` | Create — tag-triggered release workflow |
| `cmd/questcore/main.go` | Update — version vars + `--version` flag |
| `Makefile` | Update — ldflags in build target |

## Task List

- [ ] Add version vars and `--version` flag to `cmd/questcore/main.go`
- [ ] Update Makefile build target with ldflags
- [ ] Create `.github/release.yml`
- [ ] Create `.github/workflows/release.yml`
- [ ] Verify `make ci` passes
- [ ] Push branch, open PR
