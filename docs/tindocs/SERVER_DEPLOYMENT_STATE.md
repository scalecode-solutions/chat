# Server Deployment State

**Last Updated:** January 3, 2026

## Server: scalecode.dev

### Directory Structure

| Path | Description |
|------|-------------|
| `/opt/tinode-src` | Tinode server source code |
| `/opt/tinode` | Docker compose deployment |

### Current Source Code State

**Location:** `/opt/tinode-src`

**Git Remote:**
```
origin  https://github.com/scalecode-solutions/chat.git (fetch)
origin  https://github.com/scalecode-solutions/chat.git (push)
```

✅ **Server is now using the scalecode-solutions fork.**

**Branch:** `main`

**All changes are now committed and tracked in the fork.**

**Features implemented:**
- Emoji reactions
- Reply/quote validation
- Edit/unsend messages (max 10 edits in 15min, unsend within 10min)
- Message encryption at rest

### Docker Setup

**Docker Compose:** `/opt/tinode/docker-compose.yml`

**Services:**
1. **db** - PostgreSQL 15 (`tinode-postgres`)
2. **tinode** - Custom image (`tinode-postgres-fixed:latest`)

**Docker Images:**
| Image | Tag | Size |
|-------|-----|------|
| tinode-postgres-fixed | latest | 93.9MB |
| tinode-postgres-fixed | backup-20251231-161111 | 93.8MB |

### Deployment Method

1. **Pull** latest changes from fork:
   ```bash
   ssh root@scalecode.dev "cd /opt/tinode-src && git pull origin main"
   ```
2. **Build** Docker image on server:
   ```bash
   ssh root@scalecode.dev "cd /opt/tinode-src && docker build -f docker/tinode/Dockerfile.custom -t tinode-postgres-fixed:latest ."
   ```
3. **Deploy** with docker compose:
   ```bash
   ssh root@scalecode.dev "cd /opt/tinode && docker compose down && docker compose up -d"
   ```

### Feature Status on Server

| Feature | Status | Notes |
|---------|--------|-------|
| Emoji Reactions | ✅ Deployed | Proto + handler changes applied |
| Reply/Quote Validation | ✅ Deployed | Validates reply seq references |
| Edit/Unsend Messages | ✅ Deployed | Max 10 edits/15min, unsend within 10min |
| Message Encryption | ✅ Deployed | AES encryption at rest |

---

## Local Development Setup

**Local repo remotes (`/Users/tmarq/Github/tinchat/tinode-src`):**
- `origin` → `https://github.com/scalecode-solutions/chat.git` (your fork - default push)
- `upstream` → `https://github.com/tinode/chat.git` (original Tinode - read-only)

**Workflow:**
1. Make changes locally
2. Commit and push to `origin` (your fork)
3. SSH to server and `git pull origin main`
4. Rebuild Docker image and deploy
