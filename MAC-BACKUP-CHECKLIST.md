# Mac Backup Checklist - Before Returning

**Important:** Complete this checklist BEFORE returning your Mac.

---

## 1. Git Configuration Backup

### Current Git Settings
Run these commands and save the output:

```bash
# View all git config
git config --list --show-origin > ~/Desktop/git-config-backup.txt

# Specifically save user info
echo "Git User Name: $(git config --global user.name)" >> ~/Desktop/git-info.txt
echo "Git User Email: $(git config --global user.email)" >> ~/Desktop/git-info.txt
```

- [ ] Saved git configuration
- [ ] Noted down Git username
- [ ] Noted down Git email

---

## 2. SSH Keys Backup

### Check for SSH Keys
```bash
# List all SSH keys
ls -la ~/.ssh/
```

### Backup SSH Keys (if they exist)
```bash
# Create backup directory on Desktop
mkdir -p ~/Desktop/ssh-backup

# Copy SSH keys
cp ~/.ssh/id_* ~/Desktop/ssh-backup/ 2>/dev/null
cp ~/.ssh/config ~/Desktop/ssh-backup/ 2>/dev/null
cp ~/.ssh/known_hosts ~/Desktop/ssh-backup/ 2>/dev/null

# List what was backed up
ls -la ~/Desktop/ssh-backup/
```

**Important:** If you have SSH keys:
- [ ] Backed up private keys (id_rsa, id_ed25519, etc.)
- [ ] Backed up public keys (id_rsa.pub, id_ed25519.pub, etc.)
- [ ] Backed up SSH config file
- [ ] **Transfer securely** - Use encrypted USB or secure cloud storage
- [ ] **Delete from cloud after transferring to new laptop**

---

## 3. Project Files Check

### Check for uncommitted changes
```bash
cd ~/personal/distributed-rate-limiter

# Check git status
git status

# Check for untracked files
git status --short

# Check for local branches not pushed
git branch -vv | grep -v origin
```

- [ ] All changes committed
- [ ] All commits pushed to GitHub
- [ ] No important untracked files
- [ ] All local branches pushed or merged

### Check for local configuration files
```bash
# Look for environment files
find ~/personal/distributed-rate-limiter -name ".env*" -o -name "*.local.*" -o -name "config.local.*"

# List all hidden files in project root
ls -la ~/personal/distributed-rate-limiter/
```

- [ ] Checked for `.env` files
- [ ] Checked for `.local` config files
- [ ] Backed up any important local configs

---

## 4. Development Tools & Settings

### VS Code Settings
```bash
# Check if you have custom VS Code settings
cat ~/Library/Application\ Support/Code/User/settings.json

# Backup VS Code settings
mkdir -p ~/Desktop/vscode-backup
cp ~/Library/Application\ Support/Code/User/settings.json ~/Desktop/vscode-backup/
cp ~/Library/Application\ Support/Code/User/keybindings.json ~/Desktop/vscode-backup/ 2>/dev/null
```

- [ ] Backed up VS Code settings (if customized)
- [ ] Backed up VS Code keybindings (if customized)
- [ ] Listed installed extensions: `code --list-extensions > ~/Desktop/vscode-extensions.txt`

### Shell Configuration
```bash
# Backup shell configs
mkdir -p ~/Desktop/shell-backup
cp ~/.bashrc ~/Desktop/shell-backup/ 2>/dev/null
cp ~/.bash_profile ~/Desktop/shell-backup/ 2>/dev/null
cp ~/.zshrc ~/Desktop/shell-backup/ 2>/dev/null
cp ~/.zprofile ~/Desktop/shell-backup/ 2>/dev/null
```

- [ ] Backed up shell configurations
- [ ] Noted any custom aliases or functions

---

## 5. GitHub & Cloud Services

### GitHub Access
```bash
# Test GitHub access
ssh -T git@github.com
```

- [ ] Verified GitHub SSH access works
- [ ] Noted GitHub username: _______________
- [ ] Saved GitHub password/token if using HTTPS

### GitHub Personal Access Tokens
- [ ] Check if you have any GitHub PATs: https://github.com/settings/tokens
- [ ] Note expiration dates
- [ ] Plan to create new ones on new laptop if needed

---

## 6. Browser Data

### Export Bookmarks
- [ ] Chrome/Edge: Settings → Bookmarks → Export bookmarks
- [ ] Firefox: Bookmarks → Show All Bookmarks → Import and Backup → Export
- [ ] Safari: File → Export Bookmarks

### Important Sites to Remember
- [ ] GitHub.com credentials
- [ ] Any cloud storage (Google Drive, Dropbox, OneDrive)
- [ ] Development tool accounts (Docker Hub, etc.)

---

## 7. Documentation & Notes

### Save Important Information
Create a file: `~/Desktop/dev-notes.txt` with:

```bash
# Create notes file
cat > ~/Desktop/dev-notes.txt << 'EOF'
# Development Environment Notes

## Current Setup
- OS: macOS
- Go Version: $(go version)
- Git Version: $(git --version)
- Redis: Installed? (yes/no)
- Docker: Installed? (yes/no)

## Important Paths
- Projects location: ~/personal/
- Workspace: ~/personal/distributed-rate-limiter

## Tools Installed
- golangci-lint
- protoc
- (list others)

## Custom Settings
- (note any custom configurations)

## Important Contacts/Resources
- GitHub: github.com/zahra-abedi
- Email: (your email)

## Notes
- (any other important information)

EOF
```

- [ ] Created development notes
- [ ] Listed all custom tools or utilities
- [ ] Documented any special configurations

---

## 8. Final Project Backup

### Create Complete Project Archive
```bash
cd ~/personal

# Create archive (without node_modules, vendor, etc.)
tar -czf ~/Desktop/distributed-rate-limiter-backup.tar.gz \
  --exclude='node_modules' \
  --exclude='vendor' \
  --exclude='bin' \
  --exclude='.git' \
  --exclude='coverage.*' \
  distributed-rate-limiter/

# Verify archive
tar -tzf ~/Desktop/distributed-rate-limiter-backup.tar.gz | head -20
```

- [ ] Created project archive
- [ ] Verified archive contents
- [ ] **Note:** This is just a safety backup - main source is on GitHub

---

## 9. Cloud Storage Transfer

### Upload to Cloud
Create a folder on cloud storage: "Mac Backup - [Date]"

Upload these files:
- [ ] `git-config-backup.txt`
- [ ] `git-info.txt`
- [ ] `ssh-backup/` folder (encrypted!)
- [ ] `vscode-backup/` folder
- [ ] `vscode-extensions.txt`
- [ ] `shell-backup/` folder
- [ ] `dev-notes.txt`
- [ ] `DEV-SETUP-WINDOWS.md` (this guide)
- [ ] `SETUP-CHECKLIST.md`
- [ ] Any `.env` or config files
- [ ] `distributed-rate-limiter-backup.tar.gz` (optional, GitHub is source of truth)

### Recommended Cloud Storage
- Google Drive (if you have Google account)
- OneDrive (comes with Windows)
- iCloud (accessible from web)
- Encrypted USB drive

---

## 10. Security Checklist

### Before Wiping Mac
- [ ] Logged out of GitHub in browser
- [ ] Logged out of all cloud services
- [ ] Removed SSH keys from GitHub (will add new ones from new laptop)
- [ ] Deauthorized any licensing tools
- [ ] Backed up any 2FA recovery codes

### After Transfer to Cloud
- [ ] Verified all files uploaded successfully
- [ ] Can access cloud storage from phone/web
- [ ] Made note of cloud storage password

---

## 11. Quick Info Sheet

Write down on paper (or in phone notes):

```
Personal Development Setup Info
================================

GitHub Username: zahra-abedi
GitHub Email: _________________
GitHub Repository: github.com/zahra-abedi/distributed-rate-limiter

Cloud Backup Location: _________________
Cloud Storage Username: _________________

Git Config:
- Name: _________________
- Email: _________________

Important Notes:
_________________________________
_________________________________
_________________________________
```

- [ ] Filled out info sheet
- [ ] Saved in phone notes or on paper

---

## 12. Verification Before Wipe

### Double-Check Everything
```bash
# Verify GitHub has latest code
cd ~/personal/distributed-rate-limiter
git status
git log -1  # See last commit
git remote -v  # Verify remote

# List all files on Desktop to transfer
ls -lh ~/Desktop/
```

- [ ] All code pushed to GitHub
- [ ] All backup files on Desktop
- [ ] All backup files uploaded to cloud
- [ ] Can access cloud storage from another device
- [ ] Setup guides saved

### Final Verification Checklist
- [ ] ✅ Git config backed up
- [ ] ✅ SSH keys backed up (if existed)
- [ ] ✅ Project code pushed to GitHub
- [ ] ✅ VS Code settings backed up
- [ ] ✅ Shell configs backed up
- [ ] ✅ Browser bookmarks exported
- [ ] ✅ Development notes created
- [ ] ✅ All files uploaded to cloud
- [ ] ✅ Can access cloud storage
- [ ] ✅ DEV-SETUP-WINDOWS.md saved
- [ ] ✅ SETUP-CHECKLIST.md saved

---

## 13. On Your New Dell Laptop

### First Steps
1. Download setup guides from cloud storage
2. Follow DEV-SETUP-WINDOWS.md step by step
3. Use SETUP-CHECKLIST.md to track progress
4. Restore SSH keys (if you backed them up) or create new ones
5. Clone repository from GitHub
6. Restore any custom configs

### Don't Forget
- Delete sensitive files (especially SSH keys) from cloud storage after transferring to new laptop
- Update SSH keys on GitHub if you created new ones
- Test everything works before fully transitioning

---

## Emergency Contacts

If you realize you forgot something after returning the Mac:
- All code is safely on GitHub
- You can always regenerate SSH keys
- Git config is just name/email (easy to remember)
- Most tools can be reinstalled

**Don't stress!** The most important thing (your code) is on GitHub. Everything else can be recreated.

---

## Quick Command Reference

**View git config:**
```bash
git config --list
```

**Check SSH keys:**
```bash
ls -la ~/.ssh/
```

**Current Git status:**
```bash
git status
git branch -vv
```

**Installed tools:**
```bash
go version
git --version
docker --version
```

---

**✓ Ready to return Mac when all items checked!**

Good luck with your new setup! 🚀
