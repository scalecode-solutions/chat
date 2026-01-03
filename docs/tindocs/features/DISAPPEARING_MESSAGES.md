# Disappearing Messages

## Overview

Auto-delete messages after a configurable time period. This is a critical safety feature for Clingy and a standard feature in Signal, WhatsApp, and Telegram.

## User Experience

```
┌─────────────────────────────────────┐
│ 🕐 Disappearing messages: ON (24h)  │  ← Topic banner
├─────────────────────────────────────┤
│                                     │
│ Messages in this chat will          │
│ automatically delete after 24 hours │
│                                     │
│ ┌─────────────────────────────────┐ │
│ │ Hey, are you safe?              │ │
│ │                    2:30 PM  🕐  │ │  ← Timer icon
│ └─────────────────────────────────┘ │
│                                     │
│         ┌─────────────────────────┐ │
│         │ Yes, I'm okay           │ │
│         │ 🕐  2:31 PM             │ │
│         └─────────────────────────┘ │
└─────────────────────────────────────┘
```

## Timer Options

| Option | Duration | Use Case |
|--------|----------|----------|
| Off | Never | Normal chat |
| 1 hour | 3600s | High-risk immediate |
| 24 hours | 86400s | Daily cleanup |
| 7 days | 604800s | Weekly cleanup |
| 30 days | 2592000s | Monthly cleanup |

## Protocol Design

### 1. Set Disappearing Timer (Topic Setting)

```json
{
  "set": {
    "id": "123",
    "topic": "usrXXX",
    "desc": {
      "private": {
        "disappear": {
          "enabled": true,
          "ttl": 86400
        }
      }
    }
  }
}
```

**Fields:**
- `disappear.enabled` - Whether disappearing messages are on
- `disappear.ttl` - Time-to-live in seconds (0 = off)

### 2. Message with Expiration

When disappearing is enabled, messages get an expiration timestamp in `head`:

```json
{
  "data": {
    "topic": "usrXXX",
    "from": "usrYYY",
    "seq": 42,
    "ts": "2026-01-02T03:45:00Z",
    "head": {
      "expires": "2026-01-03T03:45:00Z"
    },
    "content": "Are you safe?"
  }
}
```

### 3. Server Broadcasts Timer Change

When a user changes the disappearing setting:

```json
{
  "info": {
    "topic": "usrXXX",
    "from": "usrYYY",
    "what": "disappear",
    "ttl": 86400
  }
}
```

## Server-Side Changes

### 1. Topic Metadata

Store disappearing settings in topic's `private` data (per-user) or `public` data (topic-wide).

**Option A: Per-user setting (like Signal)**
Each user can set their own timer. Messages they send use their timer.

**Option B: Topic-wide setting (like WhatsApp)**
One timer for the whole topic. Either party can change it.

**Recommendation:** Option B for simplicity.

### 2. Message Publishing (topic.go)

When publishing a message, check if topic has disappearing enabled:

```go
func (t *Topic) handlePubBroadcast(msg *ClientComMessage) {
    // ... existing validation ...
    
    // Check for disappearing messages setting
    if t.disappearTTL > 0 {
        expires := msg.Timestamp.Add(time.Duration(t.disappearTTL) * time.Second)
        if msg.Pub.Head == nil {
            msg.Pub.Head = make(map[string]any)
        }
        msg.Pub.Head["expires"] = expires.Format(time.RFC3339)
    }
    
    // ... continue with publish ...
}
```

### 3. Background Cleanup Job

New goroutine to periodically delete expired messages:

```go
// In main.go or separate file
func startMessageCleanupJob() {
    ticker := time.NewTicker(5 * time.Minute) // Run every 5 minutes
    go func() {
        for range ticker.C {
            if globals.shuttingDown {
                return
            }
            cleanupExpiredMessages()
        }
    }()
}

func cleanupExpiredMessages() {
    now := time.Now()
    
    // Delete messages where head->expires < now
    count, err := store.Messages.DeleteExpired(now)
    if err != nil {
        logs.Err.Printf("Failed to cleanup expired messages: %v", err)
        return
    }
    
    if count > 0 {
        logs.Info.Printf("Cleaned up %d expired messages", count)
    }
}
```

### 4. Database Functions (adapter.go)

```go
// DeleteExpired removes messages that have passed their expiration time
func (a *adapter) MessageDeleteExpired(before time.Time) (int64, error) {
    result, err := a.db.Exec(`
        DELETE FROM messages 
        WHERE head->>'expires' IS NOT NULL 
        AND (head->>'expires')::timestamp < $1
    `, before)
    if err != nil {
        return 0, err
    }
    return result.RowsAffected()
}

// Alternative: Batch delete with limit to avoid long locks
func (a *adapter) MessageDeleteExpiredBatch(before time.Time, limit int) (int64, error) {
    result, err := a.db.Exec(`
        DELETE FROM messages 
        WHERE id IN (
            SELECT id FROM messages 
            WHERE head->>'expires' IS NOT NULL 
            AND (head->>'expires')::timestamp < $1
            LIMIT $2
        )
    `, before, limit)
    if err != nil {
        return 0, err
    }
    return result.RowsAffected()
}
```

### 5. Topic Loading

Load disappearing setting when topic is loaded:

```go
type Topic struct {
    // ... existing fields ...
    disappearTTL int64 // TTL in seconds, 0 = disabled
}

func (t *Topic) loadDisappearSetting() {
    // Load from topic desc or first user's private setting
    if desc := t.getDesc(); desc != nil {
        if disappear, ok := desc.Private["disappear"].(map[string]any); ok {
            if ttl, ok := disappear["ttl"].(float64); ok {
                t.disappearTTL = int64(ttl)
            }
        }
    }
}
```

### 6. Handle Setting Change

```go
func (t *Topic) handleDisappearChange(sess *Session, asUid types.Uid, ttl int64) error {
    // Update topic setting
    t.disappearTTL = ttl
    
    // Persist to database
    err := store.Topics.UpdatePrivate(t.name, asUid, map[string]any{
        "disappear": map[string]any{
            "enabled": ttl > 0,
            "ttl":     ttl,
        },
    })
    if err != nil {
        return err
    }
    
    // Broadcast change to all subscribers
    info := &ServerComMessage{
        Info: &MsgServerInfo{
            Topic: t.original(asUid),
            From:  asUid.UserId(),
            What:  "disappear",
            TTL:   ttl,
        },
        RcptTo:    t.name,
        Timestamp: types.TimeNow(),
        SkipSid:   sess.sid,
    }
    t.broadcastToSessions(info)
    
    return nil
}
```

## Client-Side Changes (React Native)

### 1. Data Model

```typescript
interface TopicMeta {
  // ... existing ...
  disappear?: {
    enabled: boolean;
    ttl: number; // seconds
  };
}

interface ChatMessage {
  // ... existing ...
  expires?: string; // ISO timestamp
}
```

### 2. Topic Settings UI

```tsx
function DisappearingMessagesSettings({ topic, onUpdate }) {
  const options = [
    { label: 'Off', value: 0 },
    { label: '1 hour', value: 3600 },
    { label: '24 hours', value: 86400 },
    { label: '7 days', value: 604800 },
    { label: '30 days', value: 2592000 },
  ];
  
  const currentTTL = topic.disappear?.ttl || 0;
  
  return (
    <View>
      <Text style={styles.title}>Disappearing Messages</Text>
      <Text style={styles.subtitle}>
        Messages will be automatically deleted after the selected time
      </Text>
      
      {options.map(opt => (
        <TouchableOpacity 
          key={opt.value}
          onPress={() => onUpdate(opt.value)}
          style={[styles.option, currentTTL === opt.value && styles.selected]}
        >
          <Text>{opt.label}</Text>
          {currentTTL === opt.value && <Check size={20} />}
        </TouchableOpacity>
      ))}
    </View>
  );
}
```

### 3. Chat Header Banner

```tsx
function ChatHeader({ topic }) {
  const ttl = topic.disappear?.ttl;
  
  return (
    <View>
      <Text>{topic.name}</Text>
      {ttl > 0 && (
        <View style={styles.disappearBanner}>
          <Clock size={14} />
          <Text>Messages disappear after {formatDuration(ttl)}</Text>
        </View>
      )}
    </View>
  );
}
```

### 4. Message Timer Indicator

```tsx
function MessageBubble({ message }) {
  const hasExpiry = !!message.expires;
  
  return (
    <View>
      <Text>{message.content}</Text>
      <View style={styles.footer}>
        {hasExpiry && <Clock size={12} style={styles.timerIcon} />}
        <Text>{formatTime(message.ts)}</Text>
      </View>
    </View>
  );
}
```

### 5. Client-Side Cleanup

Even though server deletes messages, client should also clean up locally:

```typescript
// Run periodically or on app foreground
async function cleanupExpiredMessages() {
  const now = new Date().toISOString();
  await Database.deleteExpiredMessages(now);
}

// In database.ts
async function deleteExpiredMessages(before: string): Promise<void> {
  await db.executeSql(`
    DELETE FROM messages 
    WHERE expires IS NOT NULL AND expires < ?
  `, [before]);
}
```

### 6. Handle Disappear Info

```typescript
function handleInfo(info: any) {
  if (info.what === 'disappear') {
    // Update topic metadata
    updateTopicDisappear(info.topic, info.ttl);
    
    // Show system message
    addSystemMessage(info.topic, 
      info.ttl > 0 
        ? `Disappearing messages set to ${formatDuration(info.ttl)}`
        : 'Disappearing messages turned off'
    );
    
    callbacks.onDisappearChanged?.(info.topic, info.ttl);
  }
}
```

### 7. Set Disappearing Timer

```typescript
async function setDisappearingMessages(topic: string, ttl: number): Promise<void> {
  const msg = {
    set: {
      id: generateId(),
      topic,
      desc: {
        private: {
          disappear: {
            enabled: ttl > 0,
            ttl
          }
        }
      }
    }
  };
  
  return sendAndWait(msg);
}
```

## Configuration

### Server Config

```json
{
  "disappearing_messages": {
    "enabled": true,
    "cleanup_interval": 300,
    "cleanup_batch_size": 1000,
    "max_ttl": 2592000,
    "default_ttl": 0
  }
}
```

| Setting | Description | Default |
|---------|-------------|---------|
| `enabled` | Allow disappearing messages | true |
| `cleanup_interval` | Seconds between cleanup runs | 300 (5 min) |
| `cleanup_batch_size` | Max messages to delete per run | 1000 |
| `max_ttl` | Maximum allowed TTL | 2592000 (30 days) |
| `default_ttl` | Default TTL for new topics | 0 (off) |

### Per-Topic Override

Topics can have disappearing messages forced on or disabled:

```json
{
  "topic_config": {
    "disappear_forced": true,
    "disappear_ttl": 86400,
    "disappear_changeable": false
  }
}
```

## Database Index

Add index for efficient expired message lookup:

```sql
CREATE INDEX idx_messages_expires ON messages ((head->>'expires'))
WHERE head->>'expires' IS NOT NULL;
```

## Edge Cases

| Case | Handling |
|------|----------|
| Message sent while offline | Expiry calculated from send time, not delivery |
| User changes TTL | Only affects new messages, not existing |
| Message with reply to expired | Show "Original message expired" |
| Attachment with expired message | Delete attachment file too |
| Encrypted messages | Expiry in head (unencrypted), content encrypted |

## Security Considerations

1. **Server-side enforcement** - Don't rely on client to delete
2. **No recovery** - Once deleted, messages are gone
3. **Attachments** - Must also delete associated files
4. **Audit trail** - Consider logging deletions (without content) for compliance
5. **Backup exclusion** - Expired messages shouldn't be in backups

## Testing

1. Enable disappearing (1 hour) - messages get `expires` in head
2. Wait for expiry - server deletes messages
3. Client syncs - doesn't receive expired messages
4. Change TTL - only new messages affected
5. Disable disappearing - new messages don't expire
6. Reply to expired message - shows placeholder
7. Attachment expires - file deleted from storage

## Complexity Assessment

| Component | Effort |
|-----------|--------|
| Server - message publishing | Low |
| Server - cleanup job | Medium |
| Server - setting handler | Low |
| Database - delete function | Low |
| Database - index | Low |
| Client - settings UI | Medium |
| Client - indicators | Low |
| Client - local cleanup | Low |
| **Total** | **Medium** |

## Files to Modify

### Server
- `server/main.go` - Start cleanup goroutine
- `server/topic.go` - Add expiry to messages, handle setting
- `server/datamodel.go` - Add TTL to `MsgServerInfo`
- `server/store/store.go` - Add `DeleteExpired` interface
- `server/db/postgres/adapter.go` - Implement `DeleteExpired`
- `server/tinode.conf` - Add config options

### Database
- Migration to add index on `head->>'expires'`

### Client
- `src/services/tinode.ts` - Add `setDisappearingMessages`, handle info
- `src/services/database.ts` - Add `deleteExpiredMessages`
- `src/screens/ChatScreen.tsx` - Show banner, timer icons
- `src/screens/ChatSettingsScreen.tsx` - Disappearing settings UI
- `src/components/MessageBubble.tsx` - Timer indicator
