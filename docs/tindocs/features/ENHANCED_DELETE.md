# Enhanced Delete (Delete for Self vs Delete for Everyone)

## Overview

Extend Tinode's existing delete functionality to clearly distinguish between:
1. **Delete for me** - Remove message from my view only
2. **Delete for everyone** - Remove message from all participants

This is how WhatsApp, Telegram, and iMessage handle deletion.

## Current Tinode Behavior

Tinode already has soft delete (for self) and hard delete (for everyone):

```json
// Soft delete (for self only)
{
  "del": {
    "topic": "usrXXX",
    "what": "msg",
    "delseq": [{"low": 42}],
    "hard": false
  }
}

// Hard delete (for everyone) - requires 'D' permission
{
  "del": {
    "topic": "usrXXX",
    "what": "msg",
    "delseq": [{"low": 42}],
    "hard": true
  }
}
```

**Current limitations:**
1. Hard delete requires 'D' permission (usually only topic owner)
2. No clear UI distinction between delete types
3. No "delete for everyone" option for regular users on their own messages
4. No configurable policy per-topic

## Proposed Changes

### 1. New Delete Modes

```json
{
  "del": {
    "topic": "usrXXX",
    "what": "msg",
    "delseq": [{"low": 42}],
    "mode": "self" | "everyone" | "hard"
  }
}
```

| Mode | Effect | Permission Required |
|------|--------|---------------------|
| `self` | Delete from my view only | Reader (R) |
| `everyone` | Delete for all participants, leave tombstone | Writer (W) + own message only |
| `hard` | Permanently delete from database | Deleter (D) |

### 2. Permission Model

**Current:**
- `R` (Reader) - Can soft-delete (for self)
- `D` (Deleter) - Can hard-delete (for everyone)

**Proposed:**
- `R` (Reader) - Can delete for self
- `W` (Writer) - Can delete own messages for everyone (with tombstone)
- `D` (Deleter) - Can hard-delete any message (no tombstone)

### 3. Tombstone Messages

When a message is deleted "for everyone", instead of removing it completely, leave a tombstone:

```json
{
  "data": {
    "topic": "usrXXX",
    "from": "usrYYY",
    "seq": 42,
    "ts": "2026-01-02T03:45:00Z",
    "head": {
      "deleted": true,
      "deleted_at": "2026-01-02T04:00:00Z",
      "deleted_by": "usrYYY"
    },
    "content": null
  }
}
```

## Server-Side Changes

### 1. Modify Delete Handler (topic.go)

```go
func (t *Topic) replyDelMsg(sess *Session, asUid types.Uid, asChan bool, msg *ClientComMessage) error {
    now := types.TimeNow()
    del := msg.Del
    
    // Determine delete mode
    mode := del.Mode
    if mode == "" {
        // Legacy: use 'hard' field for backwards compatibility
        if del.Hard {
            mode = "hard"
        } else {
            mode = "self"
        }
    }
    
    pud := t.perUser[asUid]
    userMode := pud.modeGiven & pud.modeWant
    
    switch mode {
    case "self":
        // Delete for self - requires R permission
        if !userMode.IsReader() {
            sess.queueOut(ErrPermissionDeniedReply(msg, now))
            return errors.New("del.msg: permission denied")
        }
        return t.deleteForSelf(sess, asUid, del, msg, now)
        
    case "everyone":
        // Delete for everyone - requires W permission, own messages only
        if !userMode.IsWriter() {
            sess.queueOut(ErrPermissionDeniedReply(msg, now))
            return errors.New("del.msg: permission denied")
        }
        return t.deleteForEveryone(sess, asUid, del, msg, now)
        
    case "hard":
        // Hard delete - requires D permission
        if !userMode.IsDeleter() {
            sess.queueOut(ErrPermissionDeniedReply(msg, now))
            return errors.New("del.msg: permission denied")
        }
        return t.hardDelete(sess, asUid, del, msg, now)
    }
    
    return errors.New("del.msg: invalid mode")
}

func (t *Topic) deleteForEveryone(sess *Session, asUid types.Uid, del *MsgClientDel, msg *ClientComMessage, now time.Time) error {
    // Validate all messages are owned by sender
    for _, dq := range del.DelSeq {
        seqStart := dq.LowId
        seqEnd := dq.HiId
        if seqEnd == 0 {
            seqEnd = seqStart + 1
        }
        
        for seq := seqStart; seq < seqEnd; seq++ {
            origMsg, err := store.Messages.GetBySeq(t.name, seq)
            if err != nil {
                continue // Message doesn't exist
            }
            if origMsg.From != asUid.String() {
                sess.queueOut(ErrPermissionDeniedReply(msg, now))
                return errors.New("can only delete own messages for everyone")
            }
        }
    }
    
    // Mark messages as deleted (tombstone)
    ranges := normalizeRanges(del.DelSeq)
    err := store.Messages.MarkDeleted(t.name, ranges, asUid, now)
    if err != nil {
        sess.queueOut(ErrUnknownReply(msg, now))
        return err
    }
    
    // Broadcast deletion to all subscribers
    t.broadcastDelete(asUid, ranges, now, msg.sess.sid)
    
    sess.queueOut(NoErrReply(msg, now))
    return nil
}
```

### 2. Database Changes (adapter.go)

```go
// MarkDeleted marks messages as deleted for everyone (tombstone)
func (a *adapter) MessageMarkDeleted(topic string, ranges []types.Range, deletedBy types.Uid, deletedAt time.Time) error {
    for _, r := range ranges {
        _, err := a.db.Exec(`
            UPDATE messages 
            SET content = NULL,
                deletedfor = NULL,
                head = COALESCE(head, '{}'::jsonb) || jsonb_build_object(
                    'deleted', true,
                    'deleted_at', $1::text,
                    'deleted_by', $2
                )
            WHERE topic = $3 AND seqid >= $4 AND ($5 = 0 OR seqid < $5)
        `, deletedAt.Format(time.RFC3339), deletedBy.String(), topic, r.Low, r.Hi)
        if err != nil {
            return err
        }
    }
    return nil
}
```

### 3. Broadcast Delete Info

```go
func (t *Topic) broadcastDelete(deletedBy types.Uid, ranges []types.Range, deletedAt time.Time, skipSid string) {
    info := &ServerComMessage{
        Info: &MsgServerInfo{
            Topic:     t.original(deletedBy),
            From:      deletedBy.UserId(),
            What:      "del",
            DeletedAt: &deletedAt,
            DelSeq:    rangeDeserialize(ranges),
        },
        RcptTo:    t.name,
        Timestamp: deletedAt,
        SkipSid:   skipSid,
    }
    t.broadcastToSessions(info)
    
    // Also notify offline users via presence
    t.presSubsOffline("del", &presParams{
        delSeq: rangeDeserialize(ranges),
        actor:  deletedBy.UserId(),
    }, &presFilters{filterIn: types.ModeRead}, nilPresFilters, skipSid, true)
}
```

## Topic-Level Configuration

Allow topics to configure delete behavior:

```json
{
  "set": {
    "topic": "grpXXX",
    "desc": {
      "private": {
        "delete_policy": {
          "allow_delete_for_everyone": true,
          "delete_for_everyone_window": 86400,
          "sender_only_delete": true
        }
      }
    }
  }
}
```

| Setting | Description | Default |
|---------|-------------|---------|
| `allow_delete_for_everyone` | Enable "delete for everyone" | true |
| `delete_for_everyone_window` | Time window in seconds (0 = no limit) | 86400 (24h) |
| `sender_only_delete` | Only sender can delete for everyone | true |

## Client-Side Changes (React Native)

### 1. Delete Options UI

```tsx
function DeleteOptionsSheet({ message, onDelete, onClose }) {
  const isOwnMessage = message.from === currentUserId;
  const canDeleteForEveryone = isOwnMessage && isWithinDeleteWindow(message);
  
  return (
    <BottomSheet>
      <TouchableOpacity onPress={() => onDelete('self')}>
        <Text>Delete for me</Text>
        <Text style={styles.subtitle}>This message will be removed from your chat</Text>
      </TouchableOpacity>
      
      {canDeleteForEveryone && (
        <TouchableOpacity onPress={() => onDelete('everyone')}>
          <Text>Delete for everyone</Text>
          <Text style={styles.subtitle}>This message will be removed for all participants</Text>
        </TouchableOpacity>
      )}
      
      <TouchableOpacity onPress={onClose}>
        <Text>Cancel</Text>
      </TouchableOpacity>
    </BottomSheet>
  );
}
```

### 2. Display Deleted Messages

```tsx
function MessageBubble({ message }) {
  if (message.head?.deleted) {
    return (
      <View style={styles.deletedMessage}>
        <Text style={styles.deletedText}>
          🚫 This message was deleted
        </Text>
      </View>
    );
  }
  
  return (
    <View>
      <Text>{message.content}</Text>
    </View>
  );
}
```

### 3. Handle Delete Info

```typescript
function handleInfo(info: any) {
  if (info.what === 'del' && info.deleted_at) {
    // Message deleted for everyone
    for (const range of info.delseq || []) {
      const start = range.low;
      const end = range.hi || start + 1;
      for (let seq = start; seq < end; seq++) {
        markMessageDeleted(info.topic, seq, info.deleted_at, info.from);
      }
    }
    callbacks.onMessagesDeleted?.(info.topic, info.delseq);
  }
}
```

### 4. Tinode Service Methods

```typescript
async function deleteMessage(topic: string, seq: number, mode: 'self' | 'everyone'): Promise<void> {
  const msg = {
    del: {
      id: generateId(),
      topic,
      what: 'msg',
      delseq: [{ low: seq }],
      mode
    }
  };
  
  return sendAndWait(msg);
}
```

## Client Configuration Option

For Clingy specifically, we may want to disable "delete for everyone" to preserve evidence:

```typescript
// In app config
const CHAT_CONFIG = {
  allowDeleteForEveryone: false, // Disable for safety - preserve evidence
};

// In UI
{CHAT_CONFIG.allowDeleteForEveryone && canDeleteForEveryone && (
  <TouchableOpacity onPress={() => onDelete('everyone')}>
    <Text>Delete for everyone</Text>
  </TouchableOpacity>
)}
```

## Migration

1. No schema changes required - uses existing `head` JSON field
2. Existing soft/hard delete continues to work (backwards compatible)
3. New `mode` field is optional - falls back to `hard` boolean

## Edge Cases

| Case | Handling |
|------|----------|
| Delete for everyone after window | Reject with error |
| Delete someone else's message for everyone | Reject - sender only |
| Delete in group | All members see tombstone |
| Offline user syncs | Gets tombstone, not original content |
| Delete already deleted message | No-op, success |

## Testing

1. Delete for self - message hidden only for deleter
2. Delete for everyone (own message) - all see tombstone
3. Delete for everyone (other's message) - rejected
4. Delete for everyone after time window - rejected
5. Hard delete with D permission - message fully removed
6. Offline user syncs deleted message - sees tombstone

## Complexity Assessment

| Component | Effort |
|-----------|--------|
| Server - mode handling | Low |
| Server - tombstone logic | Low |
| Server - broadcast | Low |
| Database | None (uses head) |
| Client - delete options UI | Low |
| Client - tombstone display | Low |
| **Total** | **Low** |

## Files to Modify

### Server
- `server/datamodel.go` - Add `Mode` to `MsgClientDel`
- `server/topic.go` - Modify `replyDelMsg`, add `deleteForEveryone`
- `server/db/postgres/adapter.go` - Add `MessageMarkDeleted`

### Client
- `src/services/tinode.ts` - Add `deleteMessage` with mode
- `src/screens/ChatScreen.tsx` - Add delete options sheet
- `src/components/MessageBubble.tsx` - Show tombstone for deleted
- `src/components/DeleteOptionsSheet.tsx` - New component
