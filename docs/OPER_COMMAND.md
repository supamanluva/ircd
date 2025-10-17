# OPER Command Implementation

## Overview
The OPER command allows authorized users to gain server operator (IRCop) privileges. Server operators have elevated permissions for server management and moderation.

## Implementation Details

### Configuration Structure
Operators are configured in `config/config.yaml`:

```yaml
operators:
  - name: "admin"
    password: "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy"
  - name: "oper"
    password: "$2a$10$e0MYzXyjpJS7Pd94qMTnYu8qgx7Ky5.XYVzMSrVPXpLDXbDdSQT0W"
```

**Password Hashing:**
- Passwords are stored as bcrypt hashes (cost factor 10)
- Never store plain-text passwords in config

### Server Configuration (`internal/server/server.go`)

**Added to Config struct:**
```go
type Config struct {
    // ... existing fields ...
    Operators []Operator // Server operators
}

type Operator struct {
    Name     string
    Password string // bcrypt hashed password
}
```

### Command Handler (`internal/commands/handler.go`)

**Handler Enhancement:**
```go
type Handler struct {
    // ... existing fields ...
    operators map[string]string // name -> bcrypt password hash
}
```

**Operator Type:**
```go
type Operator struct {
    Name     string
    Password string // bcrypt hashed
}
```

### OPER Command Handler

**Function:** `handleOper(c *client.Client, msg *parser.Message) error`

**Syntax:**
```
OPER <name> <password>
```

**Process:**
1. Verify user is registered
2. Check parameters (need name and password)
3. Lookup operator name in configuration
4. Verify password using bcrypt.CompareHashAndPassword()
5. Grant operator mode (+o) to user
6. Send RPL_YOUREOPER (381) confirmation

**Error Handling:**
- Missing parameters → `ERR_NEEDMOREPARAMS (461)`
- Wrong name or password → `ERR_PASSWDMISMATCH (464)`

### Numeric Replies

**RPL_YOUREOPER (381):**
```
381 <nick> :You are now an IRC operator
```

**ERR_PASSWDMISMATCH (464):**
```
464 <nick> :Password incorrect
```

**ERR_NEEDMOREPARAMS (461):**
```
461 <nick> OPER :Not enough parameters
```

## Usage Examples

### Successful Authentication
```
alice: OPER admin admin123
Server: 381 alice :You are now an IRC operator
```

### Wrong Password
```
bob: OPER admin wrongpass
Server: 464 bob :Password incorrect
```

### Unknown Operator Name
```
charlie: OPER unknown password
Server: 464 charlie :Password incorrect
```

### Missing Parameters
```
dave: OPER admin
Server: 461 dave OPER :Not enough parameters
```

### Operator in WHOIS
After successful OPER:
```
eve: WHOIS alice
Server: 313 eve alice :is an IRC operator
```

## Generating Password Hashes

### Using htpasswd (Recommended)
```bash
htpasswd -bnBC 10 "" yourpassword | tr -d ':\n' | sed 's/$2y/$2a/'
```

### Using Go
```go
package main

import (
    "fmt"
    "golang.org/x/crypto/bcrypt"
)

func main() {
    password := "yourpassword"
    hash, _ := bcrypt.GenerateFromPassword([]byte(password), 10)
    fmt.Println(string(hash))
}
```

### Using Online Tools
- https://bcrypt-generator.com/
- Set rounds to 10
- Copy the $2a$ or $2b$ hash

## Security Considerations

### Password Security
1. **bcrypt Hashing**: Uses bcrypt with cost factor 10
   - Computationally expensive (prevents brute force)
   - Salted (prevents rainbow tables)
   - Slow verification (rate limiting built-in)

2. **No Plain-Text Storage**: Passwords never stored in plain text
   - Config file contains only hashes
   - Memory contains only hashes
   - Logs do not expose passwords

3. **Constant-Time Comparison**: bcrypt.CompareHashAndPassword uses constant-time comparison
   - Prevents timing attacks

### Access Control
1. **Configuration-Based**: Operators defined in config file
   - Requires server restart to add/remove operators
   - Prevents runtime privilege escalation

2. **Audit Logging**: All OPER attempts are logged
   - Successful: Info level with nickname and oper name
   - Failed: Warn level with attempt details

3. **Mode Tracking**: Operator status tracked via +o mode
   - Visible in WHOIS
   - Can be removed with MODE -o
   - Persists for session only

## Operator Privileges

Once authenticated, operators gain:
- **+o mode**: Operator flag set on user
- **Enhanced visibility**: Shown in WHOIS with RPL_WHOISOPERATOR (313)
- **Future capabilities**: Ready for additional oper-only commands

### Not Yet Implemented (Future)
- KILL - Forcibly disconnect users
- KLINE - Ban users by mask
- REHASH - Reload configuration
- WALLOPS - Broadcast to all operators
- CONNECT/SQUIT - Server linking

## Configuration File Integration

### Config Loading (`cmd/ircd/main.go`)

Operators are loaded from YAML:
```go
var configData struct {
    // ... other fields ...
    Operators []struct {
        Name     string `yaml:"name"`
        Password string `yaml:"password"`
    } `yaml:"operators"`
}
```

Converted to server config:
```go
operators := make([]server.Operator, len(configData.Operators))
for i, op := range configData.Operators {
    operators[i] = server.Operator{
        Name:     op.Name,
        Password: op.Password,
    }
}
```

### Handler Initialization

Operators passed to command handler:
```go
cmdOperators := make([]commands.Operator, len(cfg.Operators))
for i, op := range cfg.Operators {
    cmdOperators[i] = commands.Operator{
        Name:     op.Name,
        Password: op.Password,
    }
}

srv.handler = commands.New(cfg.ServerName, log, srv, srv, cmdOperators)
```

## Testing

Test script: `tests/test_oper.sh`

**Test Cases:**
1. Successful OPER with correct credentials (admin/admin123)
2. Failed OPER with wrong password
3. Failed OPER with unknown operator name
4. Failed OPER without enough parameters
5. Second operator credentials work (oper/oper456)
6. Operator status visible in WHOIS

Run tests:
```bash
./tests/test_oper.sh
```

## RFC Compliance

### RFC 1459 Section 4.1.5 - OPER Command

**Command Format:**
```
OPER <user> <password>
```

**Replies:**
- RPL_YOUREOPER (381)
- ERR_NEEDMOREPARAMS (461)
- ERR_PASSWDMISMATCH (464)
- ERR_NOOPERHOST (491) - Not implemented (host checking)

This implementation follows RFC 1459 specifications with modern security enhancements (bcrypt instead of plain-text passwords).

## Dependencies

**Added:**
- `golang.org/x/crypto/bcrypt` - Password hashing and verification

## Related Files
- `internal/commands/handler.go` - OPER command handler
- `internal/commands/replies.go` - RPL_YOUREOPER (381), ERR_PASSWDMISMATCH (464)
- `internal/server/server.go` - Config struct with Operators
- `cmd/ircd/main.go` - Config loading
- `config/config.yaml` - Operator configuration
- `tests/test_oper.sh` - Integration tests

## Future Enhancements
- [ ] Host-based operator authentication (ERR_NOOPERHOST)
- [ ] Operator classes with different privilege levels
- [ ] OPER-only commands (KILL, KLINE, REHASH, etc.)
- [ ] Operator message broadcast (WALLOPS)
- [ ] Connection class restrictions
- [ ] Two-factor authentication
- [ ] Audit log for operator actions
- [ ] Configurable password rotation
- [ ] SASL authentication for operators

## Example Configuration

```yaml
# config/config.yaml
operators:
  # Primary admin
  - name: "admin"
    password: "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy"  # admin123
  
  # Secondary operator
  - name: "oper"
    password: "$2a$10$e0MYzXyjpJS7Pd94qMTnYu8qgx7Ky5.XYVzMSrVPXpLDXbDdSQT0W"  # oper456
  
  # Moderator
  - name: "mod"
    password: "$2a$10$Xy4aAbBEZKN6z7DeadBeefPassword123HashGoesHere"  # mod789
```

## Security Best Practices

1. **Strong Passwords**: Use long, random passwords for operators
2. **Unique Credentials**: Each operator should have unique credentials
3. **Regular Rotation**: Change operator passwords periodically
4. **Audit Logs**: Monitor OPER attempts in logs
5. **Access Control**: Limit who has access to config file
6. **Backup Security**: Encrypt config backups containing hashed passwords
7. **Session Management**: Operator status doesn't persist across reconnects
