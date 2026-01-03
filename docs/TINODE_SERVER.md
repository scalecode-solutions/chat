# Tinode Chat Server Documentation

## Server Information

| Setting | Value |
|---------|-------|
| **Server URL** | `https://chat.degenerates.dev` |
| **WebSocket URL** | `wss://chat.degenerates.dev/v0/channels` |
| **Long Polling URL** | `https://chat.degenerates.dev/v0/channels` |
| **gRPC Port** | `16060` (internal only) |
| **API Version** | `v0` |
| **Database** | PostgreSQL 15 |
| **Docker Image** | `tinode-postgres-fixed:latest` (custom build) |

## API Key

The server requires a valid API key for all requests. Generate one using:

```bash
ssh root@scalecode.dev "docker exec tinode-srv /opt/tinode/keygen"
```

**Current API Key Salt**: `T713/rYYgW7g4m3vG6zGRh7+FM1t0T8j13koXScOAj4=`

**Valid API Key**: `AQAAAAABAAAoeOI7tA3HsYvdzDhYhZJy`

To use the API key, include it in the `hi` (handshake) message when connecting.

---

## Default Test Users

The server was initialized with sample data. These accounts are available for testing:

| Username | Password | Email |
|----------|----------|-------|
| alice | alice123 | alice@example.com |
| bob | bob123 | bob@example.com |
| carol | carol123 | carol@example.com |
| dave | dave123 | dave@example.com |
| eve | eve123 | eve@example.com |
| frank | frank123 | frank@example.com |
| tino | *(random - check logs)* | tino@example.com |

---

## Connection Protocol

Tinode uses a custom JSON-based protocol over WebSocket or long polling.

### 1. Establish Connection

Connect to WebSocket:
```javascript
const ws = new WebSocket('wss://chat.degenerates.dev/v0/channels');
```

### 2. Handshake (`hi`)

Send immediately after connection:
```json
{
  "hi": {
    "id": "123",
    "ver": "0.22",
    "ua": "MyApp/1.0",
    "lang": "en-US"
  }
}
```

**Response:**
```json
{
  "ctrl": {
    "id": "123",
    "code": 201,
    "text": "created",
    "ts": "2025-12-31T12:00:00.000Z",
    "params": {
      "ver": "0.22.14",
      "build": "..."
    }
  }
}
```

### 3. Login (`login`)

Authenticate with username/password:
```json
{
  "login": {
    "id": "124",
    "scheme": "basic",
    "secret": "base64(username:password)"
  }
}
```

**Example** (for user `alice` with password `alice123`):
```json
{
  "login": {
    "id": "124",
    "scheme": "basic",
    "secret": "YWxpY2U6YWxpY2UxMjM="
  }
}
```

**Response:**
```json
{
  "ctrl": {
    "id": "124",
    "code": 200,
    "text": "ok",
    "params": {
      "user": "usrXXXXXXXXXX",
      "authlvl": "auth"
    }
  }
}
```

### 4. Create Account (`acc`)

Register a new user:
```json
{
  "acc": {
    "id": "125",
    "user": "new",
    "scheme": "basic",
    "secret": "base64(username:password)",
    "login": true,
    "desc": {
      "public": {
        "fn": "Display Name"
      }
    }
  }
}
```

---

## Message Types

### Client to Server

| Message | Description |
|---------|-------------|
| `hi` | Handshake, establish session parameters |
| `acc` | Create/update account |
| `login` | Authenticate |
| `sub` | Subscribe to topic |
| `leave` | Unsubscribe from topic |
| `pub` | Publish message to topic |
| `get` | Query topic metadata/messages |
| `set` | Update topic metadata |
| `del` | Delete messages/topics/subscriptions |
| `note` | Send notification (typing, read receipt) |

### Server to Client

| Message | Description |
|---------|-------------|
| `ctrl` | Response to client request |
| `data` | Content message |
| `meta` | Topic metadata |
| `pres` | Presence notification |
| `info` | Notification (typing, read receipt) |

---

## Topic Types

| Prefix | Type | Description |
|--------|------|-------------|
| `me` | Me | User's own topic for notifications |
| `fnd` | Find | Search/discovery topic |
| `usr*` | P2P | Direct message with another user |
| `grp*` | Group | Group chat topic |
| `sys` | System | System notifications (admin only) |

---

## Subscribe to Topics

### Subscribe to `me` topic (required for notifications):
```json
{
  "sub": {
    "id": "126",
    "topic": "me"
  }
}
```

### Subscribe to a group or P2P topic:
```json
{
  "sub": {
    "id": "127",
    "topic": "grpXXXXXXXXXX",
    "get": {
      "what": "desc sub data"
    }
  }
}
```

---

## Send Messages

### Publish to a topic:
```json
{
  "pub": {
    "id": "128",
    "topic": "grpXXXXXXXXXX",
    "content": "Hello, world!"
  }
}
```

### Send typing notification:
```json
{
  "note": {
    "topic": "grpXXXXXXXXXX",
    "what": "kp"
  }
}
```

### Send read receipt:
```json
{
  "note": {
    "topic": "grpXXXXXXXXXX",
    "what": "read",
    "seq": 123
  }
}
```

---

## Server Management

### Docker Compose Location
```
/opt/tinode/docker-compose.yml
```

### View Logs
```bash
ssh root@scalecode.dev "docker logs tinode-srv -f"
```

### Restart Server
```bash
ssh root@scalecode.dev "cd /opt/tinode && docker compose restart"
```

### Stop Server
```bash
ssh root@scalecode.dev "cd /opt/tinode && docker compose down"
```

### Start Server
```bash
ssh root@scalecode.dev "cd /opt/tinode && docker compose up -d"
```

### Generate New API Key
```bash
ssh root@scalecode.dev "docker exec tinode-srv /opt/tinode/keygen"
```

---

## Database

| Setting | Value |
|---------|-------|
| **Type** | PostgreSQL 15 |
| **Container** | `tinode-postgres` |
| **Database** | `tinode` |
| **User** | `tinode` |
| **Password** | `tinode_secure_2024` |

### Access PostgreSQL CLI
```bash
ssh root@scalecode.dev "docker exec -it tinode-postgres psql -U tinode -d tinode"
```

---

## Configuration

The main config file is at `/opt/tinode/tinode.conf` inside the container.

### Key Settings

| Setting | Value |
|---------|-------|
| **Listen Port** | 6060 |
| **Max Message Size** | 128KB |
| **Max Subscribers/Topic** | 128 |
| **Max Tags** | 16 |
| **Media Max Size** | 8MB |

### Mount Custom Config

To customize, create a config file and mount it:

```yaml
# In docker-compose.yml, add to tinode service:
volumes:
  - ./tinode.conf:/opt/tinode/tinode.conf
```

---

## SSL Certificate

| Setting | Value |
|---------|-------|
| **Domain** | chat.degenerates.dev |
| **Certificate** | `/etc/letsencrypt/live/chat.degenerates.dev/fullchain.pem` |
| **Key** | `/etc/letsencrypt/live/chat.degenerates.dev/privkey.pem` |
| **Expires** | 2026-03-31 |
| **Auto-Renewal** | Enabled via Certbot |

---

## Custom Build Notes

This server uses a custom-built Tinode Docker image (`tinode-postgres-fixed:latest`) that fixes a bug in the official `tinode/tinode-postgres` image where database initialization fails if the database already exists but has no schema.

**Fix location**: `server/db/postgres/adapter.go` - Modified `CreateDb()` to catch PostgreSQL error code `42P04` (duplicate_database) and continue instead of failing.

**Source code**: `/opt/tinode-src` on the VPS

**Rebuild command**:
```bash
ssh root@scalecode.dev "cd /opt/tinode-src && docker build -f docker/tinode/Dockerfile.custom -t tinode-postgres-fixed:latest ."
```

---

## Client Libraries & Resources

- **Official Web Client**: https://github.com/nicholasareed/nicholasareed.github.io
- **Tindroid (Android)**: https://github.com/nicholasareed/nicholasareed.github.io
- **Tinodios (iOS)**: https://github.com/nicholasareed/nicholasareed.github.io
- **JavaScript SDK**: https://github.com/nicholasareed/nicholasareed.github.io
- **Protocol Docs**: https://github.com/nicholasareed/nicholasareed.github.io

---

## Quick Test

Test the server is responding:
```bash
curl -s https://chat.degenerates.dev/v0/channels
# Expected: {"ctrl":{"code":403,"text":"valid API key required",...}}
```

Test WebSocket connection (using websocat):
```bash
websocat wss://chat.degenerates.dev/v0/channels
# Then send: {"hi":{"id":"1","ver":"0.22"}}
```
