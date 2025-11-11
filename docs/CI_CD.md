# CI/CD Configuration

This document describes the continuous integration and deployment setup for Snippy Backend.

## GitHub Actions Workflows

### CI Workflow (`.github/workflows/ci.yml`)

Runs automatically on:

- Push to `main` or `develop` branches
- Pull requests to `main` or `develop` branches

#### Jobs

**1. Test Job**

- Runs all Go tests with race detection
- Uses PostgreSQL 16 service container
- Generates code coverage report
- Uploads coverage to Codecov
- **Duration**: ~2-3 minutes

**2. Security Job**

- Runs `gosec` security scanner
- Uploads results to GitHub Security tab (SARIF format)
- Runs `govulncheck` for dependency vulnerabilities
- **Duration**: ~1-2 minutes

**3. Format Job**

- Checks code formatting with `gofmt`
- Runs `golangci-lint` with 20+ linters
- **Duration**: ~2-3 minutes

**4. Build Job**

- Builds the application
- Creates downloadable artifact
- Only runs if all other jobs pass
- **Duration**: ~1 minute

**Total CI Time**: ~5-8 minutes

## Local Development Commands

### Make Commands

```bash
# Quick checks
make format        # Auto-format code
make format-check  # Check if formatted
make lint          # Run linter
make test          # Run tests
make security      # Run security scans
make build         # Build application

# Comprehensive check
make all           # Run all checks + build

# Utilities
make test-coverage # Generate HTML coverage report
make clean         # Remove build artifacts
make help          # Show all commands
```

### Manual Commands

```bash
# Format code
gofmt -s -w .

# Check format
test -z "$(gofmt -s -l .)"

# Run linter
golangci-lint run --timeout=5m

# Security scans
gosec ./...
govulncheck ./...

# Tests
go test -v -race ./...
go test -v -race -coverprofile=coverage.out ./...

# Build
go build -v -o snippy-backend .
```

## Git Hooks

### Pre-commit Hook

Automatically runs before each commit:

1. Code format check
2. Fast linter checks
3. Unit tests

**Install**: `ln -s ../../.githooks/pre-commit .git/hooks/pre-commit`

**Bypass** (not recommended): `git commit --no-verify`

## Linter Configuration

### golangci-lint (`.golangci.yml`)

Enabled linters (20+):

- `errcheck` - Unchecked errors
- `gosimple` - Code simplification
- `govet` - Suspicious constructs
- `ineffassign` - Ineffectual assignments
- `staticcheck` - Static analysis
- `unused` - Unused code
- `gosec` - Security issues
- `gofmt` - Code formatting
- `goimports` - Import formatting
- `misspell` - Spelling errors
- `bodyclose` - HTTP response body closure
- `sqlclosecheck` - SQL resource cleanup
- `revive` - Go linting rules
- And more...

### Configuration Highlights

- Cyclomatic complexity limit: 15
- Type assertions checked
- Blank error returns checked
- All vet checks enabled
- Medium severity for security issues

## Security Scanning

### gosec

Scans for common security issues:

- SQL injection
- File path injection
- Weak crypto
- Hardcoded credentials
- And more...

Results are uploaded to GitHub Security tab.

### govulncheck

Checks dependencies for known vulnerabilities:

- CVE database
- Go vulnerability database
- Direct and transitive dependencies

## Code Coverage

- Coverage reports generated on every test run
- Uploaded to Codecov for tracking
- Viewable in PR comments
- Historic trends available

Target: Maintain >80% coverage

## CI Badges

Add to README:

```markdown
[![CI](https://github.com/jheysaaz/snippy-backend/actions/workflows/ci.yml/badge.svg)](https://github.com/jheysaaz/snippy-backend/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/jheysaaz/snippy-backend)](https://goreportcard.com/report/github.com/jheysaaz/snippy-backend)
[![codecov](https://codecov.io/gh/jheysaaz/snippy-backend/branch/main/graph/badge.svg)](https://codecov.io/gh/jheysaaz/snippy-backend)
```

## Troubleshooting

### CI Failures

**Test Job Fails**

- Check PostgreSQL connection
- Review test logs
- Run locally: `make test`

**Security Job Fails**

- Review gosec output
- Check for new vulnerabilities
- Run locally: `make security`

**Format Job Fails**

- Run `make format` to auto-fix
- Run `make lint` to check locally
- Fix reported issues

**Build Job Fails**

- Check for compilation errors
- Verify all dependencies
- Run locally: `make build`

### Local Development Issues

**Linter not found**

```bash
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin
```

**gosec not found**

```bash
go install github.com/securego/gosec/v2/cmd/gosec@latest
```

**govulncheck not found**

```bash
go install golang.org/x/vuln/cmd/govulncheck@latest
```

## Best Practices

1. **Run checks before pushing**

   ```bash
   make all
   ```

2. **Install pre-commit hook**

   ```bash
   ln -s ../../.githooks/pre-commit .git/hooks/pre-commit
   ```

3. **Write tests for new features**

   - Aim for >80% coverage
   - Test edge cases
   - Include table-driven tests

4. **Keep dependencies updated**

   ```bash
   go get -u ./...
   go mod tidy
   ```

5. **Monitor security advisories**
   - Check GitHub Security tab
   - Run `govulncheck` regularly
   - Review gosec warnings

## Future Enhancements

Potential CI/CD improvements:

- [ ] Automated deployments
- [ ] Performance benchmarking
- [ ] Integration test suite
- [ ] Docker image builds
- [ ] Semantic versioning
- [ ] Changelog generation
- [ ] Dependency updates (Dependabot)

## Resources

- [GitHub Actions Documentation](https://docs.github.com/en/actions)
- [golangci-lint Linters](https://golangci-lint.run/usage/linters/)
- [gosec Rules](https://github.com/securego/gosec#available-rules)
- [Go Vulnerability Database](https://vuln.go.dev/)
