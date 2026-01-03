# Server Issue: Reactions Not Returned in Message Data

## ✅ RESOLVED - Server is working correctly

## Problem (Original)
When fetching messages with `{get what:"data"}`, the server appeared to not return the `head` field containing reactions.

## Investigation Result

**The server IS returning reactions correctly.** Tested on 2025-12-31:

```json
{
  "data": {
    "topic": "usrVqvAnkvyYco",
    "from": "usrtQbI2DNoXMU",
    "ts": "2025-12-31T22:18:17.764Z",
    "seq": 1,
    "head": {
      "reactions": {
        "👍": ["usrtQbI2DNoXMU"]
      }
    },
    "content": "Test message for reactions"
  }
}
```

## Root Cause
Messages without reactions have `head: null` in the database, so the `head` field is omitted from the JSON response (due to `omitempty` tag). This is expected behavior.

**Only messages WITH reactions will have the `head` field in the response.**

## Client-Side Fix
The client app needs to:
1. Check if `message.head` exists before accessing `message.head.reactions`
2. Handle the case where `head` is undefined/null

```javascript
// Safe way to get reactions
const reactions = message.head?.reactions || {};
```

## Verification
Database shows reactions are stored correctly:
```sql
SELECT seqid, topic, head FROM messages WHERE head::text != 'null';

 seqid |           topic           |                  head                   
-------+---------------------------+-----------------------------------------
    26 | p2pVqvAnkvyYcp627-g5Pdw8w | {"reactions":{"😝":["usretu_oOT3cPM"]}}
    29 | p2pVqvAnkvyYcp627-g5Pdw8w | {"reactions":{"😘":["usretu_oOT3cPM","usrVqvAnkvyYco"]}}
```
