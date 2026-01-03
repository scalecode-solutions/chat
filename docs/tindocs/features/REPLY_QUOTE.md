# Reply/Quote Messages

## Overview

Allow users to reply to a specific message, showing a preview of the original message above the reply. This is a standard feature in WhatsApp, Telegram, iMessage, Slack, etc.

## User Experience

```
┌─────────────────────────────────────┐
│ ┌─────────────────────────────────┐ │
│ │ Replying to: "Are you okay?"    │ │  ← Quote preview
│ └─────────────────────────────────┘ │
│ Yes, I'm fine. Just busy today.     │  ← Reply message
│                              2:30 PM │
└─────────────────────────────────────┘
```

## Protocol Design

### Option A: Use Existing `head` Field (Recommended)

Store the reply reference in the message's `head` field, which already exists and is returned to clients.

**Publish message with reply:**
```json
{
  "pub": {
    "id": "123",
    "topic": "usrXXX",
    "head": {
      "reply": {
        "seq": 42,
        "preview": "Are you okay?"
      }
    },
    "content": "Yes, I'm fine. Just busy today."
  }
}
```

**Fields:**
- `head.reply.seq` - SeqId of the message being replied to (required)
- `head.reply.preview` - Truncated preview text of original message (optional, max 100 chars)

### Option B: New `reply_to` Field

Add a dedicated field to `MsgClientPub`:

```go
type MsgClientPub struct {
    // ... existing fields ...
    ReplyTo int `json:"reply_to,omitempty"` // SeqId of message being replied to
}
```

**Recommendation:** Option A is simpler - no schema changes, uses existing `head` mechanism.

## Server-Side Changes

### 1. Validation (session.go or topic.go)

When processing a `{pub}` message with `head.reply`:

```go
// In publish handler
if head, ok := msg.Pub.Head["reply"].(map[string]any); ok {
    if replySeq, ok := head["seq"].(float64); ok {
        seq := int(replySeq)
        // Validate: seq must be > 0 and <= topic.lastID
        if seq <= 0 || seq > t.lastID {
            // Invalid reply reference - strip it or reject
            delete(msg.Pub.Head, "reply")
        }
        // Optionally: verify the message exists and user can read it
    }
}
```

### 2. Storage

No database changes needed - `head` is already stored as JSON in the `messages` table.

### 3. Retrieval

No changes needed - `head` is already returned when fetching messages.

## Client-Side Changes (React Native)

### 1. Data Model

```typescript
interface ChatMessage {
  // ... existing fields ...
  reply?: {
    seq: number;
    preview?: string;
  };
}
```

### 2. UI Components

**ReplyPreview component:**
```tsx
function ReplyPreview({ replySeq, preview, onPress }: Props) {
  return (
    <TouchableOpacity onPress={() => scrollToMessage(replySeq)}>
      <View style={styles.replyContainer}>
        <View style={styles.replyBar} />
        <Text style={styles.replyText} numberOfLines={1}>
          {preview || `Message #${replySeq}`}
        </Text>
      </View>
    </TouchableOpacity>
  );
}
```

**In message bubble:**
```tsx
function MessageBubble({ message }: Props) {
  return (
    <View>
      {message.reply && (
        <ReplyPreview 
          replySeq={message.reply.seq} 
          preview={message.reply.preview}
          onPress={() => scrollToMessage(message.reply.seq)}
        />
      )}
      <Text>{message.content}</Text>
    </View>
  );
}
```

### 3. Reply Action

**Long-press menu:**
- Add "Reply" option to message long-press menu
- When selected, show reply composer with preview of original message
- Store `replySeq` in composer state
- Include in `head` when sending

### 4. Scroll to Original

When user taps the reply preview:
- Find message with matching `seq` in local list
- Scroll to that message
- Briefly highlight it

## Database Schema

**No changes required.** The `head` column already stores arbitrary JSON.

## Migration

None required - this is purely additive.

## Edge Cases

| Case | Handling |
|------|----------|
| Reply to deleted message | Show "Original message deleted" in preview |
| Reply to message user can't see | Strip reply reference on server |
| Very long preview text | Truncate to 100 chars client-side before sending |
| Reply to reply | Allowed - just reference the direct parent |
| Cross-topic reply | Not allowed - validate topic matches |

## Testing

1. Send message with reply reference
2. Verify `head.reply` is stored in database
3. Verify `head.reply` is returned when fetching messages
4. Verify client displays reply preview
5. Verify tap-to-scroll works
6. Verify reply to deleted message shows placeholder
7. Verify invalid seq references are stripped

## Complexity Assessment

| Component | Effort |
|-----------|--------|
| Server validation | Low (few lines) |
| Database | None |
| Client UI | Medium (new components) |
| **Total** | **Low-Medium** |

## Files to Modify

### Server
- `server/topic.go` - Add validation in publish handler
- `server/datamodel.go` - No changes (head already exists)

### Client (React Native)
- `src/components/ReplyPreview.tsx` - New component
- `src/components/MessageBubble.tsx` - Add reply preview
- `src/screens/ChatScreen.tsx` - Add reply state, scroll logic
- `src/services/tinode.ts` - Include reply in publish
