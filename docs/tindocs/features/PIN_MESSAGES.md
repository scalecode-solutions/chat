# Pin Messages

## Overview

Allow users to pin important messages to the top of a conversation for easy access. Common in Telegram, Slack, Discord, and WhatsApp (recent).

## User Experience

```
┌─────────────────────────────────────┐
│ 📌 Pinned Messages (2)         ▼   │  ← Tap to expand
├─────────────────────────────────────┤
│ ┌─────────────────────────────────┐ │
│ │ 📌 Emergency contact: 911      │ │
│ │ Call if you feel unsafe        │ │
│ └─────────────────────────────────┘ │
│ ┌─────────────────────────────────┐ │
│ │ 📌 Safe word: "pineapple"      │ │
│ └─────────────────────────────────┘ │
├─────────────────────────────────────┤
│         Regular messages...         │
└─────────────────────────────────────┘
```

## Protocol Design

### 1. Pin a Message

**New note type: `{note what="pin"}`**

```json
{
  "note": {
    "topic": "usrXXX",
    "what": "pin",
    "seq": 42
  }
}
```

### 2. Unpin a Message

```json
{
  "note": {
    "topic": "usrXXX",
    "what": "unpin",
    "seq": 42
  }
}
```

### 3. Server Response (Broadcast)

```json
{
  "info": {
    "topic": "usrXXX",
    "from": "usrYYY",
    "what": "pin",
    "seq": 42,
    "pinned": true
  }
}
```

### 4. Get Pinned Messages

Pinned message IDs stored in topic metadata, returned with topic subscription:

```json
{
  "meta": {
    "topic": "usrXXX",
    "desc": {
      "public": {
        "pinned": [42, 38, 15]
      }
    }
  }
}
```

## Server-Side Changes

### 1. Data Structures (datamodel.go)

Extend `MsgClientNote` to handle pin/unpin (already has `what` and `seq`):

```go
// No changes needed - existing fields work:
// What: "pin" or "unpin"
// SeqId: message to pin/unpin
```

Extend `MsgServerInfo`:

```go
type MsgServerInfo struct {
    // ... existing fields ...
    Pinned *bool `json:"pinned,omitempty"` // For pin/unpin broadcasts
}
```

### 2. Handle Pin Note (session.go)

Add to the note handler switch:

```go
func (s *Session) note(msg *ClientComMessage) {
    // ... existing validation ...
    
    switch msg.Note.What {
    // ... existing cases ...
    
    case "pin", "unpin":
        if msg.Note.SeqId <= 0 {
            return
        }
        // Route to topic
    }
    
    // ... rest of handler ...
}
```

### 3. Handle Pin in Topic (topic.go)

```go
func (t *Topic) handleNoteBroadcast(msg *ClientComMessage) {
    // ... existing code ...
    
    switch msg.Note.What {
    // ... existing cases ...
    
    case "pin":
        if !mode.IsWriter() {
            return
        }
        t.handlePin(msg, true)
        return
        
    case "unpin":
        if !mode.IsWriter() {
            return
        }
        t.handlePin(msg, false)
        return
    }
}

func (t *Topic) handlePin(msg *ClientComMessage, pin bool) {
    asUid := types.ParseUserId(msg.AsUser)
    seqId := msg.Note.SeqId
    
    // Validate message exists
    if seqId > t.lastID || seqId <= 0 {
        return
    }
    
    // Get current pinned list
    pinned := t.getPinnedMessages()
    
    if pin {
        // Check max pinned limit (e.g., 10)
        if len(pinned) >= 10 {
            // Could send error or just ignore
            return
        }
        
        // Check if already pinned
        for _, p := range pinned {
            if p == seqId {
                return // Already pinned
            }
        }
        
        // Add to pinned list
        pinned = append(pinned, seqId)
    } else {
        // Remove from pinned list
        newPinned := make([]int, 0, len(pinned))
        for _, p := range pinned {
            if p != seqId {
                newPinned = append(newPinned, p)
            }
        }
        pinned = newPinned
    }
    
    // Update topic metadata
    err := store.Topics.UpdatePublic(t.name, map[string]any{
        "pinned": pinned,
    })
    if err != nil {
        logs.Warn.Printf("topic[%s]: failed to update pinned: %v", t.name, err)
        return
    }
    
    // Update local cache
    t.setPinnedMessages(pinned)
    
    // Broadcast to all subscribers
    info := &ServerComMessage{
        Info: &MsgServerInfo{
            Topic:  msg.Original,
            From:   msg.AsUser,
            What:   msg.Note.What,
            SeqId:  seqId,
            Pinned: &pin,
        },
        RcptTo:    msg.RcptTo,
        Timestamp: msg.Timestamp,
        SkipSid:   msg.sess.sid,
    }
    t.broadcastToSessions(info)
}

func (t *Topic) getPinnedMessages() []int {
    if t.public == nil {
        return nil
    }
    if pinned, ok := t.public["pinned"].([]any); ok {
        result := make([]int, 0, len(pinned))
        for _, p := range pinned {
            if seq, ok := p.(float64); ok {
                result = append(result, int(seq))
            }
        }
        return result
    }
    return nil
}

func (t *Topic) setPinnedMessages(pinned []int) {
    if t.public == nil {
        t.public = make(map[string]any)
    }
    t.public["pinned"] = pinned
}
```

### 4. Include Pinned in Topic Desc

When returning topic description, pinned messages are already in `public`:

```go
// In replyGetDesc - no changes needed
// public already includes pinned array
```

## Client-Side Changes (React Native)

### 1. Data Model

```typescript
interface TopicMeta {
  // ... existing ...
  pinned?: number[]; // Array of pinned message seqIds
}

interface ChatMessage {
  // ... existing ...
  isPinned?: boolean; // Computed from topic.pinned
}
```

### 2. Pinned Messages Header

```tsx
function PinnedMessagesHeader({ pinnedSeqs, messages, onTapPinned }) {
  const [expanded, setExpanded] = useState(false);
  
  if (!pinnedSeqs?.length) return null;
  
  const pinnedMessages = pinnedSeqs
    .map(seq => messages.find(m => m.seq === seq))
    .filter(Boolean);
  
  return (
    <View style={styles.pinnedContainer}>
      <TouchableOpacity 
        style={styles.pinnedHeader}
        onPress={() => setExpanded(!expanded)}
      >
        <Pin size={16} />
        <Text>Pinned Messages ({pinnedMessages.length})</Text>
        <ChevronDown size={16} style={expanded && styles.rotated} />
      </TouchableOpacity>
      
      {expanded && (
        <View style={styles.pinnedList}>
          {pinnedMessages.map(msg => (
            <TouchableOpacity 
              key={msg.seq}
              style={styles.pinnedItem}
              onPress={() => onTapPinned(msg.seq)}
            >
              <Text numberOfLines={2}>{msg.content}</Text>
            </TouchableOpacity>
          ))}
        </View>
      )}
    </View>
  );
}
```

### 3. Pin Action in Message Menu

```tsx
const messageActions = [
  { label: 'Reply', action: 'reply' },
  { label: 'React', action: 'react' },
  { 
    label: isPinned ? 'Unpin' : 'Pin', 
    action: isPinned ? 'unpin' : 'pin',
    icon: Pin
  },
  { label: 'Delete', action: 'delete' },
];
```

### 4. Pin Indicator on Message

```tsx
function MessageBubble({ message, isPinned }) {
  return (
    <View style={[styles.bubble, isPinned && styles.pinnedBubble]}>
      {isPinned && (
        <View style={styles.pinIndicator}>
          <Pin size={12} />
        </View>
      )}
      <Text>{message.content}</Text>
    </View>
  );
}
```

### 5. Tinode Service Methods

```typescript
async function pinMessage(topic: string, seq: number): Promise<void> {
  const msg = {
    note: {
      topic,
      what: 'pin',
      seq
    }
  };
  send(msg); // Note messages don't get responses
}

async function unpinMessage(topic: string, seq: number): Promise<void> {
  const msg = {
    note: {
      topic,
      what: 'unpin',
      seq
    }
  };
  send(msg);
}
```

### 6. Handle Pin Info

```typescript
function handleInfo(info: any) {
  if (info.what === 'pin' || info.what === 'unpin') {
    const isPinned = info.pinned === true;
    updateTopicPinned(info.topic, info.seq, isPinned);
    callbacks.onMessagePinned?.(info.topic, info.seq, isPinned);
  }
}

function updateTopicPinned(topic: string, seq: number, pinned: boolean) {
  const topicMeta = getTopicMeta(topic);
  let pinnedList = topicMeta.pinned || [];
  
  if (pinned) {
    if (!pinnedList.includes(seq)) {
      pinnedList = [...pinnedList, seq];
    }
  } else {
    pinnedList = pinnedList.filter(s => s !== seq);
  }
  
  setTopicMeta(topic, { ...topicMeta, pinned: pinnedList });
}
```

## Configuration

### Server Config

```json
{
  "pinned_messages": {
    "enabled": true,
    "max_pinned_per_topic": 10
  }
}
```

### Permission Model

| Action | Permission Required |
|--------|---------------------|
| Pin message | Writer (W) |
| Unpin message | Writer (W) or original pinner |
| View pinned | Reader (R) |

## Storage

Pinned messages stored in topic's `public` metadata:

```json
{
  "public": {
    "fn": "Topic Name",
    "pinned": [42, 38, 15]
  }
}
```

**Why `public` not `private`?**
- Pinned messages should be visible to all topic members
- `private` is per-user, `public` is shared

## Edge Cases

| Case | Handling |
|------|----------|
| Pin deleted message | Allow - shows "Message deleted" in pinned list |
| Pin message in channel | Only admins can pin |
| Unpin by non-pinner | Allow if user has W permission |
| Max pins reached | Reject silently or show error |
| Pin same message twice | No-op |
| Pinned message expires (disappearing) | Remove from pinned list on cleanup |

## Testing

1. Pin message - appears in pinned list
2. Unpin message - removed from pinned list
3. Pin limit - can't pin more than max
4. Pin deleted message - shows placeholder
5. Other user sees pin - broadcast works
6. Subscribe to topic - gets pinned list in meta
7. Tap pinned - scrolls to message

## Complexity Assessment

| Component | Effort |
|-----------|--------|
| Server - note handler | Low |
| Server - pin logic | Low |
| Server - topic metadata | Low |
| Database | None (uses existing public) |
| Client - pinned header | Medium |
| Client - pin action | Low |
| Client - indicators | Low |
| **Total** | **Low** |

## Files to Modify

### Server
- `server/session.go` - Add pin/unpin to note handler
- `server/topic.go` - Add `handlePin` function
- `server/datamodel.go` - Add `Pinned` to `MsgServerInfo`

### Client
- `src/services/tinode.ts` - Add `pinMessage`, `unpinMessage`, handle info
- `src/screens/ChatScreen.tsx` - Add pinned header, pin actions
- `src/components/PinnedMessagesHeader.tsx` - New component
- `src/components/MessageBubble.tsx` - Pin indicator
- `src/components/MessageActions.tsx` - Pin/unpin option
