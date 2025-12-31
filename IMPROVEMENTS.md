# Tinode Improvements Brainstorm

This document tracks potential improvements, bug fixes, and feature additions for our custom Tinode fork.

---

## Already Implemented ✅

### 1. PostgreSQL Database Creation Fix (Our Fix)
**Issue**: Official `tinode/tinode-postgres` image fails when database already exists but has no schema.

**Fix**: Modified `server/db/postgres/adapter.go` `CreateDb()` to catch PostgreSQL error code `42P04` (duplicate_database) and continue instead of failing.

**Status**: ✅ Deployed and working

---

## Features Already in Tinode (No Work Needed)

After reviewing the codebase, these features from GitHub issues are **already implemented**:

### 1. Reply/Quote Messages ✅
**GitHub Issue**: Was requested but already exists!

**How it works**: Use the `head` field in `{pub}` messages:
```json
{
  "pub": {
    "topic": "usrXXX",
    "head": {
      "reply": "grp1XUtEhjv6HND:123"
    },
    "content": "This is my reply"
  }
}
```
- `reply`: topic-unique ID of the message being replied to (format: `":123"` or `"topicName:123"`)
- `thread`: for threading conversations (first message ID in thread)

**Location**: Documented in `docs/API.md` line 935

### 2. Message Delete Age Limit ✅
**Config**: `msg_delete_age` in `tinode.conf`

Already supports limiting how old messages can be when deleted by non-owners. Set in seconds (e.g., `600` = 10 minutes).

**Location**: `server/main.go` lines 315-319

### 3. P2P Delete Permission ✅
**Config**: `p2p_delete_enabled` in `tinode.conf`

Controls whether P2P participants can hard-delete messages.

### 4. Read/Recv Receipts ✅
Already fully implemented via `{note}` messages:
- `"what": "recv"` - message received
- `"what": "read"` - message read
- `"what": "kp"` - typing indicator

**Location**: `server/datamodel.go` lines 309-323

### 5. Drafty Rich Text Format ✅
Full support for:
- Bold, italic, strikethrough, code
- Links, mentions, hashtags
- Images, audio, video attachments
- File attachments
- Interactive buttons/forms

**Location**: `docs/drafty.md`

---

## Open Bugs (from GitHub)

### 1. Unread Message Counter Doesn't Account for Deleted Messages
**GitHub Issue**: [#898](https://github.com/tinode/chat/issues/898)

**Problem**: When messages are deleted, the unread counter doesn't update. If user1 is offline and user2 deletes messages, user1 gets incorrect count when coming online.

**Proposed Fix** (from issue):
1. In `{get}` query for "me" topic subs meta, count unread as `SeqId - ReadSeqId`
2. Query DB for deleted messages between `ReadSeqId` and `SeqId`
3. Subtract deleted count from unread count

**Location**: `server/topic.go` around line 2351 (`replyGetSub` function)

**Complexity**: Medium - requires additional DB query
**Priority**: Medium - affects UX but not critical for 1:1 chat

---

### 2. Group Topic Default Access Changes Not Propagated
**GitHub Issue**: [#720](https://github.com/tinode/chat/issues/720)

**Problem**: When group topic default access (defacs) is changed, existing members don't get updated.

**Relevance**: Low for our use case (1:1 chat only)

---

## Still Needed - Feature Requests

### 1. Message Expiration / Auto-Delete Timer ⭐
**GitHub Issue**: [#941](https://github.com/tinode/chat/issues/941)

**Description**: Automatically hard-delete messages after a configurable time period (disappearing messages).
- Currently only `msg_delete_age` exists (limits manual deletion age)
- Need: automatic timer-based deletion like Signal/WhatsApp

**Status**: NOT IMPLEMENTED - would need new feature
**Complexity**: Medium-High
**Priority**: Medium

---

### 2. Message Encryption at Rest ⭐
**GitHub Issue**: [#967](https://github.com/tinode/chat/issues/967)

**Description**: Encrypt messages stored in the database.
- NOT end-to-end encryption
- Server-side encryption for data at rest

**Status**: NOT IMPLEMENTED
**Complexity**: Medium
**Priority**: Medium-High for production

---

### 3. View Current Active Sessions
**GitHub Issue**: [#968](https://github.com/tinode/chat/issues/968)

**Description**: Allow users to see their active sessions across devices.

**Status**: NOT IMPLEMENTED (devices are tracked but not exposed to users)
**Complexity**: Low-Medium
**Priority**: Low

---

### 4. Location Sharing
**GitHub Issue**: [#963](https://github.com/tinode/chat/issues/963)

**Description**: Share location in chat messages.

**Status**: NOT IMPLEMENTED (but easy - just use Drafty with custom entity)
**Complexity**: Low (mostly client-side)
**Priority**: Low

---

### 5. Contact Sharing
**GitHub Issue**: [#964](https://github.com/tinode/chat/issues/964)

**Description**: Share contact cards in chat.

**Status**: NOT IMPLEMENTED (but easy - just use Drafty with custom entity)
**Complexity**: Low
**Priority**: Low

---

## Custom Improvements We Could Add

### 1. Delivery Timestamp in Receipts
**Current State**: `{note what="recv"}` doesn't include timestamp of when message was received.

**Improvement**: Add timestamp to receipt notifications so sender knows exactly when message was delivered/read.

**Current protocol**:
```json
{"note": {"topic": "usrXXX", "what": "recv", "seq": 123}}
```

**Could add**:
```json
{"note": {"topic": "usrXXX", "what": "recv", "seq": 123, "ts": "2025-12-31T12:00:00Z"}}
```

**Complexity**: Low
**Priority**: High (you specifically need sent/delivered/read with timestamps)

---

### 2. Message Reactions
**Current State**: Not supported.

**Improvement**: Add emoji reactions to messages (like iMessage, Slack).

**Complexity**: Medium (need new message type or head field)
**Priority**: Low (nice to have)

---

## Recommended Priority Order

For your Yahoo Messenger-style 1:1 chat app:

1. ✅ **Reply/Quote Messages** - Already implemented!
2. ✅ **Read/Delivered Receipts** - Already implemented!
3. ⚠️ **Delivery Timestamps** - Minor enhancement needed
4. 🐛 **Unread Counter Bug Fix** (#898) - Affects UX
5. 🔒 **Message Encryption at Rest** (#967) - Security
6. ⏱️ **Message Expiration** (#941) - Privacy feature

---

## Technical Notes

### Code Structure
- `server/` - Main server code
- `server/topic.go` - Topic/subscription handling
- `server/session.go` - Session management
- `server/db/postgres/adapter.go` - PostgreSQL adapter (we already modified this)
- `pbx/` - Protocol buffer definitions

### Building
```bash
# Build for PostgreSQL
go build -tags postgres -o tinode ./server
go build -tags postgres -o init-db ./tinode-db

# Or use our custom Dockerfile
docker build -f docker/tinode/Dockerfile.custom -t tinode-postgres-fixed:latest .
```

---

## Next Steps

1. [ ] Decide which improvements to prioritize
2. [ ] Create separate branches for each feature
3. [ ] Implement and test
4. [ ] Update documentation
