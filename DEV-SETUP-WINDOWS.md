# Development Environment Setup Guide - Windows/WSL2

**Last Updated:** 2026-02-26
**Target OS:** Windows 11/10 with WSL2
**Project:** Distributed Rate Limiter

---

## Table of Contents
1. [Install WSL2](#1-install-wsl2)
2. [Install and Configure Ubuntu in WSL2](#2-install-and-configure-ubuntu-in-wsl2)
3. [Install VS Code](#3-install-vs-code)
4. [Install Git and Configure](#4-install-git-and-configure)
5. [Install Go](#5-install-go)
6. [Install Development Tools](#6-install-development-tools)
7. [Install Redis](#7-install-redis)
8. [Install Docker Desktop](#8-install-docker-desktop)
9. [Clone and Setup Project](#9-clone-and-setup-project)
10. [Verify Installation](#10-verify-installation)
11. [Optional Tools](#11-optional-tools)
12. [SSH Key Setup for GitHub](#12-ssh-key-setup-for-github)

---

## 1. Install WSL2

### Step 1.1: Enable WSL
Open **PowerShell as Administrator** and run:

```powershell
wsl --install
```

This command will:
- Enable the WSL feature
- Enable Virtual Machine Platform
- Download and install the latest Linux kernel
- Set WSL2 as default
- Install Ubuntu by default

### Step 1.2: Restart Your Computer
After installation completes, restart your computer.

### Step 1.3: Verify WSL Version
After restart, open PowerShell and verify WSL2 is installed:

```powershell
wsl --list --verbose
```

You should see output showing Ubuntu with VERSION 2.

---

## 2. Install and Configure Ubuntu in WSL2

### Step 2.1: Launch Ubuntu
- Search for "Ubuntu" in Windows Start Menu
- Launch the Ubuntu application
- First launch will take a few minutes to complete setup

### Step 2.2: Create User Account
When prompted:
- Enter a Unix username (lowercase, no spaces)
- Enter and confirm a password
- **Important:** Remember this password - you'll need it for `sudo` commands

### Step 2.3: Update Ubuntu
```bash
sudo apt update && sudo apt upgrade -y
```

### Step 2.4: Install Essential Build Tools
```bash
sudo apt install -y build-essential curl wget unzip git
```

---

## 3. Install VS Code

### Step 3.1: Download and Install VS Code
1. Download VS Code from: https://code.visualstudio.com/
2. Run the installer on Windows (not in WSL)
3. During installation, make sure to check:
   - ✅ Add "Open with Code" action to Windows Explorer file context menu
   - ✅ Add "Open with Code" action to Windows Explorer directory context menu
   - ✅ Add to PATH

### Step 3.2: Install WSL Extension
1. Open VS Code on Windows
2. Press `Ctrl+Shift+X` to open Extensions
3. Search for "WSL"
4. Install the **"WSL"** extension by Microsoft

### Step 3.3: Install Go Extension
1. In VS Code Extensions (`Ctrl+Shift+X`)
2. Search for "Go"
3. Install **"Go"** extension by Go Team at Google

### Step 3.4: Connect to WSL
1. Press `Ctrl+Shift+P` to open Command Palette
2. Type "WSL: Connect to WSL"
3. VS Code will reopen connected to your WSL environment
4. You'll see "WSL: Ubuntu" in the bottom-left corner

### Step 3.5: Install Additional Extensions in WSL
Once connected to WSL, install these extensions:
- **GitLens** - Git supercharged
- **Git Graph** - View git history
- **EditorConfig** - Code style consistency
- **Makefile Tools** - Better Makefile support
- **YAML** - YAML language support

---

## 4. Install Git and Configure

### Step 4.1: Install Git in WSL
Git should already be installed from step 2.4, but verify:

```bash
git --version
```

If not installed:
```bash
sudo apt install -y git
```

### Step 4.2: Configure Git Identity
```bash
# Set your name (use your real name)
git config --global user.name "Your Name"

# Set your email (use the email associated with your GitHub account)
git config --global user.email "your.email@example.com"
```

### Step 4.3: Configure Git Settings
```bash
# Set default branch name to 'main'
git config --global init.defaultBranch main

# Enable color output
git config --global color.ui auto

# Set default editor (nano is beginner-friendly, you can use vim or code)
git config --global core.editor nano

# Enable credential caching (stores credentials for 1 hour)
git config --global credential.helper cache

# Or use Windows Credential Manager (recommended for WSL)
git config --global credential.helper "/mnt/c/Program\ Files/Git/mingw64/bin/git-credential-manager.exe"

# Enable case-sensitive file names
git config --global core.ignorecase false

# Use LF line endings (important for cross-platform development)
git config --global core.autocrlf input

# Enable rebase on pull
git config --global pull.rebase false
```

### Step 4.4: Verify Configuration
```bash
git config --list
```

---

## 5. Install Go

### Step 5.1: Download Go
```bash
# Remove any previous Go installation
sudo rm -rf /usr/local/go

# Download Go 1.25 (check for latest 1.25.x version at https://go.dev/dl/)
wget https://go.dev/dl/go1.25.0.linux-amd64.tar.gz

# Extract to /usr/local
sudo tar -C /usr/local -xzf go1.25.0.linux-amd64.tar.gz

# Remove the downloaded archive
rm go1.25.0.linux-amd64.tar.gz
```

**Note:** If Go 1.25.0 is not available yet, install the latest 1.24.x version and update when 1.25 is released.

### Step 5.2: Configure Go Environment
Add Go to your PATH by editing `.bashrc`:

```bash
nano ~/.bashrc
```

Add these lines at the end of the file:
```bash
# Go environment
export PATH=$PATH:/usr/local/go/bin
export GOPATH=$HOME/go
export PATH=$PATH:$GOPATH/bin
```

Save and exit:
- Press `Ctrl+X`
- Press `Y` to confirm
- Press `Enter` to save

### Step 5.3: Apply Changes
```bash
source ~/.bashrc
```

### Step 5.4: Verify Go Installation
```bash
go version
```

Expected output: `go version go1.25.0 linux/amd64`

### Step 5.5: Test Go Installation
```bash
# Create a test program
mkdir -p ~/test-go
cd ~/test-go
echo 'package main

import "fmt"

func main() {
    fmt.Println("Go is working!")
}' > main.go

# Run it
go run main.go

# Clean up
cd ~
rm -rf ~/test-go
```

---

## 6. Install Development Tools

### Step 6.1: Install golangci-lint
```bash
# Download and install golangci-lint
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin

# Verify installation
golangci-lint --version
```

### Step 6.2: Install Protocol Buffers Compiler (protoc)
```bash
# Install protoc
sudo apt install -y protobuf-compiler

# Verify installation
protoc --version

# Install Go protobuf plugins
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

### Step 6.3: Install Make
Make should already be installed from build-essential, verify:
```bash
make --version
```

### Step 6.4: Install Additional Useful Tools
```bash
# Install jq (JSON processor)
sudo apt install -y jq

# Install tree (directory tree viewer)
sudo apt install -y tree

# Install htop (process viewer)
sudo apt install -y htop
```

---

## 7. Install Redis

### Step 7.1: Install Redis Server
```bash
sudo apt install -y redis-server
```

### Step 7.2: Configure Redis
```bash
# Edit Redis configuration
sudo nano /etc/redis/redis.conf
```

Find the line with `supervised no` and change it to:
```
supervised systemd
```

Save and exit (`Ctrl+X`, `Y`, `Enter`)

### Step 7.3: Start Redis
```bash
# Start Redis service
sudo systemctl start redis-server

# Enable Redis to start on boot
sudo systemctl enable redis-server

# Check Redis status
sudo systemctl status redis-server
```

### Step 7.4: Test Redis
```bash
# Test Redis connection
redis-cli ping
```

Expected output: `PONG`

```bash
# Test basic commands
redis-cli set test "Hello Redis"
redis-cli get test
```

---

## 8. Install Docker Desktop

### Step 8.1: Download and Install Docker Desktop
1. Download Docker Desktop from: https://www.docker.com/products/docker-desktop/
2. Run the installer on Windows
3. During installation, ensure "Use WSL 2 instead of Hyper-V" is selected

### Step 8.2: Start Docker Desktop
1. Launch Docker Desktop from Windows Start Menu
2. Wait for Docker to start (this may take a few minutes the first time)
3. Accept the service agreement

### Step 8.3: Configure Docker for WSL2
1. Open Docker Desktop Settings (gear icon)
2. Go to "Resources" → "WSL Integration"
3. Enable integration with your Ubuntu distribution
4. Click "Apply & Restart"

### Step 8.4: Verify Docker in WSL
In your WSL Ubuntu terminal:
```bash
# Check Docker version
docker --version

# Check Docker Compose version
docker compose version

# Test Docker
docker run hello-world
```

---

## 9. Clone and Setup Project

### Step 9.1: Create Workspace Directory
```bash
# Create a workspace directory in your home folder
mkdir -p ~/workspace
cd ~/workspace
```

### Step 9.2: Clone Repository
```bash
# Clone your repository
git clone https://github.com/zahra-abedi/distributed-rate-limiter.git

# Navigate to project directory
cd distributed-rate-limiter
```

### Step 9.3: Install Project Dependencies
```bash
# Download Go dependencies
make deps

# Verify dependencies
go mod verify
```

### Step 9.4: Open Project in VS Code
```bash
# Open project in VS Code (from WSL)
code .
```

This will:
- Open VS Code on Windows
- Automatically connect to WSL
- Open the project folder

---

## 10. Verify Installation

### Step 10.1: Run All Checks
```bash
cd ~/workspace/distributed-rate-limiter

# Run formatting check
make fmt-check

# Run go vet
make vet

# Run linter
make lint

# Run tests
make test

# Build the project
make build
```

### Step 10.2: Run Benchmarks
```bash
make bench
```

### Step 10.3: Verify All Tools
```bash
echo "=== Environment Verification ==="
echo ""
echo "Git version:"
git --version
echo ""
echo "Go version:"
go version
echo ""
echo "golangci-lint version:"
golangci-lint --version
echo ""
echo "protoc version:"
protoc --version
echo ""
echo "Docker version:"
docker --version
echo ""
echo "Docker Compose version:"
docker compose version
echo ""
echo "Redis status:"
redis-cli ping
echo ""
echo "Make version:"
make --version | head -n1
```

---

## 11. Optional Tools

### 11.1: Install Oh My Zsh (Better Shell)
```bash
# Install zsh
sudo apt install -y zsh

# Install Oh My Zsh
sh -c "$(curl -fsSL https://raw.githubusercontent.com/ohmyzsh/ohmyzsh/master/tools/install.sh)"

# After installation, add Go paths to ~/.zshrc
echo '# Go environment' >> ~/.zshrc
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.zshrc
echo 'export GOPATH=$HOME/go' >> ~/.zshrc
echo 'export PATH=$PATH:$GOPATH/bin' >> ~/.zshrc

source ~/.zshrc
```

### 11.2: Install Windows Terminal (Highly Recommended)
1. Install from Microsoft Store: "Windows Terminal"
2. Set Ubuntu as default profile
3. Customize theme and settings

### 11.3: Install Useful VS Code Extensions
- **Error Lens** - Inline error highlighting
- **Better Comments** - Colorful comments
- **Path Intellisense** - Autocomplete file paths
- **Remote - SSH** - SSH support
- **Thunder Client** - REST API testing

### 11.4: Install Redis Insight (GUI for Redis)
Download from: https://redis.io/insight/
Useful for visualizing Redis data during development.

---

## 12. SSH Key Setup for GitHub

### Step 12.1: Generate SSH Key
```bash
# Generate a new SSH key (use your GitHub email)
ssh-keygen -t ed25519 -C "your.email@example.com"

# When prompted for file location, press Enter (use default)
# When prompted for passphrase, you can either:
#   - Enter a passphrase for extra security
#   - Press Enter twice for no passphrase (easier but less secure)
```

### Step 12.2: Start SSH Agent
```bash
# Start the ssh-agent
eval "$(ssh-agent -s)"

# Add your SSH key to the agent
ssh-add ~/.ssh/id_ed25519
```

### Step 12.3: Copy SSH Public Key
```bash
# Display your public key
cat ~/.ssh/id_ed25519.pub

# Copy the entire output (starts with ssh-ed25519 and ends with your email)
```

### Step 12.4: Add SSH Key to GitHub
1. Go to GitHub: https://github.com/settings/keys
2. Click "New SSH key"
3. Title: "Dell Laptop - WSL2"
4. Key type: Authentication Key
5. Paste your public key
6. Click "Add SSH key"

### Step 12.5: Test SSH Connection
```bash
# Test connection to GitHub
ssh -T git@github.com
```

Expected output:
```
Hi username! You've successfully authenticated, but GitHub does not provide shell access.
```

### Step 12.6: Update Git Remote to Use SSH (Optional)
If you want to use SSH instead of HTTPS for pushing:
```bash
cd ~/workspace/distributed-rate-limiter

# Check current remote
git remote -v

# Change to SSH (if currently using HTTPS)
git remote set-url origin git@github.com:zahra-abedi/distributed-rate-limiter.git

# Verify change
git remote -v
```

### Step 12.7: Configure SSH Agent to Auto-Start (Optional)
Add to your `~/.bashrc` or `~/.zshrc`:
```bash
nano ~/.bashrc
```

Add at the end:
```bash
# Auto-start SSH agent
if [ -z "$SSH_AUTH_SOCK" ]; then
   eval "$(ssh-agent -s)" > /dev/null
   ssh-add ~/.ssh/id_ed25519 2>/dev/null
fi
```

---

## Troubleshooting

### WSL2 Issues

**Problem:** WSL2 is slow or uses too much RAM
```bash
# Create .wslconfig in Windows user directory
# In WSL, run:
echo "[wsl2]
memory=8GB
processors=4" > /mnt/c/Users/YOUR_WINDOWS_USERNAME/.wslconfig

# Restart WSL from PowerShell:
wsl --shutdown
```

**Problem:** Cannot access WSL files from Windows
- Access WSL filesystem from Windows Explorer: `\\wsl$\Ubuntu\home\yourusername`

### Git Issues

**Problem:** Permission denied when pushing to GitHub
- Verify SSH key is added: `ssh -T git@github.com`
- Check remote URL: `git remote -v`
- Ensure SSH agent is running: `eval "$(ssh-agent -s)" && ssh-add ~/.ssh/id_ed25519`

### Go Issues

**Problem:** Command not found after installing Go tools
```bash
# Make sure GOPATH/bin is in PATH
export PATH=$PATH:$(go env GOPATH)/bin
source ~/.bashrc
```

### Docker Issues

**Problem:** Docker daemon not running
- Ensure Docker Desktop is running on Windows
- Check WSL integration is enabled in Docker Desktop settings

### Redis Issues

**Problem:** Redis server not starting
```bash
# Check Redis logs
sudo journalctl -u redis-server -n 50 --no-pager

# Restart Redis
sudo systemctl restart redis-server
```

---

## Quick Reference Commands

### Development Workflow
```bash
# Navigate to project
cd ~/workspace/distributed-rate-limiter

# Pull latest changes
git pull

# Create a new branch
git checkout -b feature/my-feature

# Install/update dependencies
make deps

# Run tests
make test

# Run linter
make lint

# Build project
make build

# Commit changes
git add .
git commit -m "feat: add new feature"

# Push to GitHub
git push origin feature/my-feature
```

### Redis Commands
```bash
# Start Redis
sudo systemctl start redis-server

# Stop Redis
sudo systemctl stop redis-server

# Restart Redis
sudo systemctl restart redis-server

# Redis CLI
redis-cli

# Monitor Redis commands
redis-cli monitor
```

### Docker Commands
```bash
# List running containers
docker ps

# List all containers
docker ps -a

# Stop all containers
docker stop $(docker ps -aq)

# Remove all containers
docker rm $(docker ps -aq)

# List images
docker images

# Remove unused images
docker image prune
```

---

## Next Steps

1. **Transfer this document to your Dell laptop**
   - Email it to yourself
   - Save to cloud storage (Google Drive, OneDrive)
   - Copy to USB drive

2. **Follow this guide step-by-step on your new laptop**
   - Don't skip steps
   - Verify each step before moving to the next
   - Run the verification commands

3. **Clone your projects**
   ```bash
   cd ~/workspace
   git clone git@github.com:zahra-abedi/distributed-rate-limiter.git
   ```

4. **Backup important files from your Mac**
   - SSH keys: `~/.ssh/`
   - Git config: `~/.gitconfig`
   - VS Code settings (if customized)
   - Any local environment files (`.env`, etc.)

5. **Additional considerations**
   - Export your browser bookmarks
   - Save any important notes or documentation
   - Export database schemas/data if applicable
   - Backup any local configuration files

---

## Resources

- **Go Documentation:** https://go.dev/doc/
- **WSL Documentation:** https://learn.microsoft.com/en-us/windows/wsl/
- **Docker Documentation:** https://docs.docker.com/
- **Redis Documentation:** https://redis.io/docs/
- **VS Code WSL Tutorial:** https://code.visualstudio.com/docs/remote/wsl
- **golangci-lint:** https://golangci-lint.run/
- **Protocol Buffers:** https://protobuf.dev/

---

## Support

If you encounter any issues:
1. Check the Troubleshooting section above
2. Search for error messages online
3. Check GitHub issues for the specific tool
4. Ask in relevant community forums (Reddit: r/golang, r/wsl, Stack Overflow)

---

**Good luck with your new development environment! 🚀**
