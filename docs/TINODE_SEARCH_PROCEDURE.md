# Tinode User Search Procedure (Corrected)

This document describes the **correct** procedure for searching users in Tinode using the `fnd` (find) topic.

---

## Overview

Tinode uses a **tag-based discovery system**. The `fnd` topic searches for users by matching **tags**, not display names. The search query must be a **string**, not an object.

---

## Key Correction

**WRONG** (what the app was doing):
```json
{"set": {"topic": "fnd", "desc": {"public": {"fn": "travis"}}}}
```

**CORRECT** (what Tinode expects):
```json
{"set": {"topic": "fnd", "desc": {"public": "travis"}}}
```

The `public` field must be a **string** (the search query), not an object.

---

## Step-by-Step Procedure

### Step 1: Check if Already a User ID
```typescript
if (username.startsWith('usr')) {
  // Skip search, use directly
  await subscribeTopic(username);
  return username;
}
```

### Step 2: Subscribe to `fnd` Topic
```json
{
  "sub": {
    "id": "1",
    "topic": "fnd"
  }
}
```
**Response**: `{"ctrl": {"code": 200}}` or `{"ctrl": {"code": 304}}` (already subscribed)

### Step 3: Set Search Query (CORRECTED)
```json
{
  "set": {
    "id": "2",
    "topic": "fnd",
    "desc": {
      "public": "travis"
    }
  }
}
```

**Important**: `public` is a **string** containing the search term, NOT an object like `{"fn": "travis"}`.

**Response**: `{"ctrl": {"code": 200}}`

### Step 4: Get Search Results
```json
{
  "get": {
    "id": "3",
    "topic": "fnd",
    "what": "sub"
  }
}
```

### Step 5: Parse Results

**If user found**:
```json
{
  "meta": {
    "topic": "fnd",
    "sub": [
      {
        "topic": "usrVqvAnkvyYco",
        "public": {"fn": "shelby"},
        "updated": "2025-12-31T20:23:23.119Z",
        "acs": {"mode": "JRWPAS"},
        "private": []
      }
    ]
  }
}
```

**If no user found**:
```json
{
  "ctrl": {
    "topic": "fnd",
    "code": 204,
    "text": "no content"
  }
}
```

### Step 6: Extract User ID
The user ID is in `meta.sub[0].topic` (e.g., `usrVqvAnkvyYco`).

---

## Search Query Syntax

The search query supports multiple terms and operators:

| Query | Meaning |
|-------|---------|
| `travis` | Find users with tag containing "travis" |
| `travis shelby` | Find users matching "travis" OR "shelby" |
| `basic:travis` | Find users with exact tag "basic:travis" |
| `email:user@example.com` | Find by email tag |
| `tel:+15551234567` | Find by phone tag |

---

## Current Users

| Name | User ID | Searchable Tags |
|------|---------|-----------------|
| shelby | `usrFIwYebqAEAA` | `shelby`, `basic:shelby` |
| travis | `usrFIwY7_SAEAA` | `travis`, `basic:travis` |

---

## Complete Working Example

```javascript
// 1. Subscribe to fnd
ws.send(JSON.stringify({sub: {id: "1", topic: "fnd"}}));
// Wait for ctrl 200/304

// 2. Set search query (public is a STRING)
ws.send(JSON.stringify({
  set: {
    id: "2",
    topic: "fnd",
    desc: {
      public: "travis"  // <-- STRING, not object!
    }
  }
}));
// Wait for ctrl 200

// 3. Get results
ws.send(JSON.stringify({
  get: {
    id: "3",
    topic: "fnd",
    what: "sub"
  }
}));
// Wait for meta with sub array

// 4. Extract user ID from response
// response.meta.sub[0].topic = "usrVqvAnkvyYco"
```

---

## Message Flow Diagram

```
Client                              Server
  |                                    |
  |-- sub {topic: "fnd"} ------------>|
  |<-- ctrl {code: 200} --------------|
  |                                    |
  |-- set {topic: "fnd",              |
  |       desc: {public: "travis"}} ->|  <-- STRING not object
  |<-- ctrl {code: 200} --------------|
  |                                    |
  |-- get {topic: "fnd", what: "sub"}->|
  |<-- meta {sub: [{topic: "usrXXX"}]}-|  (if found)
  |<-- ctrl {code: 204} --------------|  (if not found)
  |                                    |
```

---

## Troubleshooting

### "204 no content" when user exists

1. **Check tags exist**:
   ```bash
   ssh root@scalecode.dev "docker exec tinode-postgres psql -U tinode -d tinode -c \"SELECT * FROM usertags;\""
   ```

2. **Add missing tag**:
   ```bash
   ssh root@scalecode.dev "docker exec tinode-postgres psql -U tinode -d tinode -c \"INSERT INTO usertags (userid, tag) VALUES (<user_id>, 'username');\""
   ```

3. **Verify search format**: Make sure `public` is a string, not an object.

---

## Related Files

- `src/services/tinode.ts` - Search implementation
- `src/screens/NewChatScreen.tsx` - UI that calls search
- `TINODE_USER_SEARCH.md` - Server-side tag documentation
- `tinode-src/docs/API.md` - Official Tinode API reference
