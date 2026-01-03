# Tinode Emoji Reactions

This document explains how emoji reactions work in our custom Tinode server.

---

## Overview

Emoji reactions allow users to react to messages with emojis (👍, ❤️, 😂, etc.). Reactions are stored in the message's `head` field and support toggle behavior (tap again to remove).

---

## How It Works

### Client Sends Reaction

```json
{
  "note": {
    "topic": "usrVqvAnkvyYco",
    "what": "react",
    "seq": 123,
    "reaction": "👍"
  }
}
```

| Field | Description |
|-------|-------------|
| `topic` | The topic/user to react in |
| `what` | Must be `"react"` |
| `seq` | Message sequence ID to react to |
| `reaction` | The emoji reaction (max 32 chars) |

### Server Behavior

1. **Validates** the request (valid seq, user has read permission)
2. **Toggles** the reaction:
   - If user already has this reaction → removes it
   - If user doesn't have this reaction → adds it
3. **Updates** the message's `head.reactions` in the database
4. **Broadcasts** `{info what:"react"}` to all topic subscribers

### Server Response (Broadcast)

All subscribers receive:

```json
{
  "info": {
    "topic": "usrVqvAnkvyYco",
    "from": "usrAAA",
    "what": "react",
    "seq": 123,
    "reaction": "👍"
  }
}
```

---

## Storage Format

Reactions are stored in the message's `head` field:

```json
{
  "head": {
    "reactions": {
      "👍": ["usrAAA", "usrBBB"],
      "❤️": ["usrCCC"],
      "😂": ["usrAAA", "usrCCC"]
    }
  }
}
```

- Keys are emoji strings
- Values are arrays of user IDs who reacted with that emoji
- Empty reaction arrays are automatically removed

---

## Retrieving Reactions

When fetching messages with `{get what:"data"}`, reactions come back in the message head:

```json
{
  "data": {
    "topic": "usrVqvAnkvyYco",
    "seq": 123,
    "head": {
      "reactions": {
        "👍": ["usrAAA"]
      }
    },
    "content": "Hello world"
  }
}
```

---

## Client Implementation

### React to a Message

```javascript
function reactToMessage(topic, seqId, emoji) {
  ws.send(JSON.stringify({
    note: {
      topic: topic,
      what: "react",
      seq: seqId,
      reaction: emoji
    }
  }));
}

// Example: React with thumbs up to message 5
reactToMessage("usrVqvAnkvyYco", 5, "👍");
```

### Handle Incoming Reactions

```javascript
ws.onmessage = (event) => {
  const msg = JSON.parse(event.data);
  
  if (msg.info && msg.info.what === "react") {
    // Someone reacted to a message
    const { topic, from, seq, reaction } = msg.info;
    console.log(`${from} reacted with ${reaction} to message ${seq}`);
    
    // Update your local message state
    updateMessageReaction(topic, seq, from, reaction);
  }
};
```

### Display Reactions

```javascript
function renderReactions(reactions) {
  if (!reactions) return null;
  
  return Object.entries(reactions).map(([emoji, users]) => ({
    emoji: emoji,
    count: users.length,
    users: users,
    // Check if current user reacted
    userReacted: users.includes(currentUserId)
  }));
}
```

---

## Toggle Behavior

Sending the same reaction twice removes it:

```javascript
// First tap: adds 👍
reactToMessage("usrXXX", 5, "👍");
// Database: {"reactions": {"👍": ["usrAAA"]}}

// Second tap: removes 👍
reactToMessage("usrXXX", 5, "👍");
// Database: {"reactions": {}}
```

---

## Validation Rules

| Rule | Limit |
|------|-------|
| Max reaction length | 32 characters |
| Valid seq range | 1 to topic's lastID |
| Permission required | Read access to topic |

---

## Database Query

To view reactions directly in the database:

```sql
SELECT seqid, topic, head->'reactions' as reactions 
FROM messages 
WHERE head->'reactions' IS NOT NULL;
```

---

## Example Session

```
Client                              Server
  |                                    |
  |-- {note what:"react",             |
  |    seq:5, reaction:"👍"} -------->|
  |                                    |
  |                          (update DB)
  |                                    |
  |<-- {info what:"react",            |
  |     seq:5, reaction:"👍",         |
  |     from:"usrAAA"} ---------------|
  |                                    |
```

---

## Files Modified (Server)

| File | Changes |
|------|---------|
| `server/datamodel.go` | Added `Reaction` field to note structs |
| `server/session.go` | Added `"react"` case validation |
| `server/topic.go` | Added `handleReaction()` function |
| `server/store/store.go` | Added `AddReaction` interface method |
| `server/db/postgres/adapter.go` | Implemented reaction storage |
| `pbx/model.proto` | Added `REACT` enum and `reaction` fields |

---

## Rollback

If issues occur, rollback to the backup image:

```bash
docker tag tinode-postgres-fixed:backup-20251231-161111 tinode-postgres-fixed:latest
cd /opt/tinode && docker compose down && docker compose up -d
```
