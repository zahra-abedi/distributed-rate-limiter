# Quick Setup Checklist

Use this alongside [DEV-SETUP-WINDOWS.md](DEV-SETUP-WINDOWS.md) for detailed instructions.

---

## Pre-Setup (On Mac - Before Returning)

- [ ] Save this document and DEV-SETUP-WINDOWS.md to cloud storage/email
- [ ] Backup SSH keys from `~/.ssh/` (if any)
- [ ] Note down Git username and email: `git config --list | grep user`
- [ ] Export browser bookmarks
- [ ] Backup any `.env` or config files from this project
- [ ] Push all uncommitted work to GitHub
- [ ] Document any special configurations or tools you use

---

## Windows Dell Laptop Setup

### Phase 1: Core System Setup (30-45 minutes)

- [ ] **1. Install WSL2**
  - [ ] Run PowerShell as Admin: `wsl --install`
  - [ ] Restart computer
  - [ ] Verify: `wsl --list --verbose`

- [ ] **2. Setup Ubuntu**
  - [ ] Launch Ubuntu from Start Menu
  - [ ] Create username and password
  - [ ] Update system: `sudo apt update && sudo apt upgrade -y`
  - [ ] Install essentials: `sudo apt install -y build-essential curl wget unzip git`

- [ ] **3. Install VS Code**
  - [ ] Download from https://code.visualstudio.com/
  - [ ] Install with PATH and context menu options
  - [ ] Install "WSL" extension
  - [ ] Install "Go" extension
  - [ ] Connect to WSL: Ctrl+Shift+P → "WSL: Connect to WSL"

### Phase 2: Git Configuration (10 minutes)

- [ ] **4. Configure Git**
  ```bash
  git config --global user.name "Your Name"
  git config --global user.email "your.email@example.com"
  git config --global init.defaultBranch main
  git config --global core.autocrlf input
  git config --global pull.rebase false
  ```

- [ ] Verify: `git config --list`

### Phase 3: Go Installation (15 minutes)

- [ ] **5. Install Go 1.25**
  ```bash
  wget https://go.dev/dl/go1.25.0.linux-amd64.tar.gz
  sudo tar -C /usr/local -xzf go1.25.0.linux-amd64.tar.gz
  rm go1.25.0.linux-amd64.tar.gz
  ```

- [ ] **6. Configure Go Environment**
  ```bash
  nano ~/.bashrc
  ```
  Add to end:
  ```bash
  export PATH=$PATH:/usr/local/go/bin
  export GOPATH=$HOME/go
  export PATH=$PATH:$GOPATH/bin
  ```
  Then: `source ~/.bashrc`

- [ ] Verify: `go version`

### Phase 4: Development Tools (20 minutes)

- [ ] **7. Install golangci-lint**
  ```bash
  curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin
  ```
  Verify: `golangci-lint --version`

- [ ] **8. Install Protocol Buffers**
  ```bash
  sudo apt install -y protobuf-compiler
  go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
  go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
  ```
  Verify: `protoc --version`

- [ ] **9. Install Redis**
  ```bash
  sudo apt install -y redis-server
  sudo nano /etc/redis/redis.conf  # Change "supervised no" to "supervised systemd"
  sudo systemctl start redis-server
  sudo systemctl enable redis-server
  ```
  Test: `redis-cli ping` (should return PONG)

### Phase 5: Docker (20 minutes)

- [ ] **10. Install Docker Desktop**
  - [ ] Download from https://www.docker.com/products/docker-desktop/
  - [ ] Install with WSL2 backend
  - [ ] Start Docker Desktop
  - [ ] Enable WSL Integration in Settings → Resources → WSL Integration
  - [ ] Restart Docker

- [ ] Verify in WSL: `docker run hello-world`

### Phase 6: SSH & GitHub (15 minutes)

- [ ] **11. Generate SSH Key**
  ```bash
  ssh-keygen -t ed25519 -C "your.email@example.com"
  eval "$(ssh-agent -s)"
  ssh-add ~/.ssh/id_ed25519
  cat ~/.ssh/id_ed25519.pub  # Copy output
  ```

- [ ] **12. Add to GitHub**
  - [ ] Go to https://github.com/settings/keys
  - [ ] Click "New SSH key"
  - [ ] Paste public key
  - [ ] Test: `ssh -T git@github.com`

### Phase 7: Project Setup (10 minutes)

- [ ] **13. Clone Repository**
  ```bash
  mkdir -p ~/workspace
  cd ~/workspace
  git clone git@github.com:zahra-abedi/distributed-rate-limiter.git
  cd distributed-rate-limiter
  ```

- [ ] **14. Install Dependencies**
  ```bash
  make deps
  ```

- [ ] **15. Open in VS Code**
  ```bash
  code .
  ```

### Phase 8: Verification (10 minutes)

- [ ] **16. Run All Checks**
  ```bash
  make fmt-check    # Format check
  make vet          # Go vet
  make lint         # Linter
  make test         # Tests
  make build        # Build
  ```

- [ ] **17. Environment Verification Script**
  ```bash
  git --version
  go version
  golangci-lint --version
  protoc --version
  docker --version
  redis-cli ping
  make --version
  ```

---

## Post-Setup Recommendations

- [ ] Install Windows Terminal from Microsoft Store
- [ ] Install VS Code extensions:
  - [ ] GitLens
  - [ ] Git Graph
  - [ ] Error Lens
  - [ ] Better Comments

- [ ] Optional: Install Oh My Zsh for better terminal experience
- [ ] Configure Windows Terminal to use Ubuntu as default
- [ ] Set up Redis Insight for Redis GUI (optional)

---

## Common Commands Reference

**Git:**
```bash
git checkout -b feature/new-feature  # New branch
git add .                            # Stage changes
git commit -m "message"              # Commit
git push origin branch-name          # Push
```

**Make:**
```bash
make test          # Run tests
make lint          # Run linter
make build         # Build project
make check         # Run all checks
```

**Redis:**
```bash
sudo systemctl start redis-server    # Start
sudo systemctl stop redis-server     # Stop
redis-cli                            # CLI access
```

**Docker:**
```bash
docker ps          # Running containers
docker images      # List images
docker stop <id>   # Stop container
```

---

## Time Estimate

- **Total setup time:** ~2-3 hours
- **Can be done in one sitting or broken up by phase**
- **Most time spent on downloads and installations**

---

## If Something Goes Wrong

1. Check detailed guide: DEV-SETUP-WINDOWS.md
2. Read error messages carefully
3. Google the error message
4. Check official documentation for the tool
5. Restart the terminal/WSL/computer
6. Try the command again

---

## Emergency Contact

Your development environment details:
- OS: Windows + WSL2 (Ubuntu)
- Go Version: 1.25
- Project: github.com/zahra-abedi/distributed-rate-limiter
- Shell: bash (or zsh if you installed Oh My Zsh)

Save this information for troubleshooting!

---

**✓ = Completed | ✗ = Issue | ⊘ = Skipped**

Good luck! 🎉
