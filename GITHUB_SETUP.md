# 🚀 GitHub Setup Guide

This guide will help you push your IRC server project to GitHub.

## Prerequisites

- Git installed on your system
- GitHub account created
- SSH key configured with GitHub (recommended) OR GitHub personal access token

## Step 1: Initialize Git Repository (if not already done)

```bash
cd /home/rae/ircd

# Initialize git if not already done
git init

# Check current status
git status
```

## Step 2: Create .gitignore (Already Done ✅)

The `.gitignore` file is already configured to exclude:
- Binary files (`/bin/`, `ircd`)
- Logs (`/logs/`, `*.log`)
- Private keys (`/certs/*.key`)
- IDE files (`.vscode/`, `.idea/`)
- Build artifacts

## Step 3: Create GitHub Repository

### Option A: Via GitHub Website
1. Go to https://github.com/new
2. Repository name: `ircd` (or your choice)
3. Description: "Modern IRC server implementation in Go"
4. Choose Public or Private
5. **Do NOT** initialize with README (we have one)
6. Click "Create repository"

### Option B: Via GitHub CLI
```bash
# Install GitHub CLI if not installed
# sudo snap install gh

gh auth login
gh repo create ircd --public --source=. --remote=origin --description="Modern IRC server in Go"
```

## Step 4: Prepare Your Code

```bash
# Check what files will be committed
git status

# Add all files (respecting .gitignore)
git add .

# Check what's staged
git status

# Create first commit
git commit -m "Initial commit: Production-ready IRC server

- 17 IRC commands implemented (NICK, USER, JOIN, PART, PRIVMSG, NOTICE, QUIT, PING, PONG, NAMES, TOPIC, MODE, KICK, WHO, WHOIS, LIST, INVITE)
- User and channel modes
- TLS/SSL support
- Rate limiting and security features
- Docker and systemd deployment
- 75% test coverage in critical packages
- Comprehensive documentation"
```

## Step 5: Push to GitHub

```bash
# Add remote (replace USERNAME with your GitHub username)
git remote add origin git@github.com:USERNAME/ircd.git

# Or if using HTTPS:
# git remote add origin https://github.com/USERNAME/ircd.git

# Verify remote
git remote -v

# Push to GitHub
git branch -M main
git push -u origin main
```

## Step 6: Configure GitHub Repository Settings

### Add Topics/Tags
Go to your repository on GitHub and add topics:
- `irc`
- `irc-server`
- `golang`
- `go`
- `chat`
- `real-time`
- `networking`
- `tcp`
- `tls`
- `rfc1459`
- `docker`
- `systemd`

### Enable GitHub Pages (Optional)
If you want to host documentation:
1. Go to Settings → Pages
2. Source: Deploy from a branch
3. Branch: main, folder: /docs
4. Save

### Add Repository Description
Edit the repository description and website URL at the top of your GitHub repo page.

## Step 7: Set Up Branch Protection (Recommended)

1. Go to Settings → Branches
2. Add rule for `main` branch:
   - ✅ Require pull request reviews before merging
   - ✅ Require status checks to pass before merging
   - ✅ Require conversation resolution before merging

## Step 8: Add GitHub Actions Status Badge

The CI/CD workflow will automatically run. Once complete, add the status badge to your README:

```markdown
![CI](https://github.com/USERNAME/ircd/workflows/CI/badge.svg)
```

## What Gets Committed?

### ✅ Included:
- Source code (`cmd/`, `internal/`)
- Documentation (`docs/`, `README.md`)
- Configuration templates (`config/config.example.yaml`)
- Build scripts (`Makefile`, `generate_cert.sh`)
- Tests (`*_test.go`, `tests/`)
- Deployment files (`docker-compose.yml`, `deploy/`)
- License and contributing guidelines

### ❌ Excluded (by .gitignore):
- Binary files (`bin/ircd`)
- Logs (`logs/`)
- Private keys (`certs/*.key`)
- IDE files (`.vscode/`, `.idea/`)
- Temporary files

## Common Git Commands

```bash
# Check status
git status

# Add files
git add <file>
git add .

# Commit changes
git commit -m "Description of changes"

# Push to GitHub
git push

# Pull latest changes
git pull

# Create new branch
git checkout -b feature/new-feature

# Switch branches
git checkout main

# View commit history
git log --oneline

# View remote info
git remote -v
```

## Troubleshooting

### Authentication Issues

If you get authentication errors:

**Option 1: SSH (Recommended)**
```bash
# Generate SSH key
ssh-keygen -t ed25519 -C "your_email@example.com"

# Add to ssh-agent
eval "$(ssh-agent -s)"
ssh-add ~/.ssh/id_ed25519

# Copy public key
cat ~/.ssh/id_ed25519.pub

# Add to GitHub: Settings → SSH and GPG keys → New SSH key
```

**Option 2: Personal Access Token**
```bash
# Generate token: GitHub → Settings → Developer settings → Personal access tokens
# Use token as password when pushing via HTTPS
```

### Large Files

If you accidentally committed large files:
```bash
# Remove from git but keep locally
git rm --cached <large-file>

# Add to .gitignore
echo "<large-file>" >> .gitignore

# Commit and push
git commit -m "Remove large file from git"
git push
```

### Undo Last Commit (Not Pushed)
```bash
git reset --soft HEAD~1  # Keep changes staged
git reset HEAD~1         # Keep changes unstaged
git reset --hard HEAD~1  # Discard changes (careful!)
```

## Next Steps After Pushing

1. **Add Shields/Badges** to README:
   - Build status
   - Code coverage (via Codecov)
   - Go Report Card
   - License badge

2. **Create Releases**:
   ```bash
   git tag -a v0.2.0 -m "Version 0.2.0: Advanced IRC commands"
   git push origin v0.2.0
   ```

3. **Set Up GitHub Discussions** for community support

4. **Add Issue Templates** for bugs and feature requests

5. **Create Project Board** for tracking development

## Example: Creating Your First Release

```bash
# Tag current version
git tag -a v0.2.0 -m "Release v0.2.0

Features:
- 17 IRC commands implemented
- TLS/SSL encryption
- Rate limiting
- Channel operators
- User and channel modes
- WHO, WHOIS, LIST, INVITE commands
- Docker and systemd deployment
- 75% test coverage"

# Push tag
git push origin v0.2.0
```

Then on GitHub:
1. Go to Releases → Draft a new release
2. Choose tag: v0.2.0
3. Title: "v0.2.0 - Advanced IRC Commands"
4. Add release notes
5. Attach compiled binaries (optional)
6. Publish release

## Repository URL

After setup, your repository will be at:
```
https://github.com/USERNAME/ircd
```

Share it with the community! 🎉

## Additional Resources

- [GitHub Documentation](https://docs.github.com/)
- [Pro Git Book](https://git-scm.com/book/en/v2)
- [GitHub CLI Documentation](https://cli.github.com/manual/)
- [Semantic Versioning](https://semver.org/)

---

**Ready to share your IRC server with the world!** 🚀
