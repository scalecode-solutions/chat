# Clingy Chat Backend - Feature Roadmap

This document outlines planned features for our custom Tinode fork, organized by priority and implementation complexity.

## ✅ Already Implemented

| Feature | Description | Status |
|---------|-------------|--------|
| **Emoji Reactions** | React to messages with emojis, stored in message `head` field | ✅ Complete |
| **Message Encryption at Rest** | AES-256-GCM encryption of message content in database | ✅ Complete |
| **Extended Token Expiration** | JWT tokens valid for 2 years instead of 14 days | ✅ Complete |

---

## 🔴 High Priority - Safety & Privacy Features

These features are critical for Clingy's core mission as a safety communication tool.

### 1. Disappearing Messages (Auto-Delete Timer)
**Tinode Issue:** #941

**Description:** Automatically hard-delete messages after a configurable time period (e.g., 1 hour, 24 hours, 7 days).

**Why It Matters:**
- Messages auto-delete even if abuser gains access to phone
- No evidence trail if device is compromised
- User can set per-conversation or global timer

**Implementation Notes:**
- Add `delete_after` field to message or topic settings
- Background job to purge expired messages
- Client UI to configure timer per conversation

**Complexity:** Medium

---

### 2. View Active Sessions
**Tinode Issue:** #968

**Description:** Allow users to see all devices/sessions currently logged into their account.

**Why It Matters:**
- Detect if abuser has secretly logged into account
- Ability to remotely log out other sessions
- Security awareness

**Implementation Notes:**
- Track sessions in database with device info, IP, last active
- API endpoint to list/revoke sessions
- Push notification when new device logs in

**Complexity:** Medium

---

### 3. Restrict Message Deletion to Sender Only
**Tinode Issue:** #940

**Description:** Option to prevent recipients from deleting messages (only sender can delete their own messages).

**Why It Matters:**
- Preserves evidence if abuser gains access
- Prevents tampering with conversation history
- Important for legal documentation

**Implementation Notes:**
- Add permission flag to topic settings
- Modify delete handler to check sender

**Complexity:** Low

---

### 4. Panic Wipe / Remote Wipe
**Not in Tinode Issues - Custom Feature**

**Description:** Ability to remotely wipe all messages and chat data from a device.

**Why It Matters:**
- If phone is taken, user can wipe from another device
- Emergency "burn everything" option
- Trusted contact could trigger wipe

**Implementation Notes:**
- Special server command to mark account for wipe
- Client checks on connect and clears local data
- Optional: wipe on failed PIN attempts

**Complexity:** Medium

---

## 🟡 Medium Priority - Communication Features

### 5. Voice Messages
**Tinode Issue:** #966

**Description:** Record and send audio messages.

**Why It Matters:**
- Faster than typing when time is limited
- Can communicate when can't look at screen
- More personal connection

**Implementation Notes:**
- Audio recording in client
- Upload as attachment with special type
- Playback UI in chat

**Complexity:** Medium (mostly client-side)

---

### 6. Location Sharing
**Tinode Issue:** #963

**Description:** Share current GPS location with a contact.

**Why It Matters:**
- Emergency "I'm here" to trusted contact
- Meet-up coordination
- Safety check-ins

**Implementation Notes:**
- Client gets GPS coordinates
- Send as special message type with map preview
- Option for live location sharing (updates periodically)

**Complexity:** Medium

---

### 7. Video Messages
**Tinode Issue:** #821

**Description:** Record and send short video clips.

**Why It Matters:**
- Document situations with video evidence
- More expressive communication
- Show rather than tell

**Implementation Notes:**
- Video recording in client
- Compression before upload
- Thumbnail generation
- Playback UI

**Complexity:** High

---

### 8. Contact/Info Sharing
**Tinode Issue:** #964

**Description:** Share contact cards (name, phone, address) as structured messages.

**Why It Matters:**
- Share shelter contact info easily
- Share lawyer/advocate details
- Structured data vs plain text

**Implementation Notes:**
- vCard-like format in message content
- Rich preview in chat
- Tap to save to contacts

**Complexity:** Low

---

## 🟢 Standard Chat Features (Parity with Other Apps)

These features make the chat backend competitive with mainstream apps.

### 9. Message Editing
**Description:** Edit sent messages within a time window.

**Implementation Notes:**
- Store edit history
- Show "edited" indicator
- Time limit (e.g., 15 minutes)

**Complexity:** Medium

---

### 10. Reply/Quote Messages
**Description:** Reply to a specific message with quote preview.

**Implementation Notes:**
- Reference original message seq ID in `head`
- Client renders quoted message above reply

**Complexity:** Low (mostly client-side)

---

### 11. Forward Messages
**Description:** Forward a message to another conversation.

**Implementation Notes:**
- Copy message content to new topic
- Optional: show "forwarded" indicator

**Complexity:** Low

---

### 12. Pin Messages
**Description:** Pin important messages to top of conversation.

**Implementation Notes:**
- Add pinned message IDs to topic metadata
- Client UI to show pinned messages

**Complexity:** Low

---

### 13. Message Search
**Description:** Full-text search across messages.

**Implementation Notes:**
- PostgreSQL full-text search on decrypted content
- Or: client-side search of local messages
- Challenge: encrypted messages need decryption for search

**Complexity:** High (due to encryption)

---

### 14. Typing Indicators
**Description:** Show when other person is typing.

**Status:** ✅ Already supported by Tinode (`{note what="kp"}`)

---

### 15. Read Receipts
**Description:** Show when messages are delivered and read.

**Status:** ✅ Already supported by Tinode (`{note what="recv/read"}`)

---

### 16. Online/Presence Status
**Description:** Show when users are online/offline/away.

**Status:** ✅ Already supported by Tinode (presence in `me` topic)

---

### 17. Message Threads
**Description:** Create threaded conversations within a chat.

**Implementation Notes:**
- Messages can have parent_seq reference
- UI groups threaded messages together

**Complexity:** Medium

---

### 18. Scheduled Messages
**Description:** Schedule a message to send at a future time.

**Implementation Notes:**
- Store scheduled messages in separate table
- Background job to send at scheduled time
- Useful for: "If I don't check in by 6pm, send this message"

**Complexity:** Medium

---

### 19. Polls
**Description:** Create simple polls in chat.

**Implementation Notes:**
- Special message type with options
- Track votes in message head
- Show results inline

**Complexity:** Medium

---

### 20. File Sharing (General)
**Description:** Share documents, PDFs, etc.

**Status:** ✅ Already supported by Tinode (attachments)

---

## 🔵 Advanced Features (Future)

### 21. Group Chats
**Description:** Multi-person conversations.

**Status:** ✅ Already supported by Tinode (grp topics)

**Clingy consideration:** May not be needed for core use case, but useful for support groups.

---

### 22. Channels/Broadcasts
**Description:** One-to-many broadcast messages.

**Status:** ✅ Already supported by Tinode (chn topics)

---

### 23. Bots/Automation
**Description:** Automated responses, chatbots.

**Status:** ✅ Already supported by Tinode

**Clingy consideration:** Could have a "safety bot" that sends check-in reminders.

---

### 24. Video/Voice Calls
**Description:** Real-time audio/video calls.

**Status:** Partially supported by Tinode (WebRTC signaling)

**Complexity:** Very High

**Clingy consideration:** Lower priority - calls can be traced/overheard.

---

### 25. Stories/Status Updates
**Description:** Ephemeral status updates visible to contacts.

**Complexity:** High

**Clingy consideration:** Not relevant for safety use case.

---

## 📊 Implementation Priority Matrix

| Feature | Safety Value | General Value | Complexity | Priority |
|---------|--------------|---------------|------------|----------|
| Disappearing Messages | 🔴 Critical | Medium | Medium | **1** |
| View Active Sessions | 🔴 Critical | High | Medium | **2** |
| Voice Messages | Medium | High | Medium | **3** |
| Panic Wipe | 🔴 Critical | Low | Medium | **4** |
| Reply/Quote | Low | High | Low | **5** |
| Message Editing | Low | High | Medium | **6** |
| Location Sharing | High | Medium | Medium | **7** |
| Scheduled Messages | High | Medium | Medium | **8** |
| Restrict Deletion | High | Low | Low | **9** |
| Pin Messages | Low | Medium | Low | **10** |

---

---

## 🌐 Features from Other Chat Platforms

### From WhatsApp
| Feature | Description | Relevance |
|---------|-------------|-----------|
| **Disappearing messages** | Auto-delete after 24h/7d/90d | 🔴 Critical for safety |
| **View once media** | Photo/video viewable only once | 🔴 Critical for safety |
| **Two-step verification** | PIN required to register on new device | High |
| **Fingerprint/Face lock** | Biometric to open app | Medium (we have PIN) |
| **Chat backup encryption** | E2E encrypted backups | Medium |
| **Message reactions** | ✅ Implemented | - |
| **Broadcast lists** | Send to multiple contacts at once | Low |
| **Status/Stories** | 24h ephemeral posts | Low |

### From Signal
| Feature | Description | Relevance |
|---------|-------------|-----------|
| **Disappearing messages** | Per-conversation timer | 🔴 Critical |
| **Screen security** | Block screenshots | 🔴 Critical for safety |
| **Incognito keyboard** | Disable keyboard learning | High |
| **Relay calls through Signal** | Hide IP address | Medium |
| **Sealed sender** | Hide metadata from server | High |
| **Note to self** | Message yourself | Low |
| **Registration lock** | PIN to re-register | High |

### From Telegram
| Feature | Description | Relevance |
|---------|-------------|-----------|
| **Secret chats** | E2E encrypted, device-specific | Medium |
| **Self-destruct timer** | Messages delete after read | 🔴 Critical |
| **Edit messages** | Edit sent messages | Medium |
| **Delete for everyone** | Delete messages from all devices | High |
| **Slow mode** | Limit message frequency in groups | Low |
| **Scheduled messages** | Send at specific time | High |
| **Silent messages** | Send without notification sound | High for safety |
| **Saved messages** | Bookmark messages | Low |
| **Chat folders** | Organize conversations | Low |
| **Nearby people** | Find users nearby | ❌ Dangerous for Clingy |

### From iMessage
| Feature | Description | Relevance |
|---------|-------------|-----------|
| **Tapback reactions** | ✅ Implemented | - |
| **Message effects** | Animations on send | Low |
| **Digital touch** | Draw/heartbeat | Low |
| **Inline replies** | Reply to specific message | Medium |
| **Pin conversations** | Pin important chats to top | Medium |
| **Mentions** | @mention in groups | Low |
| **Edit/Unsend** | Edit or unsend recent messages | Medium |

### From Slack/Discord (Business/Community)
| Feature | Description | Relevance |
|---------|-------------|-----------|
| **Threads** | Threaded replies | Low |
| **Reactions** | ✅ Implemented | - |
| **Channels** | Topic-based conversations | Low |
| **User roles/permissions** | Admin, moderator, etc. | Low |
| **Integrations/Webhooks** | Connect external services | Medium |
| **Search** | Full-text message search | Medium |
| **Bookmarks/Saved items** | Save important messages | Low |
| **Custom emoji** | Upload custom emoji | Low |
| **Status** | Custom status message | Low |

### From Messenger/Instagram DMs
| Feature | Description | Relevance |
|---------|-------------|-----------|
| **Vanish mode** | Messages disappear when chat closed | 🔴 Critical |
| **Message requests** | Approve messages from strangers | Medium |
| **Nicknames** | Custom names for contacts | Low |
| **Chat themes/colors** | Customize chat appearance | Low |
| **Quick reactions** | Double-tap to react | Low |
| **Polls** | Create polls in chat | Low |
| **Watch together** | Shared media viewing | Low |

---

## 🛡️ Safety-Specific Features (Not in Mainstream Apps)

These are custom features specifically for Clingy's safety mission.

### 1. Duress PIN / Decoy Mode
**Description:** A second PIN that opens a fake/decoy chat with innocent messages.

**Why It Matters:** If forced to reveal PIN, user shows decoy. Real chat stays hidden.

**Implementation:**
- Two PINs stored: real and decoy
- Decoy PIN opens pre-populated innocent conversation
- Real PIN opens actual hidden chat

---

### 2. Check-In System
**Description:** Scheduled check-ins. If user doesn't check in, alert trusted contact.

**Why It Matters:** Dead man's switch for safety.

**Implementation:**
- User sets check-in schedule (e.g., every 24 hours)
- If missed, server sends alert to designated contact
- Optional: send pre-written emergency message

---

### 3. Silent Panic Button
**Description:** Shake phone or press volume buttons to silently alert trusted contact with location.

**Why It Matters:** Emergency alert without touching screen.

**Implementation:**
- Background listener for shake/button combo
- Sends GPS + timestamp to emergency contact
- No visible indication on phone

---

### 4. Evidence Export
**Description:** Export chat history in court-admissible format with timestamps and metadata.

**Why It Matters:** Legal proceedings require documented evidence.

**Implementation:**
- Generate PDF/HTML with full chat history
- Include message timestamps, delivery receipts
- Cryptographic signature for authenticity
- Export to email or secure cloud

---

### 5. Trusted Contact Verification
**Description:** Verify contacts are who they say they are via QR code or safety question.

**Why It Matters:** Prevent abuser from impersonating trusted contact.

**Implementation:**
- QR code exchange in person
- Or: shared secret question only both know
- Verified badge on contact

---

### 6. Network Anonymization
**Description:** Route traffic through proxy/Tor to hide that user is using Clingy.

**Why It Matters:** Abuser monitoring network can't see app traffic.

**Implementation:**
- Optional Tor/proxy routing
- Domain fronting to look like normal traffic
- High complexity, future consideration

---

### 7. Fake App Icon/Name
**Description:** Change app icon and name to something else (e.g., "Notes", "Calculator").

**Why It Matters:** Additional layer of disguise on home screen.

**Implementation:**
- iOS: Limited (alternate icons)
- Android: Activity aliases for different icons
- Complexity: Medium

---

### 8. Offline Message Queue
**Description:** Queue messages when offline, send when connection restored.

**Why It Matters:** Unreliable network situations, limited connectivity windows.

**Implementation:**
- Local queue with retry logic
- Already partially supported by Tinode

---

## Notes

- All features should maintain the "innocent pregnancy app" cover
- No features should generate suspicious notifications
- Consider offline-first for unreliable network situations
- Legal compliance: maintain audit trail for evidence purposes (configurable per deployment)
- Some features (like nearby people) are explicitly dangerous for Clingy's use case
