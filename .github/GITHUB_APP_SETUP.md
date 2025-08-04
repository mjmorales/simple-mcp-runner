# GitHub App Setup for Semantic Release

This repository supports using a GitHub App for semantic-release automation instead of the default `GITHUB_TOKEN`. This provides better security and more granular permissions.

## Setup Instructions

### Option 1: Using GitHub App with Installation Token

1. **Create or use your existing GitHub App**
2. **Add Repository Variables** (Settings → Secrets and variables → Actions → Variables):
   - `APP_ID`: Your GitHub App ID
3. **Add Repository Secrets** (Settings → Secrets and variables → Actions → Secrets):
   - `APP_PRIVATE_KEY`: Your GitHub App's private key (PEM format)

### Option 2: Using Pre-generated App Token

1. **Generate a token from your GitHub App**
2. **Add Repository Secret**:
   - `APP_TOKEN`: Your pre-generated GitHub App token

## Required Permissions

Your GitHub App needs these permissions:
- **Contents**: Write (to create tags and push changes)
- **Metadata**: Read (basic repository info)
- **Pull requests**: Write (if creating releases from PRs)
- **Issues**: Write (for release notes)

## How It Works

The workflow will:
1. Try to use GitHub App token (if configured)
2. Fall back to default `GITHUB_TOKEN` if no app is configured
3. Use the app token for:
   - Checking out code with full history
   - Creating tags and releases
   - Pushing changelog updates

## Benefits of Using GitHub App

- **Better Security**: App tokens are scoped to specific repositories
- **Bypass Branch Protection**: Apps can push to protected branches
- **Custom Permissions**: More granular control than personal tokens
- **Organization Control**: Apps can be installed org-wide

## Testing

After setup, push a commit with conventional format to trigger:

```bash
git commit -m "feat: test GitHub app integration"
git push origin main
```

The semantic-release workflow should run using your GitHub App's permissions.