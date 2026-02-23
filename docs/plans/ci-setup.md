# CI/CD Setup — GitHub Actions + Makefile + golangci-lint

## Goal

Add automated build/test/lint gates for pull requests and post-merge validation
on main. Adopt a "local CI" philosophy where the Makefile is the single source
of truth — GitHub Actions calls `make ci`, the same command developers run locally.

## Approach

### Trunk-Based Development

- All development in short-lived feature branches off `main`
- PRs merge to `main` after CI passes
- Post-merge CI on `main` for validation

### Local CI Parity

- `Makefile` defines all check targets: `build`, `test`, `lint`, `vet`, `fmt-check`
- `make ci` runs the full pipeline (fmt-check -> vet -> lint -> build -> test)
- GitHub Actions workflow calls `make ci` — identical to local
- `CI` env var (set automatically by GitHub Actions) available if behavior needs to diverge

### Linting

- golangci-lint v2 with practical linter set: govet, errcheck, staticcheck,
  unused, gosimple, ineffassign, typecheck, gocritic, misspell
- errcheck suppressed in test files
- No pedantic style linters (wsl, nlreturn, funlen, etc.)

## Files

| File | Action | Purpose |
|------|--------|---------|
| `.github/workflows/ci.yml` | Create | GitHub Actions workflow calling `make ci` |
| `Makefile` | Create | Build/test/lint targets for local and CI use |
| `.golangci.yml` | Create | golangci-lint v2 configuration |
| `.gitignore` | Update | Add `/bin/` for Makefile build output |
| `.github/pull_request_template.md` | Update | Add CI checklist item |
| 6 Go source files | Update | Fix existing `gofmt` violations |

## Task List

- [x] Create this plan document
- [ ] Create feature branch `ci/github-actions-setup`
- [ ] Fix `gofmt` violations (`gofmt -w .`)
- [ ] Create `.golangci.yml` and fix lint issues
- [ ] Create `Makefile` + update `.gitignore`
- [ ] Create `.github/workflows/ci.yml` + update PR template
- [ ] Verify `make ci` passes locally
- [ ] Push branch and open PR
