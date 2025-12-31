# Emoji Reactions Implementation Plan

This document outlines the server-side changes needed to implement proper emoji reactions in Tinode.

---

## Research Summary

### How `{note}` Messages Work

1. **Client sends** `{note}` message to session
2. **`session.go:note()`** validates the message and routes to topic
3. **`topic.go:handleNote()`** processes based on `what` field:
   - `"read"` - updates ReadSeqId, broadcasts to subscribers
   - `"recv"` - updates RecvSeqId, broadcasts to subscribers  
   - `"kp"` - typing indicator, just broadcasts
   - `"call"` - video call events

### Current Valid `what` Values

From `server/session.go` lines 1256-1278:
```go
switch msg.Note.What {
case "data":      // payload required
case "kp", "kpa", "kpv":  // typing indicators
case "call":      // video calls (P2P only)
case "read", "recv":  // read receipts
default:
    return  // IGNORED - unknown type
}
```

### Message Storage

From `server/store/types/types.go` lines 1212-1227:
```go
type Message struct {
    ObjHeader
    DeletedAt  *time.Time
    DelId      int
    DeletedFor []SoftDelete
    SeqId      int
    Topic      string
    From       string
    Head       KVMap    // <-- Could store reactions here
    Content    any
}
```

### Database Schema (PostgreSQL)

From `server/db/postgres/adapter.go` line 496-500:
```sql
CREATE TABLE messages(
    ...
    head      JSON,
    content   JSON,
    ...
);
```

---

## Implementation Options

### Option A: Store Reactions in Message `Head` Field

**Approach**: Add reactions to the existing `Head` JSON field of messages.

**Pros**:
- No database schema changes
- Reactions persist with the message
- Retrieved automatically when fetching messages

**Cons**:
- Need new adapter method to update message Head
- Reactions are tied to message, not separate entity

**Message format**:
```json
{
  "head": {
    "reactions": {
      "👍": ["usrAAA", "usrBBB"],
      "❤️": ["usrCCC"]
    }
  }
}
```

---

### Option B: New `{note what="react"}` Type

**Approach**: Add a new note type for reactions, store in message Head.

**Client sends**:
```json
{
  "note": {
    "topic": "usrXXX",
    "what": "react",
    "seq": 123,
    "reaction": "👍"
  }
}
```

**Server**:
1. Validates reaction (emoji only, length limit)
2. Updates message Head with reaction
3. Broadcasts `{info what="react"}` to topic subscribers

---

## Recommended Approach: Option B

This is the cleanest implementation that follows Tinode's existing patterns.

---

## Files That Need Changes

### 1. `server/datamodel.go`

**Add reaction field to MsgClientNote**:
```go
type MsgClientNote struct {
    Topic    string `json:"topic"`
    What     string `json:"what"`
    SeqId    int    `json:"seq,omitempty"`
    Unread   int    `json:"unread,omitempty"`
    Event    string `json:"event,omitempty"`
    Payload  json.RawMessage `json:"payload,omitempty"`
    Reaction string `json:"reaction,omitempty"`  // NEW: emoji reaction
}
```

**Add reaction field to MsgServerInfo**:
```go
type MsgServerInfo struct {
    Topic    string `json:"topic"`
    From     string `json:"from,omitempty"`
    What     string `json:"what"`
    SeqId    int    `json:"seq,omitempty"`
    Reaction string `json:"reaction,omitempty"`  // NEW
    // ... existing fields
}
```

---

### 2. `server/session.go`

**Update `note()` function to handle "react"**:

Around line 1256, add case for "react":
```go
switch msg.Note.What {
case "react":  // NEW
    if msg.Note.SeqId <= 0 || msg.Note.Reaction == "" {
        return
    }
    // Validate reaction is a valid emoji (optional)
case "data":
    // ... existing
```

---

### 3. `server/topic.go`

**Update `handleNote()` to process reactions**:

Around line 1163, add case for "react":
```go
switch msg.Note.What {
case "react":  // NEW
    if !mode.IsReader() {
        return
    }
case "kp", "kpa", "kpv":
    // ... existing
```

**Add new function `handleReaction()`**:
```go
func (t *Topic) handleReaction(msg *ClientComMessage) {
    // 1. Get the message from store
    // 2. Update Head.reactions
    // 3. Save updated message
    // 4. Broadcast {info what="react"} to subscribers
}
```

---

### 4. `server/store/store.go`

**Add method to MessagesPersistenceInterface**:
```go
type MessagesPersistenceInterface interface {
    Save(msg *types.Message, attachmentURLs []string, readBySender bool) (error, bool)
    DeleteList(...) error
    GetAll(...) ([]types.Message, error)
    GetDeleted(...) ([]types.Range, int, error)
    UpdateHead(topic string, seqId int, head types.KVMap) error  // NEW
}
```

**Implement UpdateHead in messagesMapper**:
```go
func (messagesMapper) UpdateHead(topic string, seqId int, head types.KVMap) error {
    return adp.MessageUpdateHead(topic, seqId, head)
}
```

---

### 5. `server/db/postgres/adapter.go`

**Add MessageUpdateHead method**:
```go
func (a *adapter) MessageUpdateHead(topic string, seqId int, head t.KVMap) error {
    ctx, cancel := a.getContext()
    if cancel != nil {
        defer cancel()
    }
    
    _, err := a.db.Exec(ctx,
        `UPDATE messages SET head=$1, updatedat=$2 WHERE topic=$3 AND seqid=$4`,
        head, t.TimeNow(), topic, seqId)
    return err
}
```

---

### 6. `server/pbconverter.go` (for gRPC support)

**Update protobuf converters** to handle reaction field in notes.

---

## Data Flow

```
Client                          Server                           Database
  |                                |                                 |
  |-- {note what:"react",         |                                 |
  |    seq:123, reaction:"👍"} -->|                                 |
  |                                |                                 |
  |                          session.note()                          |
  |                                |                                 |
  |                          topic.handleNote()                      |
  |                                |                                 |
  |                          topic.handleReaction()                  |
  |                                |                                 |
  |                                |-- store.Messages.GetOne() ---->|
  |                                |<-- message with Head -----------|
  |                                |                                 |
  |                                |   (update Head.reactions)       |
  |                                |                                 |
  |                                |-- store.Messages.UpdateHead -->|
  |                                |<-- success ---------------------|
  |                                |                                 |
  |<-- {info what:"react",        |                                 |
  |     seq:123, reaction:"👍",   |                                 |
  |     from:"usrAAA"} -----------|                                 |
  |                                |                                 |
```

---

## Reaction Storage Format

In the message `Head` field:
```json
{
  "reactions": {
    "👍": ["usrFIwY7_SAEAA"],
    "❤️": ["usrFIwYebqAEAA", "usrFIwY7_SAEAA"],
    "😂": ["usrFIwYebqAEAA"]
  }
}
```

---

## Toggle Behavior

When a user sends the same reaction twice, it should toggle (remove):
1. If user already has this reaction → remove it
2. If user doesn't have this reaction → add it

---

## Validation

1. **Reaction must be valid emoji** (optional, could allow any short string)
2. **SeqId must exist** in the topic
3. **User must have read permission** on the topic
4. **Reaction length limit** (e.g., max 8 characters for multi-codepoint emoji)

---

## Estimated Effort

| File | Changes | Complexity |
|------|---------|------------|
| `datamodel.go` | Add fields | Low |
| `session.go` | Add case | Low |
| `topic.go` | Add handler | Medium |
| `store/store.go` | Add interface method | Low |
| `db/postgres/adapter.go` | Add SQL method | Low |
| `pbconverter.go` | Update converters | Low |

**Total estimate**: ~4-6 hours of work

---

## Testing Plan

1. Send reaction to existing message
2. Verify reaction stored in database
3. Verify {info} broadcast to other subscribers
4. Toggle reaction off
5. Multiple users react to same message
6. React to non-existent message (should fail silently)
7. React without permission (should fail silently)

---

## Alternative: Simpler Client-Side Approach

If server changes are too complex, reactions can be done client-side using message replacement:

1. User reacts to message seq 123
2. Client fetches message 123
3. Client sends `{pub}` with `head.replace: ":123"` and updated reactions
4. Server stores as new message that replaces 123

**Pros**: No server changes
**Cons**: Creates new message IDs, more complex client logic
