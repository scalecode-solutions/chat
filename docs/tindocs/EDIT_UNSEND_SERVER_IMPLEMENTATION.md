# Edit/Unsend Messages - Server Implementation

**Implemented:** January 3, 2026  
**Branch:** `main`  
**Commit:** `1ad41c37`

---

## Overview

This document details the server-side implementation of the Edit/Unsend Messages feature. This feature allows users to:
- **Edit** their own messages (max 10 edits within 15 minutes of sending)
- **Unsend** their own messages (within 10 minutes of sending)

---

## Constraints

| Constraint | Value | Description |
|------------|-------|-------------|
| Edit Time Window | 15 minutes | Can only edit within 15 min of original message |
| Max Edit Count | 10 | Maximum number of times a message can be edited |
| Unsend Time Window | 10 minutes | Can only unsend within 10 min of original message |
| Ownership | Sender only | Only the message sender can edit/unsend |

---

## Files Modified

| File | Changes |
|------|---------|
| `server/datamodel.go` | Added `Content` field to `MsgClientNote`, added `Content`/`EditedAt` to `MsgServerInfo` |
| `server/topic.go` | Added `handleEdit()` and `handleUnsend()` handlers |
| `server/store/store.go` | Added `GetBySeqId`, `Edit`, `MarkUnsent` interface methods |
| `server/db/adapter.go` | Added adapter interface methods |
| `server/db/postgres/adapter.go` | Implemented PostgreSQL functions |

---

## Protocol

### Client → Server (Edit)

Client sends a `{note}` message with `what: "edit"`:

```json
{
  "note": {
    "topic": "usrXXX",
    "what": "edit",
    "seq": 123,
    "content": "Updated message text"
  }
}
```

### Client → Server (Unsend)

Client sends a `{note}` message with `what: "unsend"`:

```json
{
  "note": {
    "topic": "usrXXX",
    "what": "unsend",
    "seq": 123
  }
}
```

### Server → Clients (Edit Broadcast)

Server broadcasts `{info}` to all topic subscribers:

```json
{
  "info": {
    "topic": "usrXXX",
    "from": "usrYYY",
    "what": "edit",
    "seq": 123,
    "content": "Updated message text",
    "edited_at": "2026-01-03T12:00:00Z"
  }
}
```

### Server → Clients (Unsend Broadcast)

```json
{
  "info": {
    "topic": "usrXXX",
    "from": "usrYYY",
    "what": "unsend",
    "seq": 123
  }
}
```

---

## Data Model Changes

### MsgClientNote (datamodel.go:313-326)

Added `Content` field to support sending new message content with edit requests:

```go
type MsgClientNote struct {
    Topic    string `json:"topic"`
    What     string `json:"what"`           // "kp", "read", "recv", "call", "react", "edit", "unsend"
    SeqId    int    `json:"seq,omitempty"`
    Unread   int    `json:"unread,omitempty"`
    Event    string `json:"event,omitempty"`
    Payload  any    `json:"payload,omitempty"`
    Reaction string `json:"reaction,omitempty"`
    Content  any    `json:"content,omitempty"`  // NEW: For edit messages
}
```

### MsgServerInfo (datamodel.go:793-806)

Added `Content` and `EditedAt` fields for broadcasting edits:

```go
type MsgServerInfo struct {
    Topic    string     `json:"topic"`
    Src      string     `json:"src,omitempty"`
    From     string     `json:"from,omitempty"`
    What     string     `json:"what"`           // Includes "edit", "unsend"
    SeqId    int        `json:"seq,omitempty"`
    Reaction string     `json:"reaction,omitempty"`
    Content  any        `json:"content,omitempty"`    // NEW: Edited content
    EditedAt *time.Time `json:"edited_at,omitempty"`  // NEW: Edit timestamp
}
```

---

## Handler Implementation

### handleEdit (topic.go)

```go
func (t *Topic) handleEdit(msg *ClientComMessage) {
    asUid := types.ParseUserId(msg.AsUser)
    seqId := msg.Note.SeqId
    newContent := msg.Note.Content

    // 1. Validate SeqId range
    if seqId > t.lastID || seqId <= 0 {
        return
    }

    // 2. Validate content not empty
    if newContent == nil {
        return
    }

    // 3. Get original message
    origMsg, err := store.Messages.GetBySeqId(t.name, seqId)
    if err != nil || origMsg == nil {
        return
    }

    // 4. Verify ownership (only sender can edit)
    if origMsg.From != asUid.UserId() {
        return
    }

    // 5. Check time window (15 minutes)
    editWindow := 15 * time.Minute
    if time.Since(origMsg.CreatedAt) > editWindow {
        return
    }

    // 6. Check edit count (max 10)
    editCount := 0
    if origMsg.Head != nil {
        if count, ok := origMsg.Head["edit_count"].(float64); ok {
            editCount = int(count)
        }
    }
    if editCount >= 10 {
        return
    }

    // 7. Update message in database
    now := types.TimeNow()
    err = store.Messages.Edit(t.name, seqId, newContent, now, editCount+1)
    if err != nil {
        return
    }

    // 8. Broadcast to all subscribers
    info := &ServerComMessage{
        Info: &MsgServerInfo{
            Topic:    msg.Original,
            From:     msg.AsUser,
            What:     "edit",
            SeqId:    seqId,
            Content:  newContent,
            EditedAt: &now,
        },
        // ... routing fields
    }
    t.broadcastToSessions(info)
}
```

### handleUnsend (topic.go)

```go
func (t *Topic) handleUnsend(msg *ClientComMessage) {
    asUid := types.ParseUserId(msg.AsUser)
    seqId := msg.Note.SeqId

    // 1. Validate SeqId range
    if seqId > t.lastID || seqId <= 0 {
        return
    }

    // 2. Get original message
    origMsg, err := store.Messages.GetBySeqId(t.name, seqId)
    if err != nil || origMsg == nil {
        return
    }

    // 3. Verify ownership
    if origMsg.From != asUid.UserId() {
        return
    }

    // 4. Check time window (10 minutes)
    unsendWindow := 10 * time.Minute
    if time.Since(origMsg.CreatedAt) > unsendWindow {
        return
    }

    // 5. Mark as unsent in database
    now := types.TimeNow()
    err = store.Messages.MarkUnsent(t.name, seqId, now)
    if err != nil {
        return
    }

    // 6. Broadcast to all subscribers
    info := &ServerComMessage{
        Info: &MsgServerInfo{
            Topic: msg.Original,
            From:  msg.AsUser,
            What:  "unsend",
            SeqId: seqId,
        },
        // ... routing fields
    }
    t.broadcastToSessions(info)
}
```

---

## Database Changes

### Store Interface (store/store.go)

```go
type MessagesPersistenceInterface interface {
    // ... existing methods ...
    GetBySeqId(topic string, seqId int) (*types.Message, error)
    Edit(topic string, seqId int, content any, editedAt time.Time, editCount int) error
    MarkUnsent(topic string, seqId int, unsentAt time.Time) error
}
```

### PostgreSQL Implementation (db/postgres/adapter.go)

#### MessageGetBySeqId

Retrieves a single message by topic and sequence ID:

```go
func (a *adapter) MessageGetBySeqId(topic string, seqId int) (*t.Message, error) {
    // SELECT from messages WHERE topic=$1 AND seqid=$2 AND delid=0
}
```

#### MessageEdit

Updates message content and stores edit metadata in `head`:

```go
func (a *adapter) MessageEdit(topic string, seqId int, content any, editedAt time.Time, editCount int) error {
    // Transaction:
    // 1. Get current head
    // 2. Update head with edited_at and edit_count
    // 3. UPDATE messages SET content=$1, head=$2, updatedat=$3
}
```

**Head after edit:**
```json
{
  "edited_at": "2026-01-03T12:00:00Z",
  "edit_count": 1
}
```

#### MessageMarkUnsent

Marks message as unsent (tombstone pattern):

```go
func (a *adapter) MessageMarkUnsent(topic string, seqId int, unsentAt time.Time) error {
    // Transaction:
    // 1. Get current head
    // 2. Set head["unsent"] = true, head["unsent_at"] = timestamp
    // 3. UPDATE messages SET content=NULL, head=$1, updatedat=$2
}
```

**Head after unsend:**
```json
{
  "unsent": true,
  "unsent_at": "2026-01-03T12:00:00Z"
}
```

---

## Message Head Schema

After editing or unsending, the message's `head` field contains:

### Edited Message
```json
{
  "edited_at": "2026-01-03T12:00:00Z",
  "edit_count": 3
}
```

### Unsent Message
```json
{
  "unsent": true,
  "unsent_at": "2026-01-03T12:00:00Z"
}
```

---

## Error Handling

All validation failures result in silent drops (no error response to client). This follows the existing pattern for `{note}` messages which are fire-and-forget.

Errors are logged server-side:
- `topic[X]: edit failed - message not found`
- `topic[X]: edit denied - not message owner`
- `topic[X]: edit denied - outside edit window`
- `topic[X]: edit denied - max edit count reached`
- `topic[X]: failed to edit message: <error>`
- Similar for unsend operations

---

## Client Implementation Requirements

The client needs to:

1. **Send edit/unsend notes** via the existing `{note}` mechanism
2. **Listen for `{info}` messages** with `what: "edit"` or `what: "unsend"`
3. **Update local message cache** when receiving edit/unsend broadcasts
4. **Display edit indicator** for edited messages (e.g., "Edited" label)
5. **Display unsent placeholder** for unsent messages (e.g., "This message was unsent")
6. **Enforce UI constraints** (hide edit option after 15min, hide unsend after 10min)

---

## Testing

To test the implementation:

1. Send a message
2. Within 15 minutes, send an edit note
3. Verify the message content is updated in the database
4. Verify all subscribers receive the edit broadcast
5. Repeat edit 10 times, verify 11th edit is rejected
6. Wait 15 minutes, verify edit is rejected
7. For unsend: send message, unsend within 10 minutes, verify content is nulled

---

## Future Enhancements

- [ ] Add error responses for failed edit/unsend attempts
- [ ] Add rate limiting for edit requests
- [ ] Add edit history tracking (store previous versions)
- [ ] Add admin override for edit/unsend restrictions
- [ ] Protobuf support (currently JSON only)
