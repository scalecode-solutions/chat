# Tinode Improvements Brainstorm

This document tracks potential improvements, bug fixes, and feature additions for our custom Tinode fork.

---

## Already Implemented ✅

### 1. PostgreSQL Database Creation Fix
**Issue**: Official `tinode/tinode-postgres` image fails when database already exists but has no schema.

**Fix**: Modified `server/db/postgres/adapter.go` `CreateDb()` to catch PostgreSQL error code `42P04` (duplicate_database) and continue instead of failing.

**Status**: ✅ Deployed and working

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

## Feature Requests (Relevant to Our Use Case)

### 1. Message Expiration / Auto-Delete Timer ⭐
**GitHub Issue**: [#941](https://github.com/tinode/chat/issues/941)

**Description**: Automatically hard-delete messages after a configurable time period.
- Either party can set expiration time for P2P topics
- Site admin can enable/disable feature or force deletion

**Why Useful**: Privacy feature, reduces storage, "disappearing messages" like Signal/WhatsApp

**Complexity**: Medium-High
**Priority**: Medium

---

### 2. Message Encryption at Rest ⭐
**GitHub Issue**: [#967](https://github.com/tinode/chat/issues/967)

**Description**: Encrypt messages stored in the database to prevent access via DB tools.
- NOT end-to-end encryption (that's a separate issue #357)
- Server-side encryption for data at rest

**Why Useful**: Security/compliance, protects against DB breaches

**Complexity**: Medium
**Priority**: Medium-High for production use

---

### 3. View Current Active Sessions
**GitHub Issue**: [#968](https://github.com/tinode/chat/issues/968)

**Description**: Allow users to see their active sessions across devices.

**Why Useful**: Security feature, users can see if someone else is logged in

**Complexity**: Low-Medium
**Priority**: Low

---

### 4. Location Sharing
**GitHub Issue**: [#963](https://github.com/tinode/chat/issues/963)

**Description**: Share location in chat messages.

**Why Useful**: Common messaging feature

**Complexity**: Low (mostly client-side, server just stores coordinates)
**Priority**: Low

---

### 5. Contact Sharing
**GitHub Issue**: [#964](https://github.com/tinode/chat/issues/964)

**Description**: Share contact cards in chat.

**Why Useful**: Common messaging feature

**Complexity**: Low
**Priority**: Low

---

## Custom Improvements We Could Add

### 1. Better Delivery/Read Receipt Timestamps
**Current State**: Tinode has `recv` and `read` notifications but timestamps could be more granular.

**Improvement**: Add precise timestamps for:
- Message sent
- Message delivered to server
- Message delivered to recipient device
- Message read by recipient

**Complexity**: Low-Medium
**Priority**: High (you specifically mentioned this requirement)

---

### 2. Improved Offline Message Queueing Visibility
**Current State**: Messages queue on server when recipient is offline.

**Improvement**: Add API to query pending/queued messages count, delivery status.

**Complexity**: Low
**Priority**: Medium

---

### 3. Typing Indicator Improvements
**Current State**: Basic "kp" (key press) notification.

**Improvement**: 
- Add "stopped typing" detection (timeout-based)
- Add "recording audio/video" status

**Complexity**: Low
**Priority**: Low

---

### 4. Message Reactions
**Current State**: Not supported.

**Improvement**: Add emoji reactions to messages (like iMessage, Slack).

**Complexity**: Medium
**Priority**: Low (nice to have)

---

### 5. Reply/Quote Messages
**Current State**: Not natively supported.

**Improvement**: Add ability to reply to specific messages with quote.

**Complexity**: Medium
**Priority**: Medium (common feature in modern messengers)

---

## Recommended Priority Order

For a Yahoo Messenger-style 1:1 chat app:

1. **Delivery/Read Receipt Timestamps** - You specifically need this
2. **Unread Counter Bug Fix** (#898) - Affects UX
3. **Message Encryption at Rest** (#967) - Security
4. **Message Expiration** (#941) - Privacy feature
5. **Reply/Quote Messages** - Modern UX expectation

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
