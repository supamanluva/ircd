# Channel Keys (+k Mode) Implementation

## Overview
Channel keys provide password protection for IRC channels, preventing unauthorized users from joining.

## Implementation Details

### Channel Structure Updates
- Added `key` field to `Channel` struct to store the channel password
- Implemented thread-safe key management with mutex locks

### New Methods in `internal/channel/channel.go`

#### `SetKey(key string)`
Sets the channel password. Called when an operator sets +k mode.

#### `GetKey() string`
Returns the current channel key.

#### `CheckKey(providedKey string) bool`
Validates if a provided key matches the channel's key. Returns `true` if:
- No key is set (channel is not password-protected)
- Provided key matches the channel key

### MODE Command Updates
Enhanced `handleChannelMode()` in `internal/commands/handler.go`:

**Setting a key:**
```
MODE #channel +k password
```
- Only channel operators can set keys
- Stores the password in the channel
- Enables +k mode flag

**Removing a key:**
```
MODE #channel -k
```
- Clears the channel password
- Disables +k mode flag

### JOIN Command Updates
Enhanced `handleJoin()` to support channel keys:

**Syntax:**
```
JOIN #channel1,#channel2 key1,key2
```

**Validation:**
- Parses optional keys parameter (comma-separated)
- Checks if channel has +k mode set
- Validates provided key before allowing join
- Sends `ERR_BADCHANNELKEY (475)` if key is wrong or missing

### Error Codes
- **ERR_BADCHANNELKEY (475)**: Sent when:
  - Wrong key is provided
  - No key is provided for a +k channel
  - Format: `475 <nick> <channel> :Cannot join channel (+k)`

## Usage Examples

### Setting a Channel Key
```
alice: JOIN #private
alice: MODE #private +k secretpass
```

### Joining with Key
```
bob: JOIN #private secretpass
```

### Joining without Key (Rejected)
```
charlie: JOIN #private
Server: 475 charlie #private :Cannot join channel (+k)
```

### Removing Key
```
alice: MODE #private -k
```

### Multiple Channels with Keys
```
JOIN #chan1,#chan2,#chan3 key1,key2,key3
```

## Security Considerations

1. **Plain Text Storage**: Keys are currently stored in plain text in memory
   - For production, consider using hashed keys
   - Keys are not exposed in WHO/WHOIS responses

2. **Key Visibility**: Keys are only known to:
   - The operator who set them
   - Users who successfully join the channel
   - Not shown in channel mode listings to non-members

3. **Operator Privileges**: Only channel operators can:
   - Set channel keys with +k
   - Remove channel keys with -k

## RFC 1459 Compliance

This implementation follows RFC 1459 specifications:
- Channel key is a mode parameter
- Keys are case-sensitive
- Maximum one key per channel
- Keys persist until changed or removed

## Testing

Test script: `tests/test_channel_keys.sh`

**Test Cases:**
1. Setting channel key with MODE +k
2. Joining with correct key
3. Joining with wrong key (rejected)
4. Joining without key (rejected)
5. Removing key with MODE -k
6. Joining after key removed

Run tests:
```bash
./tests/test_channel_keys.sh
```

## Related Files
- `internal/channel/channel.go` - Channel structure and key methods
- `internal/commands/handler.go` - MODE and JOIN command handlers
- `internal/commands/replies.go` - ERR_BADCHANNELKEY (475)
- `tests/test_channel_keys.sh` - Integration tests

## Future Enhancements
- [ ] Key hashing for security
- [ ] Key expiration/rotation
- [ ] Key strength validation
- [ ] Audit logging for key changes
- [ ] Support for multiple keys (invite exceptions)
