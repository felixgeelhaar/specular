# Installation Guide

This guide covers all installation methods for Specular, the AI-Native Spec and Build Assistant.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Quick Install](#quick-install)
  - [macOS](#macos)
  - [Linux](#linux)
  - [Windows](#windows)
- [Installation Methods](#installation-methods)
  - [GitHub Releases (Recommended)](#github-releases-recommended)
  - [Homebrew (macOS/Linux)](#homebrew-macoslinux)
  - [Linux Package Managers](#linux-package-managers)
  - [Docker](#docker)
  - [Build from Source](#build-from-source)
- [Post-Installation](#post-installation)
  - [Verify Installation](#verify-installation)
  - [Docker Setup](#docker-setup)
  - [AI Provider Configuration](#ai-provider-configuration)
  - [Shell Completion](#shell-completion)
- [Upgrading](#upgrading)
- [Uninstallation](#uninstallation)
- [Troubleshooting](#troubleshooting)

---

## Prerequisites

### Required
- **Docker Engine 24.0+** or **Docker Desktop** (for execution sandboxing)
- At least one AI provider:
  - **Anthropic Claude** (API key required)
  - **OpenAI GPT** (API key required)
  - **Google Gemini** (API key required)
  - **Ollama** (self-hosted, no API key needed)

### System Requirements
- **Operating System**: Linux, macOS, or Windows
- **Architecture**: x86_64 (amd64) or ARM64 (aarch64)
- **Memory**: 4GB RAM minimum, 8GB recommended
- **Disk Space**: 500MB for Specular + ~2GB for Docker images

---

## Quick Install

### macOS

**Using Homebrew** (recommended when tap is available):
```bash
# Add tap (when available)
brew tap felixgeelhaar/tap

# Install Specular
brew install specular

# Verify installation
specular version
```

**Using Binary Download**:
```bash
# Download latest release (Intel Mac)
curl -LO https://github.com/felixgeelhaar/specular/releases/latest/download/specular_Darwin_x86_64.tar.gz

# For Apple Silicon (M1/M2/M3)
curl -LO https://github.com/felixgeelhaar/specular/releases/latest/download/specular_Darwin_arm64.tar.gz

# Extract
tar -xzf specular_*.tar.gz

# Move to PATH
sudo mv specular /usr/local/bin/

# Verify
specular version
```

### Linux

**Debian/Ubuntu** (.deb):
```bash
# Download latest .deb package
curl -LO https://github.com/felixgeelhaar/specular/releases/latest/download/specular_amd64.deb

# For ARM64
curl -LO https://github.com/felixgeelhaar/specular/releases/latest/download/specular_arm64.deb

# Install
sudo dpkg -i specular_*.deb

# Verify
specular version
```

**RHEL/Fedora/CentOS** (.rpm):
```bash
# Download latest .rpm package
curl -LO https://github.com/felixgeelhaar/specular/releases/latest/download/specular_amd64.rpm

# For ARM64
curl -LO https://github.com/felixgeelhaar/specular/releases/latest/download/specular_arm64.rpm

# Install
sudo rpm -i specular_*.rpm

# Verify
specular version
```

**Alpine Linux** (.apk):
```bash
# Download latest .apk package
curl -LO https://github.com/felixgeelhaar/specular/releases/latest/download/specular_amd64.apk

# For ARM64
curl -LO https://github.com/felixgeelhaar/specular/releases/latest/download/specular_arm64.apk

# Install
sudo apk add --allow-untrusted specular_*.apk

# Verify
specular version
```

**Generic Linux Binary**:
```bash
# Download latest release
curl -LO https://github.com/felixgeelhaar/specular/releases/latest/download/specular_Linux_x86_64.tar.gz

# For ARM64
curl -LO https://github.com/felixgeelhaar/specular/releases/latest/download/specular_Linux_arm64.tar.gz

# Extract
tar -xzf specular_*.tar.gz

# Move to PATH
sudo mv specular /usr/local/bin/

# Verify
specular version
```

### Windows

**Using Binary Download**:
```powershell
# Download latest release (PowerShell)
Invoke-WebRequest -Uri "https://github.com/felixgeelhaar/specular/releases/latest/download/specular_Windows_x86_64.zip" -OutFile "specular.zip"

# Extract
Expand-Archive -Path specular.zip -DestinationPath C:\Program Files\Specular

# Add to PATH (PowerShell as Administrator)
$env:Path += ";C:\Program Files\Specular"
[Environment]::SetEnvironmentVariable("Path", $env:Path, [System.EnvironmentVariableTarget]::Machine)

# Verify
specular version
```

**Using Scoop** (community package, when available):
```powershell
scoop bucket add extras
scoop install specular
```

**Using Chocolatey** (community package, when available):
```powershell
choco install specular
```

---

## Installation Methods

### GitHub Releases (Recommended)

The most straightforward installation method for all platforms.

#### Step 1: Download

Visit the [Releases page](https://github.com/felixgeelhaar/specular/releases) and download the appropriate archive for your platform:

| Platform | Architecture | File |
|----------|-------------|------|
| macOS | Intel (x86_64) | `specular_Darwin_x86_64.tar.gz` |
| macOS | Apple Silicon (ARM64) | `specular_Darwin_arm64.tar.gz` |
| Linux | x86_64 | `specular_Linux_x86_64.tar.gz` |
| Linux | ARM64 | `specular_Linux_arm64.tar.gz` |
| Windows | x86_64 | `specular_Windows_x86_64.zip` |

#### Step 2: Verify Checksum (Recommended)

```bash
# Download checksums file
curl -LO https://github.com/felixgeelhaar/specular/releases/latest/download/checksums.txt

# Verify (Linux/macOS)
sha256sum -c checksums.txt

# Verify (macOS with shasum)
shasum -a 256 -c checksums.txt
```

#### Step 3: Extract

**Linux/macOS**:
```bash
tar -xzf specular_*.tar.gz
```

**Windows**:
```powershell
Expand-Archive -Path specular_*.zip -DestinationPath .
```

#### Step 4: Install

**Linux/macOS**:
```bash
# System-wide installation (requires sudo)
sudo mv specular /usr/local/bin/

# User installation (no sudo required)
mkdir -p ~/bin
mv specular ~/bin/
# Add ~/bin to PATH in ~/.bashrc or ~/.zshrc if not already there
```

**Windows**:
```powershell
# Move to Program Files
Move-Item -Path .\specular.exe -Destination "C:\Program Files\Specular\"

# Add to PATH (PowerShell as Administrator)
$env:Path += ";C:\Program Files\Specular"
[Environment]::SetEnvironmentVariable("Path", $env:Path, [System.EnvironmentVariableTarget]::Machine)
```

#### Step 5: Verify

```bash
specular version
```

Expected output:
```
specular v1.0.0
```

### Homebrew (macOS/Linux)

Homebrew installation (when tap is available):

```bash
# Add the tap (one-time setup)
brew tap felixgeelhaar/tap

# Install Specular
brew install specular

# Verify installation
specular version

# Upgrade to latest version
brew upgrade specular

# Uninstall
brew uninstall specular
```

**Note**: The Homebrew tap is currently in development and will be available soon.

### Linux Package Managers

#### Debian/Ubuntu (.deb)

```bash
# Download
VERSION=1.0.0  # Replace with desired version
curl -LO https://github.com/felixgeelhaar/specular/releases/download/v${VERSION}/specular_${VERSION}_amd64.deb

# Install
sudo dpkg -i specular_${VERSION}_amd64.deb

# Install dependencies (if needed)
sudo apt-get install -f

# Verify
specular version
```

**Features**:
- ✅ Automatic dependency installation (docker-ce)
- ✅ Shell completions (bash, zsh, fish)
- ✅ Man pages
- ✅ System service integration

#### RHEL/Fedora/CentOS (.rpm)

```bash
# Download
VERSION=1.0.0  # Replace with desired version
curl -LO https://github.com/felixgeelhaar/specular/releases/download/v${VERSION}/specular_${VERSION}_amd64.rpm

# Install
sudo rpm -i specular_${VERSION}_amd64.rpm

# Or using dnf (Fedora)
sudo dnf install specular_${VERSION}_amd64.rpm

# Or using yum (RHEL/CentOS)
sudo yum localinstall specular_${VERSION}_amd64.rpm

# Verify
specular version
```

#### Alpine Linux (.apk)

```bash
# Download
VERSION=1.0.0  # Replace with desired version
curl -LO https://github.com/felixgeelhaar/specular/releases/download/v${VERSION}/specular_${VERSION}_amd64.apk

# Install
sudo apk add --allow-untrusted specular_${VERSION}_amd64.apk

# Verify
specular version
```

### Docker

Specular is available as multi-architecture Docker images.

#### Pull Image

```bash
# Pull latest version
docker pull ghcr.io/felixgeelhaar/specular:latest

# Pull specific version
docker pull ghcr.io/felixgeelhaar/specular:v1.0.0

# Pull specific architecture
docker pull ghcr.io/felixgeelhaar/specular:latest-amd64
docker pull ghcr.io/felixgeelhaar/specular:latest-arm64
```

#### Run Container

```bash
# Basic usage
docker run --rm ghcr.io/felixgeelhaar/specular:latest version

# With volume mount for workspace
docker run --rm \
  -v $(pwd):/workspace \
  -w /workspace \
  ghcr.io/felixgeelhaar/specular:latest plan --spec .specular/spec.yaml

# With environment variables for AI providers
docker run --rm \
  -v $(pwd):/workspace \
  -w /workspace \
  -e ANTHROPIC_API_KEY="${ANTHROPIC_API_KEY}" \
  -e OPENAI_API_KEY="${OPENAI_API_KEY}" \
  ghcr.io/felixgeelhaar/specular:latest generate "Hello World"

# With Docker socket for nested Docker (required for build command)
docker run --rm \
  -v $(pwd):/workspace \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -w /workspace \
  ghcr.io/felixgeelhaar/specular:latest build --plan plan.json
```

#### Create Shell Alias

```bash
# Add to ~/.bashrc or ~/.zshrc
alias specular='docker run --rm -v $(pwd):/workspace -v /var/run/docker.sock:/var/run/docker.sock -w /workspace ghcr.io/felixgeelhaar/specular:latest'

# Usage
specular version
specular plan --spec .specular/spec.yaml
```

#### Docker Compose

```yaml
# docker-compose.yml
version: '3.8'

services:
  specular:
    image: ghcr.io/felixgeelhaar/specular:latest
    volumes:
      - .:/workspace
      - /var/run/docker.sock:/var/run/docker.sock
    working_dir: /workspace
    environment:
      - ANTHROPIC_API_KEY=${ANTHROPIC_API_KEY}
      - OPENAI_API_KEY=${OPENAI_API_KEY}
      - GEMINI_API_KEY=${GEMINI_API_KEY}
```

Usage:
```bash
docker-compose run --rm specular version
docker-compose run --rm specular plan --spec .specular/spec.yaml
```

### Build from Source

For developers who want the latest unreleased features or want to contribute.

#### Prerequisites
- **Go 1.24.6+**
- **Git**
- **Make** (optional but recommended)

#### Steps

1. **Clone the repository**:
   ```bash
   git clone https://github.com/felixgeelhaar/specular.git
   cd specular
   ```

2. **Build with Make** (recommended):
   ```bash
   # Build binary
   make build

   # Binary will be at ./specular
   ./specular version

   # Install to GOPATH/bin
   make install

   # Run tests
   make test

   # Run linter
   make lint
   ```

3. **Build with Go**:
   ```bash
   # Build with version information
   go build -ldflags="-s -w \
     -X github.com/felixgeelhaar/specular/internal/version.Version=dev \
     -X github.com/felixgeelhaar/specular/internal/version.Commit=$(git rev-parse --short HEAD) \
     -X github.com/felixgeelhaar/specular/internal/version.Date=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
     -o specular ./cmd/specular

   # Verify
   ./specular version
   ```

4. **Install system-wide**:
   ```bash
   # Linux/macOS
   sudo mv specular /usr/local/bin/

   # Or add to PATH
   export PATH="$PATH:$(pwd)"
   ```

---

## Post-Installation

### Verify Installation

```bash
# Check version
specular version

# Expected output:
# Specular 1.0.0 (abc12345) built 2025-01-07T10:30:00Z with go1.24.6 for linux/amd64

# Check help
specular --help

# List available commands
specular completion bash --help
```

### Docker Setup

Specular requires Docker for secure code execution.

#### Install Docker

**macOS**:
```bash
# Install Docker Desktop
brew install --cask docker
# Or download from https://www.docker.com/products/docker-desktop
```

**Linux**:
```bash
# Debian/Ubuntu
curl -fsSL https://get.docker.com | bash
sudo usermod -aG docker $USER
newgrp docker

# RHEL/Fedora/CentOS
sudo dnf install docker-ce docker-ce-cli containerd.io
sudo systemctl start docker
sudo systemctl enable docker
sudo usermod -aG docker $USER
```

**Windows**:
- Download and install [Docker Desktop for Windows](https://www.docker.com/products/docker-desktop)
- Enable WSL 2 backend (recommended)

#### Verify Docker

```bash
# Check Docker version
docker --version

# Test Docker
docker run hello-world

# Pre-warm Specular Docker images (optional but recommended)
specular prewarm
```

### AI Provider Configuration

Specular requires at least one AI provider to be configured.

#### Initialize Provider Configuration

```bash
# Create provider configuration
specular provider init

# This creates .specular/providers.yaml with default configuration
```

#### Configure Providers

Edit `.specular/providers.yaml`:

```yaml
version: 1.0

providers:
  # Anthropic Claude (recommended for quality)
  anthropic:
    enabled: true
    models:
      - claude-3-opus-20240229
      - claude-3-sonnet-20240229
      - claude-3-haiku-20240307

  # OpenAI GPT (recommended for code generation)
  openai:
    enabled: true
    models:
      - gpt-4-turbo-preview
      - gpt-4
      - gpt-3.5-turbo

  # Google Gemini
  gemini:
    enabled: true
    models:
      - gemini-pro

  # Ollama (self-hosted, free)
  ollama:
    enabled: true
    endpoint: "http://localhost:11434"
    models:
      - llama3.2
      - codellama
```

#### Set API Keys

Set environment variables for your chosen providers:

```bash
# Add to ~/.bashrc or ~/.zshrc

# Anthropic
export ANTHROPIC_API_KEY="sk-ant-..."

# OpenAI
export OPENAI_API_KEY="sk-..."
export OPENAI_ORG_ID="org-..."  # Optional

# Google Gemini
export GEMINI_API_KEY="AIza..."
```

For Windows (PowerShell):
```powershell
# Temporary (current session)
$env:ANTHROPIC_API_KEY = "sk-ant-..."
$env:OPENAI_API_KEY = "sk-..."

# Permanent (user environment)
[Environment]::SetEnvironmentVariable("ANTHROPIC_API_KEY", "sk-ant-...", "User")
[Environment]::SetEnvironmentVariable("OPENAI_API_KEY", "sk-...", "User")
```

#### Verify Provider Setup

```bash
# List configured providers
specular provider list

# Check provider health
specular provider health

# Test with a simple generation
specular generate "What is 2 + 2?" --model-hint fast
```

### Shell Completion

Enable shell completion for better UX.

#### Bash

```bash
# Generate completion script
specular completion bash > /usr/local/etc/bash_completion.d/specular

# Or for user-level installation
mkdir -p ~/.local/share/bash-completion/completions
specular completion bash > ~/.local/share/bash-completion/completions/specular

# Reload shell
source ~/.bashrc
```

#### Zsh

```bash
# Generate completion script
specular completion zsh > /usr/local/share/zsh/site-functions/_specular

# Or for user-level installation
mkdir -p ~/.zsh/completion
specular completion zsh > ~/.zsh/completion/_specular

# Add to ~/.zshrc if not already present
fpath=(~/.zsh/completion $fpath)
autoload -Uz compinit && compinit

# Reload shell
source ~/.zshrc
```

#### Fish

```bash
# Generate completion script
specular completion fish > ~/.config/fish/completions/specular.fish

# Reload shell
source ~/.config/fish/config.fish
```

#### PowerShell

```powershell
# Generate completion script
specular completion powershell | Out-File -Encoding UTF8 $PROFILE\specular.ps1

# Add to PowerShell profile
Add-Content $PROFILE ". $PROFILE\specular.ps1"

# Reload profile
. $PROFILE
```

---

## Upgrading

### Homebrew

```bash
brew upgrade specular
```

### Linux Package Managers

#### Debian/Ubuntu
```bash
# Download new version
curl -LO https://github.com/felixgeelhaar/specular/releases/latest/download/specular_amd64.deb

# Upgrade
sudo dpkg -i specular_amd64.deb
```

#### RHEL/Fedora
```bash
# Download new version
curl -LO https://github.com/felixgeelhaar/specular/releases/latest/download/specular_amd64.rpm

# Upgrade
sudo rpm -U specular_amd64.rpm
```

### Binary Installation

```bash
# Download new version
curl -LO https://github.com/felixgeelhaar/specular/releases/latest/download/specular_<OS>_<ARCH>.tar.gz

# Extract and replace
tar -xzf specular_*.tar.gz
sudo mv specular /usr/local/bin/

# Verify
specular version
```

### Docker

```bash
# Pull latest image
docker pull ghcr.io/felixgeelhaar/specular:latest

# Or pull specific version
docker pull ghcr.io/felixgeelhaar/specular:v1.1.0
```

---

## Uninstallation

### Homebrew

```bash
brew uninstall specular
```

### Linux Package Managers

#### Debian/Ubuntu
```bash
sudo dpkg -r specular
```

#### RHEL/Fedora
```bash
sudo rpm -e specular
```

### Binary Installation

```bash
# Remove binary
sudo rm /usr/local/bin/specular

# Remove configuration (optional)
rm -rf ~/.specular
```

### Docker

```bash
# Remove images
docker rmi ghcr.io/felixgeelhaar/specular:latest
docker rmi ghcr.io/felixgeelhaar/specular:v1.0.0

# Remove all Specular images
docker images | grep specular | awk '{print $3}' | xargs docker rmi
```

### Cleanup

```bash
# Remove configuration files
rm -rf ~/.specular

# Remove provider configuration in projects
find . -name ".specular" -type d -exec rm -rf {} +

# Remove Docker volumes (if any)
docker volume ls | grep specular | awk '{print $2}' | xargs docker volume rm
```

---

## Troubleshooting

### Installation Issues

#### "specular: command not found"

**Problem**: Binary not in PATH.

**Solution**:
```bash
# Check if binary exists
which specular

# Find specular binary
find / -name specular 2>/dev/null

# Add to PATH (add to ~/.bashrc or ~/.zshrc)
export PATH="$PATH:/path/to/specular/directory"

# Reload shell
source ~/.bashrc  # or source ~/.zshrc
```

#### Permission Denied

**Problem**: Binary doesn't have execute permissions.

**Solution**:
```bash
chmod +x /usr/local/bin/specular
```

#### Docker Not Found

**Problem**: Docker is not installed or not running.

**Solution**:
```bash
# Check Docker status
docker --version
systemctl status docker  # Linux

# Start Docker
sudo systemctl start docker  # Linux
# Or start Docker Desktop on macOS/Windows

# Install Docker if missing
# See: https://docs.docker.com/get-docker/
```

### Provider Issues

#### "No providers configured"

**Problem**: Provider configuration file missing or invalid.

**Solution**:
```bash
# Initialize provider configuration
specular provider init

# Edit configuration
vim .specular/providers.yaml

# Verify
specular provider list
```

#### "API key not found"

**Problem**: Environment variable for API key not set.

**Solution**:
```bash
# Check current environment
env | grep API_KEY

# Set API key
export ANTHROPIC_API_KEY="sk-ant-..."

# Add to shell profile for persistence
echo 'export ANTHROPIC_API_KEY="sk-ant-..."' >> ~/.bashrc
source ~/.bashrc
```

#### Provider Health Check Fails

**Problem**: Network issues or invalid API key.

**Solution**:
```bash
# Check provider health with verbose output
specular provider health --verbose

# Test API key manually
curl https://api.anthropic.com/v1/messages \
  -H "x-api-key: $ANTHROPIC_API_KEY" \
  -H "anthropic-version: 2023-06-01"

# Verify API key is correct in provider settings
cat .specular/providers.yaml | grep -A 5 anthropic
```

### Runtime Issues

#### "Docker permission denied"

**Problem**: User not in docker group.

**Solution**:
```bash
# Add user to docker group
sudo usermod -aG docker $USER

# Reload group membership
newgrp docker

# Or logout and login again
```

#### "Resource temporarily unavailable"

**Problem**: Resource limits exceeded.

**Solution**:
```bash
# Increase Docker resource limits
# Docker Desktop → Settings → Resources

# Or in daemon.json
sudo vim /etc/docker/daemon.json
# Add: {"default-ulimits": {"nofile": {"Name": "nofile", "Hard": 64000, "Soft": 64000}}}

# Restart Docker
sudo systemctl restart docker
```

### Common Error Messages

| Error | Cause | Solution |
|-------|-------|----------|
| `docker: command not found` | Docker not installed | Install Docker |
| `permission denied while trying to connect to the Docker daemon socket` | Not in docker group | Add user to docker group |
| `provider not found: anthropic` | Provider not configured | Run `specular provider init` and configure |
| `API key not found` | Environment variable not set | Set `ANTHROPIC_API_KEY` or similar |
| `spec.yaml not found` | Working directory issue | Run from project root with `.specular/` directory |
| `invalid policy.yaml` | YAML syntax error | Validate YAML syntax |
| `checkpoint corrupted` | Interrupted during checkpoint save | Delete `.specular/checkpoints/` and retry |

### Getting Help

If you encounter issues not covered here:

1. **Check Documentation**:
   - [Getting Started Guide](./getting-started.md)
   - [Provider Guide](./provider-guide.md)
   - [Architecture Decision Records](./adr/README.md)

2. **Search GitHub Issues**:
   - [Open Issues](https://github.com/felixgeelhaar/specular/issues)
   - [Closed Issues](https://github.com/felixgeelhaar/specular/issues?q=is%3Aissue+is%3Aclosed)

3. **Create New Issue**:
   - [Bug Report](https://github.com/felixgeelhaar/specular/issues/new?template=bug_report.md)
   - [Feature Request](https://github.com/felixgeelhaar/specular/issues/new?template=feature_request.md)

4. **Enable Debug Logging**:
   ```bash
   # Run with verbose output
   specular --verbose plan --spec .specular/spec.yaml

   # Or set environment variable
   export SPECULAR_LOG_LEVEL=debug
   specular plan --spec .specular/spec.yaml
   ```

---

## Next Steps

After installation:

1. **Read the Getting Started Guide**: [docs/getting-started.md](./getting-started.md)
2. **Configure AI Providers**: [docs/provider-guide.md](./provider-guide.md)
3. **Explore Examples**: [examples/README.md](../examples/README.md)
4. **Review Best Practices**: [docs/best-practices.md](./best-practices.md) (if available)

Happy building with Specular! 🚀
