# Release Process

This document describes how to create and publish releases for reMarkable Sync.

## Overview

Releases are automated using [GoReleaser Pro](https://goreleaser.com/pro) and GitHub Actions. When a new version tag is pushed, the release workflow automatically:

1. Runs tests
2. Builds binaries for all supported platforms
3. Creates archives with checksums
4. Generates changelog from commits
5. Signs artifacts (optional)
6. Publishes GitHub release
7. Updates Homebrew tap (optional)

## Prerequisites

### For Maintainers

- **GoReleaser Pro license**: Set `GORELEASER_KEY` secret in GitHub repository
- **GitHub token**: Automatically provided by GitHub Actions
- **GPG key** (optional): For signing releases
  - Set `GPG_PRIVATE_KEY` secret (base64-encoded private key)
  - Set `GPG_PASSPHRASE` secret
- **Homebrew tap token** (optional): For automatic tap updates
  - Set `HOMEBREW_TAP_GITHUB_TOKEN` secret

### For Local Testing

Install GoReleaser Pro:
```bash
# macOS
brew install goreleaser/tap/goreleaser-pro

# Linux (using Go)
go install github.com/goreleaser/goreleaser-pro@latest
```

Export your GoReleaser Pro key:
```bash
export GORELEASER_KEY="your-key-here"
```

## Semantic Versioning

We follow [Semantic Versioning 2.0.0](https://semver.org/):

- **MAJOR** version (v1.0.0 → v2.0.0): Incompatible API changes
- **MINOR** version (v1.0.0 → v1.1.0): New functionality (backwards-compatible)
- **PATCH** version (v1.0.0 → v1.0.1): Bug fixes (backwards-compatible)

Pre-release versions:
- **Alpha**: `v1.0.0-alpha.1` - Early testing, unstable
- **Beta**: `v1.0.0-beta.1` - Feature complete, testing
- **RC**: `v1.0.0-rc.1` - Release candidate, final testing

## Creating a Release

### 1. Prepare the Release

Update version-related files if needed:
```bash
# Update CHANGELOG.md manually or use a tool
# Document all changes since last release

# Commit any final changes
git add .
git commit -m "chore: prepare release v1.2.3"
git push
```

### 2. Create and Push Tag

```bash
# Create annotated tag
git tag -a v1.2.3 -m "Release v1.2.3"

# Push tag to trigger release workflow
git push origin v1.2.3
```

The GitHub Actions workflow will automatically:
- Build for all platforms
- Run tests
- Create GitHub release
- Upload artifacts

### 3. Monitor Release

1. Go to [Actions](https://github.com/platinummonkey/remarkable-sync/actions)
2. Watch the "Release" workflow
3. Once complete, check the [Releases](https://github.com/platinummonkey/remarkable-sync/releases) page

### 4. Verify Release

```bash
# Download and test binaries for your platform
curl -L https://github.com/platinummonkey/remarkable-sync/releases/download/v1.2.3/remarkable-sync_1.2.3_darwin_arm64.tar.gz | tar xz

./remarkable-sync version
```

### 5. Announce Release (Optional)

- Update README badges if needed
- Post to relevant communities
- Update documentation site if available

## Local Testing

Test the release process locally without publishing:

```bash
# Build snapshot (doesn't require tag)
goreleaser release --snapshot --clean --skip=publish

# Check output in dist/ directory
ls -la dist/

# Test specific binary
./dist/remarkable-sync_darwin_arm64/remarkable-sync version
```

## Rollback a Release

If a release has critical issues:

1. **Delete the GitHub release:**
   ```bash
   gh release delete v1.2.3
   ```

2. **Delete the tag:**
   ```bash
   git tag -d v1.2.3
   git push origin :refs/tags/v1.2.3
   ```

3. **Create a patch release** with fixes:
   ```bash
   # Fix the issues
   git commit -m "fix: critical bug in v1.2.3"
   git tag -a v1.2.4 -m "Release v1.2.4 - fixes critical bug"
   git push origin v1.2.4
   ```

## Supported Platforms

GoReleaser builds for:

- **Linux**: amd64, arm64
- **macOS**: amd64 (Intel), arm64 (Apple Silicon)
- **Windows**: amd64

Archives:
- `.tar.gz` for Linux and macOS
- `.zip` for Windows

## Commit Message Format

For best changelog generation, use [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <subject>

<body>

<footer>
```

**Types:**
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation only
- `style`: Code style (formatting, no code change)
- `refactor`: Code refactoring
- `perf`: Performance improvement
- `test`: Adding/updating tests
- `build`: Build system or dependencies
- `ci`: CI/CD changes
- `chore`: Other changes (release prep, etc.)

**Examples:**
```
feat(sync): add label-based filtering
fix(ocr): handle empty pages correctly
docs: update installation instructions
perf(converter): optimize PDF rendering
```

## Configuration Files

### `.goreleaser.yaml`

Main configuration for GoReleaser Pro. Defines:
- Build targets and flags
- Archive formats
- Changelog generation
- Release notes templates
- Signing configuration
- Homebrew tap (optional)

See inline comments for detailed explanations.

### `.github/workflows/release.yml`

GitHub Actions workflow for automated releases. Triggers on tag push (`v*`).

## Troubleshooting

### Release fails with "GORELEASER_KEY not set"

Set the `GORELEASER_KEY` secret in GitHub repository settings:
1. Go to Settings → Secrets and variables → Actions
2. Add secret `GORELEASER_KEY` with your GoReleaser Pro license key

### GPG signing fails

Either:
- Remove GPG signing from `.goreleaser.yaml` (comment out `signs` section)
- Set up GPG secrets: `GPG_PRIVATE_KEY` and `GPG_PASSPHRASE`

### Builds fail on specific platform

Check the build matrix in `.goreleaser.yaml` and test locally:
```bash
GOOS=linux GOARCH=arm64 go build -o test-binary ./cmd/remarkable-sync
```

### Changelog is empty

Ensure commits follow conventional commit format and tags are properly created:
```bash
git log --oneline v1.2.2..v1.2.3
```

## Advanced: Homebrew Tap

To enable automatic Homebrew tap updates:

1. **Create tap repository**: `github.com/platinummonkey/homebrew-remarkable-sync`

2. **Generate GitHub token** with `repo` scope

3. **Add secret** `HOMEBREW_TAP_GITHUB_TOKEN` to main repository

4. **Uncomment** `brews` section in `.goreleaser.yaml`

5. **Test installation:**
   ```bash
   brew tap platinummonkey/remarkable-sync
   brew install remarkable-sync
   ```

## Resources

- [GoReleaser Pro Documentation](https://goreleaser.com/pro)
- [Semantic Versioning](https://semver.org/)
- [Conventional Commits](https://www.conventionalcommits.org/)
- [GitHub Actions](https://docs.github.com/en/actions)
- [GPG Signing Guide](https://docs.github.com/en/authentication/managing-commit-signature-verification)
