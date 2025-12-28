# CI/CD Documentation

This document describes the continuous integration and continuous deployment (CI/CD) setup for zen-flow.

## Overview

zen-flow uses GitHub Actions for CI/CD. The workflows are defined in `.github/workflows/` and provide automated testing, linting, building, security scanning, and deployment.

## Workflows

### CI Workflow (`.github/workflows/ci.yml`)

Runs on every push and pull request to `main` and `develop` branches.

#### Jobs

1. **Lint**
   - Runs `golangci-lint` for Go code quality
   - Runs `go vet` for static analysis
   - Checks code formatting with `gofmt`
   - Validates `go.mod` and `go.sum` with `go mod tidy`
   - Lints YAML files with `yamllint`
   - Lints Helm charts with `helm lint`

2. **Test**
   - Runs unit tests with race detection
   - Runs integration tests
   - Checks test coverage threshold (minimum 75%)
   - Uploads coverage reports to Codecov (optional)

3. **Build**
   - Builds the controller binary
   - Verifies binary exists and is executable

4. **Security**
   - Runs `govulncheck` for Go vulnerability scanning
   - Runs `gosec` for security static analysis
   - Builds Docker image for Trivy scanning
   - Runs Trivy vulnerability scanner on Docker image
   - Runs Trivy filesystem scan
   - Checks for outdated base images

### Build Multi-Arch Workflow (`.github/workflows/build-multiarch.yml`)

Builds and pushes multi-architecture Docker images to Docker Hub.

**Triggers:**
- Push to `main` branch
- Git tags matching `v*` or `0.0.1-alpha`
- Manual workflow dispatch

**Features:**
- Builds for `linux/amd64` and `linux/arm64`
- Tags images with version, commit SHA, and branch name
- Uses Docker Buildx for multi-arch builds
- Caches builds using GitHub Actions cache

**Required Secrets:**
- `DOCKERHUB_USERNAME`: Docker Hub username
- `DOCKERHUB_TOKEN`: Docker Hub access token

### Publish Helm Chart Workflow (`.github/workflows/publish-helm-chart.yml`)

Publishes Helm charts to GitHub Pages.

**Triggers:**
- Push to `main` branch (when `charts/` changes)
- GitHub releases
- Manual workflow dispatch

**Features:**
- Lints Helm chart
- Packages chart as `.tgz`
- Generates Helm repository index
- Commits and pushes to `main` branch

**Repository URL:** `https://kube-zen.github.io/zen-flow`

## Local Development

### Running CI Checks Locally

```bash
# Format code
go fmt ./...

# Run go vet
go vet ./...

# Run tests
make test-unit

# Run E2E tests
make test-e2e

# Check coverage
make coverage

# Lint Helm charts
make helm-lint

# Validate examples
make validate-examples
```

### Pre-commit Checks

Before committing, ensure:
- Code is formatted (`go fmt ./...`)
- No `go vet` errors
- Tests pass (`make test-unit`)
- Coverage threshold met (75%+)
- YAML files are valid
- Helm charts lint successfully

## Dependabot

Automated dependency updates are configured via `.github/dependabot.yml`:

- **Go modules**: Weekly updates on Mondays
- **GitHub Actions**: Weekly updates on Mondays
- **Docker**: Weekly updates on Mondays

Dependabot creates pull requests with labels and assigns maintainers for review.

## Code Ownership

Code ownership is defined in `.github/CODEOWNERS`. All code is owned by `@kube-zen/maintainers` by default, with specific paths having explicit owners.

## Pull Request Process

1. Create a pull request using the template (`.github/PULL_REQUEST_TEMPLATE.md`)
2. CI workflows run automatically
3. All checks must pass before merge
4. At least one maintainer must approve
5. Merge when ready

## Issue Templates

Two issue templates are available:
- **Bug Report** (`.github/ISSUE_TEMPLATE/bug_report.md`)
- **Feature Request** (`.github/ISSUE_TEMPLATE/feature_request.md`)

## Secrets

The following secrets are required for full CI/CD functionality:

- `DOCKERHUB_USERNAME`: Docker Hub username (for multi-arch builds)
- `DOCKERHUB_TOKEN`: Docker Hub access token (for multi-arch builds)
- `CODECOV_TOKEN`: Codecov token (optional, for coverage reporting)
- `GITHUB_TOKEN`: Automatically provided by GitHub Actions

## Troubleshooting

### CI Failures

1. **Lint failures**: Run `golangci-lint run` locally to see issues
2. **Test failures**: Run `make test-unit` locally
3. **Coverage below threshold**: Add more tests or adjust threshold
4. **YAML lint failures**: Check YAML syntax and formatting
5. **Helm lint failures**: Run `helm lint charts/zen-flow` locally

### Build Failures

1. **Docker build failures**: Check Dockerfile syntax
2. **Multi-arch build failures**: Ensure Docker Buildx is available
3. **Helm chart failures**: Validate chart structure and values

### Security Scan Failures

Security scans are configured to not fail CI (`continue-on-error: true`), but results are uploaded as artifacts for review.

## Best Practices

1. **Keep workflows fast**: Use caching and parallel jobs
2. **Fail fast**: Run quick checks (lint, format) before slow checks (tests)
3. **Security first**: Always run security scans
4. **Document changes**: Update this document when adding new workflows
5. **Test locally**: Run CI checks locally before pushing

## References

- [GitHub Actions Documentation](https://docs.github.com/en/actions)
- [golangci-lint](https://golangci-lint.run/)
- [Trivy](https://aquasecurity.github.io/trivy/)
- [Helm Documentation](https://helm.sh/docs/)

