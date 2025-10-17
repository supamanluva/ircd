# Voice Mode (+v) Implementation

## Overview
Voice mode allows non-operator users to speak in moderated (+m) channels. This is useful for giving specific users speaking privileges without making them channel operators.

## Implementation Details

### Channel Structure Updates
- Added `voiced` map to track users with voice status
- Tracks nickname -> voice status mapping

### New Methods in `internal/channel/channel.go`

#### `IsVoiced(c *client.Client) bool`
Checks if a client has voice status in the channel.

#### `SetVoice(c *client.Client, hasVoice bool)`
Sets or removes voice status for a client. Thread-safe with mutex protection.

#### `CanSpeak(c *client.Client) bool`
Determines if a client can send messages in the channel:
- Returns `true` if channel is not moderated (+m)
- Returns `true` if user is a channel operator
- Returns `true` if user has voice (+v)
- Returns `false` otherwise

### Updated Methods

#### `RemoveMember()`
Now also removes voice status when a user leaves the channel.

#### `GetMemberNicks()`
Enhanced to show user prefixes:
- `@nickname` - Channel operator
- `+nickname` - Voiced user
- `nickname` - Regular user

### MODE Command Updates
Enhanced `handleChannelMode()` in `internal/commands/handler.go`:

**Granting voice:**
```
MODE #channel +v nickname
```
- Only channel operators can grant voice
- Adds user to voiced map
- User can now speak in moderated channel

**Removing voice:**
```
MODE #channel -v nickname
```
- Only channel operators can remove voice
- Removes user from voiced map
- User loses speaking privileges in moderated channel

### PRIVMSG/NOTICE Updates
Enhanced `handleMessage()` to enforce moderation:

**Checks:**
1. Is the channel moderated (+m)?
2. If yes, can the user speak? (`CanSpeak()`)
3. Sends `ERR_CANNOTSENDTOCHAN (404)` if user cannot speak

**Error Message:**
```
404 <nick> <channel> :Cannot send to channel (+m)
```

## Usage Examples

### Setting Moderated Mode
```
alice: JOIN #meeting
alice: MODE #meeting +m
```

### Regular User Blocked
```
bob: JOIN #meeting
bob: PRIVMSG #meeting :Hello?
Server: 404 bob #meeting :Cannot send to channel (+m)
```

### Granting Voice
```
alice: MODE #meeting +v bob
```

### Voiced User Can Speak
```
bob: PRIVMSG #meeting :Thanks for voice!
(Message is delivered successfully)
```

### Checking NAMES with Voice
```
alice: NAMES #meeting
Server: 353 alice = #meeting :@alice +bob charlie
```
- `@alice` - Operator (can always speak)
- `+bob` - Voiced (can speak in +m)
- `charlie` - Regular (cannot speak in +m)

### Removing Voice
```
alice: MODE #meeting -v bob
```

### Multiple Voice Operations
```
MODE #meeting +vvv bob charlie dave
MODE #meeting -vv bob charlie
```

## Mode Interaction

### Precedence
1. **Operators** - Can always speak and manage modes
2. **Voiced** - Can speak in moderated channels
3. **Regular** - Can only speak in non-moderated channels

### Combined Modes
```
# Moderated and invite-only channel
MODE #private +mi
MODE #private +v trusted_user
```

## RFC 1459 Compliance

This implementation follows IRC standards:
- Voice is a user-level channel mode
- Only operators can grant/remove voice
- Voice persists until explicitly removed or user leaves
- Voice is indicated with + prefix in NAMES
- Voice users can speak in +m channels

## Benefits

### For Channel Management
- **Granular Control**: Give speaking rights without full operator power
- **Structured Discussions**: Control who can participate
- **Q&A Sessions**: Moderate questions while allowing select speakers
- **Anti-Spam**: Limit general chat while allowing trusted users

### Use Cases
1. **Town Halls**: Operators ask questions, voiced users can respond
2. **Support Channels**: Helpers get voice to assist
3. **Presentations**: Speaker has voice, audience is silent
4. **Moderated Debates**: Selected participants get voice

## Testing

Test script: `tests/test_voice_mode.sh`

**Test Cases:**
1. Setting moderated mode (+m)
2. Regular user blocked in moderated channel
3. Granting voice with MODE +v
4. Voiced user can speak in moderated channel
5. NAMES shows + prefix for voiced users
6. Removing voice with MODE -v

Run tests:
```bash
./tests/test_voice_mode.sh
```

## Security Considerations

1. **Operator Only**: Only channel operators can grant/remove voice
   - Prevents unauthorized privilege escalation
   - Maintains channel operator authority

2. **Temporary Status**: Voice is lost when:
   - User leaves the channel
   - Operator removes voice with -v
   - Channel is destroyed

3. **No Persistence**: Voice status doesn't persist across:
   - Server restarts
   - User reconnections
   - Channel recreations

## Related Files
- `internal/channel/channel.go` - Channel structure and voice methods
- `internal/commands/handler.go` - MODE and PRIVMSG handlers
- `internal/commands/replies.go` - ERR_CANNOTSENDTOCHAN (404)
- `tests/test_voice_mode.sh` - Integration tests

## Future Enhancements
- [ ] Persistent voice lists (auto-voice on join)
- [ ] Voice expiration/time limits
- [ ] Voice level hierarchy (+v vs +vv)
- [ ] Voice through authentication
- [ ] Voice groups/roles
