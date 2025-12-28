# Contributing to zen-flow

Thank you for your interest in contributing to zen-flow! This document provides guidelines and instructions for contributing.

## Development Setup

### Prerequisites

- **Go**: 1.24 or later ([Download](https://golang.org/dl/))
- **kubectl**: Configured to access a Kubernetes cluster ([Install](https://kubernetes.io/docs/tasks/tools/))
- **Docker**: For building images ([Install](https://docs.docker.com/get-docker/))
- **Make**: For running common tasks ([Install](https://www.gnu.org/software/make/))
- **kind** (optional): For local E2E testing ([Install](https://kind.sigs.k8s.io/))
- **kubebuilder** (optional): For CRD development ([Install](https://kubebuilder.io/))

### Getting Started

1. **Fork and clone the repository:**
   ```bash
   git clone https://github.com/kube-zen/zen-flow.git
   cd zen-flow
   ```

2. **Verify Go installation:**
   ```bash
   go version  # Should be 1.24+
   ```

3. **Install dependencies:**
   ```bash
   go mod download
   ```

4. **Install development tools:**
   ```bash
   make install-tools
   ```

### Development Workflow

1. **Create feature branch:**
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Make changes and test:**
   ```bash
   make test-unit
   make fmt
   make lint
   ```

3. **Commit changes:**
   ```bash
   git add .
   git commit -m "feat: add your feature"
   ```

4. **Push and create PR:**
   ```bash
   git push origin feature/your-feature-name
   ```

### Commit Messages

We follow [Conventional Commits](https://www.conventionalcommits.org/):
- `feat:` - New feature
- `fix:` - Bug fix
- `docs:` - Documentation changes
- `refactor:` - Code refactoring
- `test:` - Test changes
- `chore:` - Build/tooling changes

### Testing

- **Unit tests:** `make test-unit`
- **E2E tests:** `make test-e2e`
- **Load tests:** `make test-load`
- **Coverage:** `make coverage`

### Pull Requests

1. Ensure your branch is up to date
2. Push your branch
3. Create a PR on GitHub
4. Ensure CI passes

### PR Checklist

- [ ] Code follows style guidelines
- [ ] Tests added/updated
- [ ] Documentation updated
- [ ] Commit messages follow conventions
- [ ] CI checks pass

## Code Style

- Follow Go standard formatting (`gofmt`)
- Run `make fmt` before committing
- Add comments for exported functions/types
- Keep functions focused and small

## License

By contributing, you agree that your contributions will be licensed under the Apache License 2.0.

Thank you for contributing! 🎉
