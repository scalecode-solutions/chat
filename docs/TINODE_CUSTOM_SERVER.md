# Tinode Custom Server Documentation

This document describes our custom Tinode chat server deployment with PostgreSQL backend and encryption at rest.

---

## Server Overview

| Setting | Value |
|---------|-------|
| **Server URL** | `https://chat.degenerates.dev` |
| **WebSocket URL** | `wss://chat.degenerates.dev/v0/channels` |
| **Long Polling URL** | `https://chat.degenerates.dev/v0/channels/lp` |
| **gRPC Port** | `16060` (internal only) |
| **API Version** | `0.25` |
| **Docker Image** | `tinode-postgres-fixed:latest` (custom build) |
| **Database** | PostgreSQL 15 |
| **Encryption** | AES-256-GCM at rest |

---

## Security Configuration

### API Key
```
AQAAAAABAAAoeOI7tA3HsYvdzDhYhZJy
```

**API Key Salt**: `T713/rYYgW7g4m3vG6zGRh7+FM1t0T8j13koXScOAj4=`

### Message Encryption at Rest
All message content is encrypted in the database using AES-256-GCM.

| Setting | Value |
|---------|-------|
| **Algorithm** | AES-256-GCM |
| **Key Size** | 256-bit (32 bytes) |
| **Encryption Key** | `0kqXklqRQkY9Wx8FgM5mzYU1Iw1udhTIrDcY2H2Zdms=` |
| **Env Variable** | `MSG_ENCRYPTION_KEY` |

**Note**: This is encryption at rest, NOT end-to-end encryption. The server can decrypt messages.

### Database Credentials
| Setting | Value |
|---------|-------|
| **Host** | `tinode-postgres` (Docker network) |
| **Database** | `tinode` |
| **User** | `tinode` |
| **Password** | `tinode_secure_2024` |

---

## Custom Modifications

Our fork includes the following changes from upstream Tinode:

### 1. PostgreSQL Database Creation Fix
**File**: `server/db/postgres/adapter.go`

**Problem**: Official image fails when database already exists but has no schema.

**Fix**: Modified `CreateDb()` to catch PostgreSQL error code `42P04` (duplicate_database) and continue.

### 2. Message Encryption at Rest
**Files**: 
- `server/store/crypto.go` (new)
- `server/store/store.go` (modified)
- `docker/tinode/config.template` (modified)

**Features**:
- AES-256-GCM encryption of message content
- Transparent encrypt on save, decrypt on read
- Configurable via `MSG_ENCRYPTION_KEY` environment variable
- Backwards compatible (handles unencrypted messages gracefully)

---

## Docker Deployment

### Docker Compose Configuration
**Location**: `/opt/tinode/docker-compose.yml`

```yaml
services:
  db:
    image: postgres:15
    container_name: tinode-postgres
    restart: unless-stopped
    environment:
      POSTGRES_USER: tinode
      POSTGRES_PASSWORD: tinode_secure_2024
      POSTGRES_DB: tinode
    volumes:
      - tinode-postgres-data:/var/lib/postgresql/data
    networks:
      - tinode-net
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U tinode -d tinode"]
      interval: 5s
      timeout: 5s
      retries: 5

  tinode:
    image: tinode-postgres-fixed:latest
    container_name: tinode-srv
    restart: unless-stopped
    depends_on:
      db:
        condition: service_healthy
    ports:
      - "6060:6060"
    environment:
      POSTGRES_DSN: "postgresql://tinode:tinode_secure_2024@db:5432/tinode?sslmode=disable"
      MSG_ENCRYPTION_KEY: "0kqXklqRQkY9Wx8FgM5mzYU1Iw1udhTIrDcY2H2Zdms="
    networks:
      - tinode-net

volumes:
  tinode-postgres-data:

networks:
  tinode-net:
    driver: bridge
```

### Build Custom Image
```bash
ssh root@scalecode.dev "cd /opt/tinode-src && docker build -f docker/tinode/Dockerfile.custom -t tinode-postgres-fixed:latest ."
```

### Management Commands
```bash
# View logs
ssh root@scalecode.dev "docker logs tinode-srv -f"

# Restart
ssh root@scalecode.dev "cd /opt/tinode && docker compose restart"

# Stop
ssh root@scalecode.dev "cd /opt/tinode && docker compose down"

# Start
ssh root@scalecode.dev "cd /opt/tinode && docker compose up -d"

# Rebuild and redeploy
ssh root@scalecode.dev "cd /opt/tinode-src && docker build -f docker/tinode/Dockerfile.custom -t tinode-postgres-fixed:latest . && cd /opt/tinode && docker compose down && docker compose up -d"
```

---

## Test Users

| Username | Password | Email |
|----------|----------|-------|
| alice | alice123 | alice@example.com |
| bob | bob123 | bob@example.com |
| carol | carol123 | carol@example.com |
| dave | dave123 | dave@example.com |
| eve | eve123 | eve@example.com |
| frank | frank123 | frank@example.com |

---

## Protocol Quick Reference

### Connect and Authenticate
```javascript
// 1. Connect
const ws = new WebSocket('wss://chat.degenerates.dev/v0/channels');
ws.setRequestHeader('X-Tinode-APIKey', 'AQAAAAABAAAoeOI7tA3HsYvdzDhYhZJy');

// 2. Handshake
ws.send(JSON.stringify({hi: {id: "1", ver: "0.22"}}));

// 3. Login (basic auth)
const secret = btoa("alice:alice123");
ws.send(JSON.stringify({login: {id: "2", scheme: "basic", secret: secret}}));
```

### Message Types

**Client → Server**:
| Message | Description |
|---------|-------------|
| `hi` | Handshake |
| `login` | Authenticate |
| `acc` | Create/update account |
| `sub` | Subscribe to topic |
| `leave` | Unsubscribe |
| `pub` | Publish message |
| `get` | Query metadata/messages |
| `set` | Update metadata |
| `del` | Delete messages/topics |
| `note` | Typing/read notifications |

**Server → Client**:
| Message | Description |
|---------|-------------|
| `ctrl` | Response to request |
| `data` | Message content |
| `meta` | Topic metadata |
| `pres` | Presence notification |
| `info` | Typing/read receipts |

### Send a Message
```javascript
ws.send(JSON.stringify({
  pub: {
    id: "123",
    topic: "usrXXXXXXXXXX",
    content: "Hello!"
  }
}));
```

### Reply to a Message
```javascript
ws.send(JSON.stringify({
  pub: {
    id: "124",
    topic: "usrXXXXXXXXXX",
    head: {
      reply: ":45"  // Reply to message seq 45
    },
    content: "This is my reply"
  }
}));
```

### Read/Delivery Receipts
```javascript
// Mark as received
ws.send(JSON.stringify({note: {topic: "usrXXX", what: "recv", seq: 123}}));

// Mark as read
ws.send(JSON.stringify({note: {topic: "usrXXX", what: "read", seq: 123}}));

// Typing indicator
ws.send(JSON.stringify({note: {topic: "usrXXX", what: "kp"}}));
```

---

## Source Code Location

| Location | Description |
|----------|-------------|
| `/Users/tmarq/Github/tinchat/tinode-src` | Local source code |
| `/opt/tinode-src` | VPS source code |
| `/opt/tinode` | VPS docker-compose deployment |

### Key Modified Files
- `server/db/postgres/adapter.go` - PostgreSQL fix
- `server/store/crypto.go` - Encryption implementation
- `server/store/store.go` - Encrypt/decrypt on save/read
- `docker/tinode/Dockerfile.custom` - Custom build
- `docker/tinode/config.template` - Config with encryption key
- `IMPROVEMENTS.md` - Feature roadmap

---

## Generate New Keys

### New API Key
```bash
ssh root@scalecode.dev "docker exec tinode-srv /opt/tinode/keygen -salt 'T713/rYYgW7g4m3vG6zGRh7+FM1t0T8j13koXScOAj4='"
```

### New Encryption Key
```bash
ssh root@scalecode.dev "openssl rand -base64 32"
```

---

## SSL Certificate

| Setting | Value |
|---------|-------|
| **Domain** | chat.degenerates.dev |
| **Certificate** | `/etc/letsencrypt/live/chat.degenerates.dev/fullchain.pem` |
| **Key** | `/etc/letsencrypt/live/chat.degenerates.dev/privkey.pem` |
| **Auto-Renewal** | Enabled via Certbot |

---

## Future Improvements

See `tinode-src/IMPROVEMENTS.md` for the full roadmap. Key items:

1. ⏱️ **Disappearing Messages** - Auto-delete by timer (#941)
2. 👁️ **View Active Sessions** - See logged-in devices (#968)
3. 📍 **Location Sharing** - Share GPS coordinates (#963)
4. ⏰ **Delivery Timestamps** - Add timestamps to receipts
