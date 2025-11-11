# Git Hooks

This directory contains Git hooks to help maintain code quality.

## Pre-commit Hook

The pre-commit hook runs automatically before each commit and performs:

1. **Code Format Check** - Ensures code is formatted with `gofmt`
2. **Linter** - Runs golangci-lint with fast checks
3. **Tests** - Runs unit tests to catch issues early

### Installation

To enable the pre-commit hook, run:

```bash
ln -s ../../.githooks/pre-commit .git/hooks/pre-commit
```

Or if you prefer to copy it:

```bash
cp .githooks/pre-commit .git/hooks/pre-commit
chmod +x .git/hooks/pre-commit
```

### Bypassing the Hook

If you need to commit without running the hooks (not recommended):

```bash
git commit --no-verify
```

### Troubleshooting

If the hook fails:

1. **Format issues**: Run `make format` to auto-fix formatting
2. **Linter issues**: Check the error output and fix the reported issues
3. **Test failures**: Fix failing tests before committing

The hook is designed to catch issues before they reach CI, saving time and reducing failed builds.
