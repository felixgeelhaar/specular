# Homebrew Tap Setup Guide

This guide explains how to set up the Homebrew tap repository for Specular.

## Overview

The Homebrew tap allows macOS and Linux users to install Specular using:
```bash
brew tap felixgeelhaar/tap
brew install specular
```

GoReleaser automatically updates the tap repository when a new release is created.

## Prerequisites

- GitHub account with push access to `felixgeelhaar/homebrew-tap` repository
- GitHub Personal Access Token with `repo` scope (for GoReleaser automation)

## Step 1: Create Homebrew Tap Repository

Create a new GitHub repository:

**Repository name:** `homebrew-tap`
**Owner:** `felixgeelhaar`
**Visibility:** Public
**Initialize:** Yes, with README

Full repository URL: `https://github.com/felixgeelhaar/homebrew-tap`

## Step 2: Initialize Repository

Clone the repository and add initial README:

```bash
git clone https://github.com/felixgeelhaar/homebrew-tap.git
cd homebrew-tap
```

Create an initial README:

```markdown
# Homebrew Tap for Specular

This is the official Homebrew tap for [Specular](https://github.com/felixgeelhaar/specular).

## Installation

\`\`\`bash
brew tap felixgeelhaar/tap
brew install specular
\`\`\`

## Usage

After installation, verify:

\`\`\`bash
specular version
\`\`\`

For more information, see the [Specular documentation](https://github.com/felixgeelhaar/specular).

## Automated Updates

This tap is automatically updated by [GoReleaser](https://goreleaser.com/) when new Specular releases are published.

## Formula

The formula for Specular is automatically generated and maintained by GoReleaser. Manual modifications to the formula will be overwritten on the next release.
\`\`\`

Commit and push:
```bash
git add README.md
git commit -m "feat: initialize homebrew tap for specular"
git push origin main
```

## Step 3: Configure GitHub Token

GoReleaser needs a GitHub token with `repo` scope to push formula updates.

### Create Personal Access Token

1. Go to GitHub Settings → Developer settings → Personal access tokens → Tokens (classic)
2. Click "Generate new token (classic)"
3. Name: "Specular Homebrew Tap"
4. Scopes: Check `repo` (full control of private repositories)
5. Generate and copy the token

### Add Token to Specular Repository

1. Go to `https://github.com/felixgeelhaar/specular/settings/secrets/actions`
2. Click "New repository secret"
3. Name: `TAP_GITHUB_TOKEN`
4. Value: Paste the token you created
5. Click "Add secret"

## Step 4: Test Release Process

When you create a release in the Specular repository, GoReleaser will:

1. Build binaries for all platforms
2. Generate Homebrew formula
3. Push formula to `homebrew-tap` repository
4. Create GitHub release with artifacts

### Creating a Test Release

```bash
# In the specular repository
git tag v0.1.0
git push origin v0.1.0
```

This triggers the GitHub Actions workflow which runs GoReleaser.

## Step 5: Verify Formula

After the release completes, check the tap repository:

```bash
cd homebrew-tap
git pull origin main
cat Formula/specular.rb
```

The formula should look like:
```ruby
class Specular < Formula
  desc "AI-Native Spec and Build Assistant with policy enforcement"
  homepage "https://github.com/felixgeelhaar/specular"
  version "0.1.0"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/felixgeelhaar/specular/releases/download/v0.1.0/specular_0.1.0_darwin_arm64.tar.gz"
      sha256 "..."
    end
    if Hardware::CPU.intel?
      url "https://github.com/felixgeelhaar/specular/releases/download/v0.1.0/specular_0.1.0_darwin_amd64.tar.gz"
      sha256 "..."
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/felixgeelhaar/specular/releases/download/v0.1.0/specular_0.1.0_linux_arm64.tar.gz"
      sha256 "..."
    end
    if Hardware::CPU.intel?
      url "https://github.com/felixgeelhaar/specular/releases/download/v0.1.0/specular_0.1.0_linux_amd64.tar.gz"
      sha256 "..."
    end
  end

  def install
    bin.install "specular"
    bash_completion.install "completions/specular.bash" => "specular"
    zsh_completion.install "completions/_specular" => "_specular"
    fish_completion.install "completions/specular.fish"
  end

  test do
    system "#{bin}/specular", "version"
  end
end
```

## Step 6: Test Installation

Users can now install Specular:

```bash
brew tap felixgeelhaar/tap
brew install specular
specular version
```

## Troubleshooting

### Formula Not Found

If users see "Error: No available formula with the name", ensure:
- The tap repository is public
- The formula file exists at `Formula/specular.rb`
- Users have run `brew tap felixgeelhaar/tap`

### Installation Fails

Check:
- Binary download URLs are accessible
- SHA256 checksums match
- Formula syntax is valid: `brew audit --strict Formula/specular.rb`

### Update Not Working

If the tap doesn't update after a release:
- Check GitHub Actions logs in the specular repository
- Verify `TAP_GITHUB_TOKEN` has correct permissions
- Ensure GoReleaser configuration is correct

## Maintenance

### Manual Formula Updates

If you need to manually update the formula:

```bash
cd homebrew-tap
# Edit Formula/specular.rb
brew audit --strict Formula/specular.rb
brew test-bot --only-formulae Formula/specular.rb
git add Formula/specular.rb
git commit -m "fix: update specular formula"
git push origin main
```

### Deprecating Old Versions

Homebrew automatically handles version updates. Old versions are available via:
```bash
brew install specular@0.1.0
```

## Resources

- [Homebrew Tap Documentation](https://docs.brew.sh/How-to-Create-and-Maintain-a-Tap)
- [GoReleaser Homebrew Integration](https://goreleaser.com/customization/homebrew/)
- [Homebrew Formula Cookbook](https://docs.brew.sh/Formula-Cookbook)
