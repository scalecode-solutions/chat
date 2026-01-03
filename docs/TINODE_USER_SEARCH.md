# Tinode User Search Guide

This document explains how user discovery works in Tinode and how to find other users.

---

## How User Search Works

Tinode uses a **tag-based discovery system**. Users are found by searching for tags, not by display names directly.

### Tag Types

| Tag Format | Example | Description |
|------------|---------|-------------|
| `basic:username` | `basic:travis` | Auto-created from login username |
| `username` | `travis` | Simple searchable tag (manually added) |
| `email:user@example.com` | `email:travis@gmail.com` | Email-based discovery |
| `tel:+1234567890` | `tel:+15551234567` | Phone number discovery |

---

## Finding Users in the App

### Method 1: Search by Username Tag
In the app's search/find feature, type the username:
```
travis
```
or
```
shelby
```

### Method 2: Search by User ID
If you know the Tinode user ID, you can use it directly:
```
usrFIwY7_SAEAA
```

### Method 3: Search with Auth Prefix
The full tag includes the auth scheme:
```
basic:travis
```

---

## Current Users

| Name | User ID | Searchable Tags |
|------|---------|-----------------|
| shelby | `usrFIwYebqAEAA` | `shelby`, `basic:shelby` |
| travis | `usrFIwY7_SAEAA` | `travis`, `basic:travis` |

---

## Adding Searchable Tags (Admin)

To make a user discoverable by a custom tag, add it to the database:

```sql
-- Add a simple name tag
INSERT INTO usertags (userid, tag) VALUES (<user_id>, 'tagname');

-- Example: Make user discoverable by nickname
INSERT INTO usertags (userid, tag) VALUES (1480585796376334336, 'trav');
```

### Via SSH:
```bash
ssh root@scalecode.dev "docker exec tinode-postgres psql -U tinode -d tinode -c \"INSERT INTO usertags (userid, tag) VALUES (<user_id>, 'newtag');\""
```

---

## How the Find Topic Works

Tinode uses a special `fnd` (find) topic for user discovery:

1. **Subscribe to fnd topic**:
   ```json
   {"sub": {"id": "1", "topic": "fnd"}}
   ```

2. **Set search query**:
   ```json
   {"set": {"id": "2", "topic": "fnd", "desc": {"public": "travis"}}}
   ```

3. **Get results**:
   ```json
   {"get": {"id": "3", "topic": "fnd", "what": "sub"}}
   ```

4. **Server returns matching users** in a `{meta}` message with subscription info.

---

## Troubleshooting

### "User not found"
- Check if the user has the tag you're searching for
- Try searching with `basic:` prefix
- Verify the user exists: 
  ```bash
  ssh root@scalecode.dev "docker exec tinode-postgres psql -U tinode -d tinode -c \"SELECT * FROM usertags;\""
  ```

### User exists but not discoverable
Add a simple tag without the auth prefix:
```sql
INSERT INTO usertags (userid, tag) VALUES (<user_id>, 'username');
```

---

## API Reference

### Find Users (WebSocket)
```javascript
// 1. Subscribe to fnd topic
ws.send(JSON.stringify({sub: {id: "1", topic: "fnd"}}));

// 2. Set search query
ws.send(JSON.stringify({
  set: {
    id: "2", 
    topic: "fnd", 
    desc: {public: "travis"}
  }
}));

// 3. Get results
ws.send(JSON.stringify({
  get: {
    id: "3", 
    topic: "fnd", 
    what: "sub"
  }
}));

// 4. Listen for {meta} response with matched users
```

### Start P2P Chat
Once you have the user ID, subscribe to start a chat:
```javascript
ws.send(JSON.stringify({
  sub: {
    id: "4", 
    topic: "usrFIwY7_SAEAA"  // The other user's ID
  }
}));
```
