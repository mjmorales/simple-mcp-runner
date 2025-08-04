# Semantic Release Setup

This repository uses [semantic-release](https://semantic-release.gitbook.io/) to automatically version and publish releases based on conventional commit messages.

## How it works

1. **Conventional Commits**: Use conventional commit format in your commit messages
2. **Automatic Versioning**: Semantic-release analyzes commits and determines the next version
3. **Tag Creation**: Creates git tags automatically 
4. **Release Notes**: Generates release notes from commits
5. **GoReleaser Integration**: Existing release.yml workflow builds binaries when tags are created

## Commit Message Format

Use the conventional commit format:

```
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

### Types that trigger releases:

- **feat**: A new feature (triggers **minor** version bump)
- **fix**: A bug fix (triggers **patch** version bump)  
- **perf**: Performance improvement (triggers **patch** version bump)
- **refactor**: Code refactoring (triggers **patch** version bump)
- **revert**: Revert a previous commit (triggers **patch** version bump)

### Types that DON'T trigger releases:

- **docs**: Documentation changes
- **style**: Code style changes (formatting, etc)
- **test**: Adding or updating tests
- **chore**: Maintenance tasks
- **ci**: CI/CD changes
- **build**: Build system changes

### Breaking Changes:

Add `BREAKING CHANGE:` in the footer or `!` after type to trigger **major** version bump:

```
feat!: drop support for Go 1.19

BREAKING CHANGE: minimum Go version is now 1.20
```

## Examples

```bash
# Patch release (1.0.0 -> 1.0.1)
git commit -m "fix: resolve MCP connection timeout issue"

# Minor release (1.0.0 -> 1.1.0)  
git commit -m "feat: add support for custom timeout configuration"

# Major release (1.0.0 -> 2.0.0)
git commit -m "feat!: redesign configuration format

BREAKING CHANGE: configuration file format has changed from YAML to JSON"

# No release
git commit -m "docs: update README with installation instructions"
git commit -m "test: add unit tests for command executor"
git commit -m "chore: update dependencies"
```

## Workflow

1. Push commits with conventional format to `main` branch
2. Semantic-release workflow runs automatically
3. If release-worthy commits are found:
   - Creates new version tag
   - Updates CHANGELOG.md
   - Creates GitHub release with notes
4. Release workflow triggers on new tag
5. GoReleaser builds and publishes binaries

## Manual Trigger

You can manually trigger semantic-release using the GitHub Actions "workflow_dispatch" event in the Actions tab.