# Edit/Unsend Messages

## Overview

Allow users to edit sent messages within a time window, or completely unsend (retract) them. Similar to WhatsApp, iMessage, Telegram.

## User Experience

### Edit
```
Original:                          After Edit:
┌─────────────────────────┐       ┌─────────────────────────┐
│ I'll be there at 5pm    │  →    │ I'll be there at 6pm    │
│                  2:30 PM│       │           (edited) 2:30 PM│
└─────────────────────────┘       └─────────────────────────┘
```

### Unsend
```
Before:                            After:
┌─────────────────────────┐       ┌─────────────────────────┐
│ Oops wrong chat!        │  →    │ 🚫 Message unsent       │
│                  2:30 PM│       │                  2:30 PM│
└─────────────────────────┘       └─────────────────────────┘
```

## Protocol Design

### Edit Message

**New message type: `{edit}`**

```json
{
  "edit": {
    "id": "123",
    "topic": "usrXXX",
    "seq": 42,
    "content": "Updated message content"
  }
}
```

**Server response:**
```json
{
  "ctrl": {
    "id": "123",
    "code": 200,
    "text": "ok",
    "params": {
      "seq": 42,
      "edited_at": "2026-01-02T03:45:00Z"
    }
  }
}
```

**Broadcast to subscribers:**
```json
{
  "info": {
    "topic": "usrXXX",
    "from": "usrYYY",
    "what": "edit",
    "seq": 42,
    "content": "Updated message content",
    "ts": "2026-01-02T03:45:00Z"
  }
}
```

### Unsend Message

**Use existing `{del}` with new flag:**

```json
{
  "del": {
    "id": "123",
    "topic": "usrXXX",
    "what": "msg",
    "delseq": [{"low": 42}],
    "hard": true,
    "unsend": true
  }
}
```

The `unsend: true` flag indicates this should:
1. Hard-delete the message content
2. Leave a tombstone showing "Message unsent"
3. Broadcast to all subscribers to update their UI

**Alternative:** Use `{note what="unsend"}` similar to reactions.

## Server-Side Changes

### 1. New Data Structures (datamodel.go)

```go
// MsgClientEdit is a request to edit a previously sent message
type MsgClientEdit struct {
    Id      string `json:"id,omitempty"`
    Topic   string `json:"topic"`
    SeqId   int    `json:"seq"`
    Content any    `json:"content"`
}

// Add to ClientComMessage
type ClientComMessage struct {
    // ... existing fields ...
    Edit *MsgClientEdit `json:"edit"`
}

// MsgServerInfo - add edit fields
type MsgServerInfo struct {
    // ... existing fields ...
    Content   any        `json:"content,omitempty"`   // For edit broadcasts
    EditedAt  *time.Time `json:"edited_at,omitempty"` // When message was edited
}
```

### 2. Message Storage Changes

**Option A: Store edit history (recommended for audit)**

Add to `messages` table or new `message_edits` table:
```sql
CREATE TABLE message_edits (
    id SERIAL PRIMARY KEY,
    topic VARCHAR(255) NOT NULL,
    seq_id INT NOT NULL,
    edited_at TIMESTAMP NOT NULL,
    previous_content JSONB,
    edited_by BIGINT NOT NULL,
    FOREIGN KEY (topic, seq_id) REFERENCES messages(topic, seqid)
);
```

**Option B: Simple overwrite**

Just update the `content` column and add `edited_at` to `head`:
```sql
UPDATE messages 
SET content = $1, 
    head = head || '{"edited_at": "2026-01-02T03:45:00Z"}'
WHERE topic = $2 AND seqid = $3;
```

### 3. Edit Handler (topic.go)

```go
func (t *Topic) handleEdit(msg *ClientComMessage) {
    now := types.TimeNow()
    asUid := types.ParseUserId(msg.AsUser)
    edit := msg.Edit
    
    // 1. Validate permissions
    pud := t.perUser[asUid]
    if !((pud.modeGiven & pud.modeWant).IsWriter()) {
        msg.sess.queueOut(ErrPermissionDeniedReply(msg, now))
        return
    }
    
    // 2. Validate seq exists and user is the sender
    origMsg, err := store.Messages.GetBySeq(t.name, edit.SeqId)
    if err != nil || origMsg == nil {
        msg.sess.queueOut(ErrNotFoundReply(msg, now))
        return
    }
    
    if origMsg.From != asUid.String() {
        // Can only edit your own messages
        msg.sess.queueOut(ErrPermissionDeniedReply(msg, now))
        return
    }
    
    // 3. Check time window (15 minutes)
    editWindow := 15 * time.Minute
    if time.Since(origMsg.CreatedAt) > editWindow {
        msg.sess.queueOut(ErrOperationNotAllowedReply(msg, now))
        return
    }
    
    // 4. Check edit count limit (max 10 edits)
    editCount := getEditCount(origMsg.Head) // Extract from head.edit_count
    if editCount >= 10 {
        msg.sess.queueOut(ErrOperationNotAllowedReply(msg, now))
        return
    }
    
    // 5. Update message in database (increment edit_count)
    err = store.Messages.Edit(t.name, edit.SeqId, edit.Content, now, editCount+1)
    if err != nil {
        msg.sess.queueOut(ErrUnknownReply(msg, now))
        return
    }
    
    // 6. Send success response
    msg.sess.queueOut(NoErrParamsReply(msg, now, map[string]any{
        "seq": edit.SeqId,
        "edited_at": now,
    }))
    
    // 7. Broadcast edit to all subscribers
    info := &ServerComMessage{
        Info: &MsgServerInfo{
            Topic:    msg.Original,
            From:     msg.AsUser,
            What:     "edit",
            SeqId:    edit.SeqId,
            Content:  edit.Content,
            EditedAt: &now,
        },
        RcptTo:    msg.RcptTo,
        Timestamp: now,
        SkipSid:   msg.sess.sid,
    }
    t.broadcastToSessions(info)
}
```

### 4. Unsend Handler

Modify existing `replyDelMsg` to handle unsend:

```go
func (t *Topic) replyDelMsg(sess *Session, asUid types.Uid, asChan bool, msg *ClientComMessage) error {
    del := msg.Del
    
    // Check if this is an unsend operation
    if del.Unsend {
        // Unsend: hard delete but leave tombstone
        // Only sender can unsend
        for _, dq := range del.DelSeq {
            origMsg, _ := store.Messages.GetBySeq(t.name, dq.LowId)
            if origMsg != nil && origMsg.From != asUid.String() {
                sess.queueOut(ErrPermissionDeniedReply(msg, now))
                return errors.New("can only unsend own messages")
            }
        }
        
        // Mark as unsent (tombstone) instead of full delete
        err := store.Messages.MarkUnsent(t.name, del.DelSeq)
        if err != nil {
            sess.queueOut(ErrUnknownReply(msg, now))
            return err
        }
        
        // Broadcast unsend to all subscribers
        t.broadcastUnsend(asUid, del.DelSeq, msg.sess.sid)
        
        sess.queueOut(NoErrReply(msg, now))
        return nil
    }
    
    // ... existing delete logic ...
}
```

### 5. Database Functions (adapter.go)

```go
// Edit updates a message's content
func (a *adapter) MessageEdit(topic string, seqId int, content any, editedAt time.Time) error {
    // Store previous version in edit history (optional)
    _, err := a.db.Exec(`
        INSERT INTO message_edits (topic, seq_id, edited_at, previous_content, edited_by)
        SELECT topic, seqid, $3, content, sender FROM messages 
        WHERE topic = $1 AND seqid = $2
    `, topic, seqId, editedAt)
    if err != nil {
        return err
    }
    
    // Update message content and mark as edited
    contentJSON, _ := json.Marshal(content)
    _, err = a.db.Exec(`
        UPDATE messages 
        SET content = $1,
            head = COALESCE(head, '{}'::jsonb) || jsonb_build_object('edited_at', $2::text)
        WHERE topic = $3 AND seqid = $4
    `, contentJSON, editedAt.Format(time.RFC3339), topic, seqId)
    
    return err
}

// MarkUnsent marks messages as unsent (tombstone)
func (a *adapter) MessageMarkUnsent(topic string, ranges []types.Range) error {
    for _, r := range ranges {
        _, err := a.db.Exec(`
            UPDATE messages 
            SET content = NULL,
                head = COALESCE(head, '{}'::jsonb) || '{"unsent": true}'::jsonb
            WHERE topic = $1 AND seqid >= $2 AND ($3 = 0 OR seqid < $3)
        `, topic, r.Low, r.Hi)
        if err != nil {
            return err
        }
    }
    return nil
}
```

## Client-Side Changes (React Native)

### 1. Data Model

```typescript
interface ChatMessage {
  // ... existing fields ...
  editedAt?: string;
  unsent?: boolean;
}
```

### 2. Edit UI

**Long-press menu:**
```tsx
const messageActions = [
  { label: 'Reply', action: 'reply' },
  { label: 'Edit', action: 'edit', condition: isOwnMessage && isWithinEditWindow },
  { label: 'Unsend', action: 'unsend', condition: isOwnMessage },
  { label: 'Delete for me', action: 'delete' },
];
```

**Edit mode:**
```tsx
function ChatInput({ editingMessage, onCancelEdit, onSendEdit }) {
  if (editingMessage) {
    return (
      <View>
        <View style={styles.editBanner}>
          <Text>Editing message</Text>
          <TouchableOpacity onPress={onCancelEdit}>
            <X size={20} />
          </TouchableOpacity>
        </View>
        <TextInput 
          defaultValue={editingMessage.content}
          onSubmitEditing={(text) => onSendEdit(editingMessage.seq, text)}
        />
      </View>
    );
  }
  // ... normal input ...
}
```

### 3. Display Edited Messages

```tsx
function MessageBubble({ message }) {
  return (
    <View>
      {message.unsent ? (
        <Text style={styles.unsentText}>🚫 Message unsent</Text>
      ) : (
        <>
          <Text>{message.content}</Text>
          {message.editedAt && (
            <Text style={styles.editedLabel}>(edited)</Text>
          )}
        </>
      )}
    </View>
  );
}
```

### 4. Handle Edit/Unsend Info

```typescript
// In tinode.ts handleInfo
function handleInfo(info: any) {
  if (info.what === 'edit') {
    // Update local message
    updateMessageContent(info.topic, info.seq, info.content, info.edited_at);
    // Notify UI
    callbacks.onMessageEdited?.(info.topic, info.seq, info.content);
  } else if (info.what === 'unsend') {
    // Mark message as unsent
    markMessageUnsent(info.topic, info.seq);
    callbacks.onMessageUnsent?.(info.topic, info.seq);
  }
}
```

## Configuration Options

```go
// In config
type configType struct {
    // ... existing ...
    
    // Maximum age of a message that can be edited (seconds). 0 = no limit.
    MsgEditAge int `json:"msg_edit_age"`
    
    // Maximum number of edits allowed per message within the edit window
    MsgEditMaxCount int `json:"msg_edit_max_count"`
    
    // Maximum age of a message that can be unsent (seconds). 0 = no limit.
    MsgUnsendAge int `json:"msg_unsend_age"`
    
    // Whether to keep edit history
    KeepEditHistory bool `json:"keep_edit_history"`
}
```

**Default values:**
- `msg_edit_age`: 900 (15 minutes)
- `msg_edit_max_count`: 10 (max 10 edits within the 15 minute window)
- `msg_unsend_age`: 600 (10 minutes)
- `keep_edit_history`: true

## Edge Cases

| Case | Handling |
|------|----------|
| Edit after time window (15 min) | Reject with error |
| Edit count exceeds limit (10) | Reject with error |
| Edit someone else's message | Reject with permission error |
| Edit deleted message | Reject - message not found |
| Unsend after time window (10 min) | Reject with error |
| Unsend in group | All members see "Message unsent" |
| Offline user receives edit | Gets updated content on sync |
| Edit with empty content | Reject - use unsend instead |

## Security Considerations

1. **Only sender can edit/unsend** - Server validates `msg.From == asUid`
2. **Time limits** - Configurable windows prevent abuse
3. **Audit trail** - Edit history preserved for legal compliance
4. **No content in unsend broadcast** - Don't leak deleted content

## Testing

1. Edit message within time window - success
2. Edit message after time window - rejected
3. Edit another user's message - rejected
4. Unsend own message - success, shows tombstone
5. Unsend another's message - rejected
6. Offline user syncs edited message - sees updated content
7. Edit history preserved in database

## Complexity Assessment

| Component | Effort |
|-----------|--------|
| Server - new message type | Medium |
| Server - edit handler | Medium |
| Server - unsend handler | Low (modify existing) |
| Database - edit history table | Low |
| Database - adapter functions | Medium |
| Client - edit UI | Medium |
| Client - unsent display | Low |
| **Total** | **Medium** |

## Files to Modify

### Server
- `server/datamodel.go` - Add `MsgClientEdit`, extend `MsgServerInfo`
- `server/session.go` - Route `{edit}` messages
- `server/topic.go` - Add `handleEdit`, modify `replyDelMsg`
- `server/store/store.go` - Add `Edit`, `MarkUnsent` interfaces
- `server/db/postgres/adapter.go` - Implement edit/unsend functions
- `server/pbconverter.go` - Add protobuf conversion for edit
- `pbx/model.proto` - Add edit message type

### Database
- Migration to add `message_edits` table
- Migration to add `edited_at` index on messages

### Client
- `src/services/tinode.ts` - Add edit/unsend methods, handle info
- `src/screens/ChatScreen.tsx` - Add edit mode state
- `src/components/MessageBubble.tsx` - Show edited/unsent state
- `src/components/ChatInput.tsx` - Edit mode UI
