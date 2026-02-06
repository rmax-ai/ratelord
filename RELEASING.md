# RELEASING: Versioning & Release Strategy

**Status**: DRAFT
**Owner**: DevOps
**Related**: `TASKS.md`

## 1. Versioning Strategy

`ratelord` adheres to **Semantic Versioning 2.0.0** (SemVer).

- **MAJOR** (`vX.0.0`): Incompatible API changes (breaking the Agent Contract).
- **MINOR** (`v0.X.0`): Backwards-compatible functionality (new Epics/Features).
- **PATCH** (`v0.0.X`): Backwards-compatible bug fixes.

### 1.1 Pre-release Labels
- `vX.Y.Z-alpha.N`: Internal unstable builds.
- `vX.Y.Z-beta.N`: Feature-complete for verifying Epic targets.
- `vX.Y.Z-rc.N`: Release Candidate. Code freeze.

## 2. Release Automation

The release process is fully automated via GitHub Actions, triggered by **Git Tags**.

### 2.1 Workflow: `release.yaml`
Trigger: `push tags: v*`

**Steps**:
1.  **Test**: Run full unit and integration suite.
2.  **Build**: Compile binaries for targets (darwin/amd64, darwin/arm64, linux/amd64, linux/arm64).
3.  **Docker**: Build and push Docker image `rmax/ratelord-d:vX.Y.Z` and `latest`.
4.  **Changelog**: Generate from Conventional Commits since last tag.
5.  **Publish**: create GitHub Release with artifacts and changelog.

### 2.2 Artifacts
- `ratelord_vX.Y.Z_darwin_arm64.tar.gz` (bin/ratelord, bin/ratelord-d, bin/ratelord-sim)
- `ratelord_vX.Y.Z_linux_amd64.tar.gz`
- `checksums.txt` (SHA256)

## 3. Commit Convention Enforcement

We use conventional commits to automate changelogs.
- `feat:` -> Minor version bump (unless breaking).
- `fix:` -> Patch version bump.
- `docs:`, `chore:`, `test:` -> No version bump (or Patch if configured).
- `BREAKING CHANGE:` footer -> Major version bump.

## 4. Release Checklist (Manual)

Before pushing a tag:
1.  [ ] **Acceptance**: Run `make acceptance` (locally or on CI).
2.  [ ] **Docs**: Ensure `RELEASE_NOTES.md` is updated (if maintaining manually) or `NEXT_STEPS.md` is clear.
3.  [ ] **Tag**: `git tag -a v0.2.0 -m "release: Epic 27 complete"`
4.  [ ] **Push**: `git push origin v0.2.0`
