# Tinode Server Issue - 500 Internal Errors

## Problem Summary

The Tinode chat server at `wss://chat.degenerates.dev/v0/channels` is returning **500 Internal Error** responses for login and topic subscription requests, despite the WebSocket connection and handshake working correctly.

---

## What's Working

1. **WebSocket Connection** - Successfully connects to `wss://chat.degenerates.dev/v0/channels?apikey=...`
2. **Handshake (`hi`)** - Returns 201 "created" with server info
3. **Account Registration (`acc`)** - Returns 200 "ok" with user token
4. **API Key** - Generated with correct salt (`T713/rYYgW7g4m3vG6zGRh7+FM1t0T8j13koXScOAj4=`)

## What's Failing

1. **Login (`login`)** - Returns 500 "internal error"
2. **Subscribe to `me` topic (`sub`)** - Returns 500 "internal error"

---

## Example Logs

```
LOG  Connecting to: wss://chat.degenerates.dev/v0/channels?apikey=AQAAAAABAAAoeOI7tA3HsYvdzDhYhZJy
LOG  WebSocket connected
LOG  Sending: {"hi":{"id":"109754","ver":"0.22","ua":"Preggo/1.0 (React Native)","lang":"en-US","api":"..."}}
LOG  Received: {"ctrl":{"id":"109754","params":{"build":"mysql:v0.25.1",...},"code":201,"text":"created",...}}
LOG  Sending: {"login":{"id":"109755","scheme":"basic","secret":"YWxpY2U6YWxpY2UxMjM="}}
LOG  Received: {"ctrl":{"id":"109755","code":500,"text":"internal error",...}}
ERROR  Login error: [Error: 500: internal error]
```

---

## Troubleshooting Steps Taken

### 1. API Key Issues (RESOLVED)
- Initially got 403 "valid API key required"
- Generated new API key with correct salt using:
  ```bash
  ssh root@scalecode.dev "docker exec tinode-srv /opt/tinode/keygen -salt 'T713/rYYgW7g4m3vG6zGRh7+FM1t0T8j13koXScOAj4='"
  ```
- Added API key to WebSocket URL as query parameter

### 2. Server Restart
- Restarted Tinode containers:
  ```bash
  ssh root@scalecode.dev "cd /opt/tinode && docker compose restart"
  ```
- Also tried full down/up cycle
- Server starts successfully, database connects

### 3. Database Verification
- Confirmed MySQL database is healthy
- Users table has data (alice, bob, carol, dave, eve, frank)
- Auth table has bcrypt-hashed passwords
- Schema looks correct

### 4. Nginx Proxy Check
- Nginx config looks correct with WebSocket upgrade headers
- Direct connection to `127.0.0.1:6060` on server works

### 5. Server Logs
- Tinode server logs show NO errors
- Only shows startup messages:
  ```
  Database adapter: 'mysql'; version: 116
  Database exists, version is correct.
  Sample data ignored.
  All done.
  ```

---

## Server Configuration

| Setting | Value |
|---------|-------|
| **Server URL** | `https://chat.degenerates.dev` |
| **WebSocket URL** | `wss://chat.degenerates.dev/v0/channels` |
| **Database** | MySQL 8.0 |
| **Tinode Image** | `tinode/tinode-mysql:latest` |
| **API Key Salt** | `T713/rYYgW7g4m3vG6zGRh7+FM1t0T8j13koXScOAj4=` |

---

## Possible Causes

1. **Tinode server internal bug** - The 500 error is not being logged
2. **Database connection issue** - Server connects at startup but may have issues during queries
3. **Configuration mismatch** - Something in `working.config` may be incorrect
4. **Version incompatibility** - Using `latest` tag, may have breaking changes

---

## Next Steps to Try

1. Enable verbose logging in Tinode server config
2. Check `working.config` for any misconfigurations
3. Try pinning to a specific Tinode version instead of `latest`
4. Check MySQL query logs for errors during login attempts
5. Test with Tinode's official web client to isolate if it's a server or client issue

---

## Files Modified (Client Side)

- `src/services/tinode.ts` - Replaced tinode-sdk with native WebSocket implementation
- `src/screens/ChatLoginScreen.tsx` - Added login/register UI
- `src/screens/ChatListScreen.tsx` - Wired up to Tinode contacts
- `src/screens/ChatScreen.tsx` - Wired up to Tinode messaging
- `src/navigation/AppNavigator.tsx` - Added chat auth flow
