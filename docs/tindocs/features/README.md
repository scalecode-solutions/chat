# Tinode Feature Specifications

This folder contains detailed implementation specifications for new features being added to our custom Tinode fork.

## Features

| Feature | Status | Spec |
|---------|--------|------|
| [Reply/Quote Messages](./REPLY_QUOTE.md) | 📋 Planned | Store reference to original message |
| [Edit/Unsend Messages](./EDIT_UNSEND.md) | 📋 Planned | Modify or retract sent messages |
| [Enhanced Delete](./ENHANCED_DELETE.md) | 📋 Planned | Delete for self vs delete for everyone |
| [Disappearing Messages](./DISAPPEARING_MESSAGES.md) | 📋 Planned | Auto-delete after time period |
| [Pin Messages](./PIN_MESSAGES.md) | 📋 Planned | Pin important messages to topic |

## Already Implemented

| Feature | Status |
|---------|--------|
| Emoji Reactions | ✅ Complete |
| Message Encryption at Rest | ✅ Complete |
| Extended Token Expiration | ✅ Complete |

## Implementation Priority

1. **Reply/Quote** - Low complexity, high value
2. **Edit/Unsend** - Medium complexity, high value
3. **Enhanced Delete** - Low complexity, medium value (partially exists)
4. **Pin Messages** - Low complexity, medium value
5. **Disappearing Messages** - Medium complexity, high value for safety

## General Approach

All features follow the same pattern:
1. **Protocol** - Define new message types or extend existing ones
2. **Server** - Implement handlers in Go
3. **Database** - Add/modify schema if needed
4. **Client** - Implement UI in React Native (Clingy) or other clients
