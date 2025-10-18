package linking

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Protocol constants
const (
	TS6Version    = 6
	MinTSVersion  = 6
	ProtocolName  = "TS"
)

// Capabilities that can be negotiated
var DefaultCapabilities = []string{
	"QS",      // Quit Storm - batch QUIT messages
	"EX",      // Exceptions - ban exceptions
	"CHW",     // Channel WHO - WHO in channels
	"IE",      // Invite Exceptions
	"KLN",     // K-line (bans)
	"UNKLN",   // Remove K-line
	"ENCAP",   // Encapsulated commands
	"SERVICES", // Services support
	"EUID",    // Extended UID
	"EOPMOD",  // Op moderation
	"MLOCK",   // Mode lock
}

// Message represents a server-to-server protocol message
type Message struct {
	Source  string   // Source SID or UID (can be empty for initial handshake)
	Command string   // Command name (PASS, SERVER, UID, etc)
	Params  []string // Command parameters
}

// ParseMessage parses a server protocol message
// Format: [:source] COMMAND param1 param2 ... :trailing
func ParseMessage(line string) (*Message, error) {
	if line == "" {
		return nil, fmt.Errorf("empty message")
	}
	
	msg := &Message{}
	parts := strings.Fields(line)
	if len(parts) == 0 {
		return nil, fmt.Errorf("empty message")
	}
	
	idx := 0
	
	// Check for source prefix
	if parts[0][0] == ':' {
		msg.Source = parts[0][1:]
		idx = 1
		if idx >= len(parts) {
			return nil, fmt.Errorf("message has source but no command")
		}
	}
	
	// Get command
	msg.Command = strings.ToUpper(parts[idx])
	idx++
	
	// Parse parameters
	msg.Params = []string{}
	for idx < len(parts) {
		if parts[idx][0] == ':' {
			// Trailing parameter - join rest of message
			trailing := strings.Join(parts[idx:], " ")
			msg.Params = append(msg.Params, trailing[1:])
			break
		}
		msg.Params = append(msg.Params, parts[idx])
		idx++
	}
	
	return msg, nil
}

// String formats a message for sending
func (m *Message) String() string {
	var sb strings.Builder
	
	if m.Source != "" {
		sb.WriteString(":")
		sb.WriteString(m.Source)
		sb.WriteString(" ")
	}
	
	sb.WriteString(m.Command)
	
	for i, param := range m.Params {
		sb.WriteString(" ")
		// Last param with space needs : prefix
		if i == len(m.Params)-1 && strings.Contains(param, " ") {
			sb.WriteString(":")
		}
		sb.WriteString(param)
	}
	
	return sb.String()
}

// BuildPASS creates a PASS message
// Format: PASS <password> TS <version> <SID>
func BuildPASS(password, sid string) *Message {
	return &Message{
		Command: "PASS",
		Params:  []string{password, "TS", strconv.Itoa(TS6Version), sid},
	}
}

// ParsePASS parses a PASS message
func ParsePASS(msg *Message) (password, sid string, version int, err error) {
	if len(msg.Params) < 4 {
		return "", "", 0, fmt.Errorf("PASS requires 4 parameters")
	}
	
	password = msg.Params[0]
	
	if msg.Params[1] != "TS" {
		return "", "", 0, fmt.Errorf("unknown protocol: %s", msg.Params[1])
	}
	
	version, err = strconv.Atoi(msg.Params[2])
	if err != nil {
		return "", "", 0, fmt.Errorf("invalid TS version: %s", msg.Params[2])
	}
	
	sid = msg.Params[3]
	
	if !ValidateSID(sid) {
		return "", "", 0, fmt.Errorf("invalid SID: %s", sid)
	}
	
	return password, sid, version, nil
}

// BuildCAPAB creates a CAPAB message
// Format: CAPAB :<capabilities>
func BuildCAPAB(capabilities []string) *Message {
	return &Message{
		Command: "CAPAB",
		Params:  []string{strings.Join(capabilities, " ")},
	}
}

// ParseCAPAB parses a CAPAB message
func ParseCAPAB(msg *Message) ([]string, error) {
	if len(msg.Params) < 1 {
		return nil, fmt.Errorf("CAPAB requires parameters")
	}
	
	capString := msg.Params[0]
	capabilities := strings.Fields(capString)
	
	return capabilities, nil
}

// BuildSERVER creates a SERVER message
// Format: SERVER <name> <hopcount> :<description>
func BuildSERVER(name string, hopcount int, description string) *Message {
	return &Message{
		Command: "SERVER",
		Params:  []string{name, strconv.Itoa(hopcount), description},
	}
}

// ParseSERVER parses a SERVER message
func ParseSERVER(msg *Message) (name string, hopcount int, description string, err error) {
	if len(msg.Params) < 3 {
		return "", 0, "", fmt.Errorf("SERVER requires 3 parameters")
	}
	
	name = msg.Params[0]
	
	hopcount, err = strconv.Atoi(msg.Params[1])
	if err != nil {
		return "", 0, "", fmt.Errorf("invalid hopcount: %s", msg.Params[1])
	}
	
	description = msg.Params[2]
	
	return name, hopcount, description, nil
}

// BuildSVINFO creates a SVINFO message
// Format: SVINFO <TS_version> <min_TS_version> <current_time>
func BuildSVINFO() *Message {
	return &Message{
		Command: "SVINFO",
		Params: []string{
			strconv.Itoa(TS6Version),
			strconv.Itoa(MinTSVersion),
			strconv.FormatInt(time.Now().Unix(), 10),
		},
	}
}

// ParseSVINFO parses a SVINFO message
func ParseSVINFO(msg *Message) (tsVersion, minVersion int, serverTime int64, err error) {
	if len(msg.Params) < 3 {
		return 0, 0, 0, fmt.Errorf("SVINFO requires 3 parameters")
	}
	
	tsVersion, err = strconv.Atoi(msg.Params[0])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid TS version: %s", msg.Params[0])
	}
	
	minVersion, err = strconv.Atoi(msg.Params[1])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid min TS version: %s", msg.Params[1])
	}
	
	serverTime, err = strconv.ParseInt(msg.Params[2], 10, 64)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid server time: %s", msg.Params[2])
	}
	
	return tsVersion, minVersion, serverTime, nil
}

// BuildUID creates a UID message to introduce a user
// Format: :<SID> UID <nick> <hopcount> <ts> <modes> <user> <host> <ip> <uid> :<realname>
func BuildUID(sid, nick, user, host, ip, uid, modes, realname string, timestamp int64) *Message {
	return &Message{
		Source:  sid,
		Command: "UID",
		Params: []string{
			nick,
			"1",                                  // hopcount
			strconv.FormatInt(timestamp, 10),     // timestamp
			modes,
			user,
			host,
			ip,
			uid,
			realname,
		},
	}
}

// ParseUID parses a UID message
func ParseUID(msg *Message) (*RemoteUser, error) {
	if len(msg.Params) < 9 {
		return nil, fmt.Errorf("UID requires 9 parameters")
	}
	
	timestamp, err := strconv.ParseInt(msg.Params[2], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid timestamp: %s", msg.Params[2])
	}
	
	uid := msg.Params[7]
	if !ValidateUID(uid) {
		return nil, fmt.Errorf("invalid UID: %s", uid)
	}
	
	user := &RemoteUser{
		UID:       uid,
		Nick:      msg.Params[0],
		User:      msg.Params[4],
		Host:      msg.Params[5],
		IP:        msg.Params[6],
		Modes:     msg.Params[3],
		RealName:  msg.Params[8],
		Timestamp: timestamp,
		Channels:  make(map[string]bool),
	}
	
	return user, nil
}

// BuildSJOIN creates a SJOIN message to introduce a channel
// Format: :<SID> SJOIN <ts> <channel> <modes> :<members>
// Members format: @UID1 +UID2 UID3 (@ = op, + = voice)
func BuildSJOIN(sid, channel string, ts int64, modes string, members map[string]string) *Message {
	// Build member string
	var memberList []string
	for uid, mode := range members {
		memberList = append(memberList, mode+uid)
	}
	membersStr := strings.Join(memberList, " ")
	
	return &Message{
		Source:  sid,
		Command: "SJOIN",
		Params: []string{
			strconv.FormatInt(ts, 10),
			channel,
			modes,
			membersStr,
		},
	}
}

// ParseSJOIN parses a SJOIN message
func ParseSJOIN(msg *Message) (channel string, ts int64, modes string, members map[string]string, err error) {
	if len(msg.Params) < 4 {
		return "", 0, "", nil, fmt.Errorf("SJOIN requires 4 parameters")
	}
	
	ts, err = strconv.ParseInt(msg.Params[0], 10, 64)
	if err != nil {
		return "", 0, "", nil, fmt.Errorf("invalid timestamp: %s", msg.Params[0])
	}
	
	channel = msg.Params[1]
	modes = msg.Params[2]
	
	// Parse members
	members = make(map[string]string)
	memberStr := msg.Params[3]
	for _, member := range strings.Fields(memberStr) {
		mode := ""
		uid := member
		
		// Extract mode prefix (@, +, etc)
		if len(member) > 0 && (member[0] == '@' || member[0] == '+') {
			mode = string(member[0])
			uid = member[1:]
		}
		
		if ValidateUID(uid) {
			members[uid] = mode
		}
	}
	
	return channel, ts, modes, members, nil
}

// BuildPING creates a PING message
func BuildPING(source, target string) *Message {
	return &Message{
		Source:  source,
		Command: "PING",
		Params:  []string{target},
	}
}

// BuildPONG creates a PONG message
func BuildPONG(source, target string) *Message {
	return &Message{
		Source:  source,
		Command: "PONG",
		Params:  []string{target},
	}
}

// BuildSQUIT creates a SQUIT message to signal server disconnect
// Format: :<source> SQUIT <server> :<reason>
func BuildSQUIT(source, server, reason string) *Message {
	return &Message{
		Source:  source,
		Command: "SQUIT",
		Params:  []string{server, reason},
	}
}

// ParseSQUIT parses a SQUIT message
func ParseSQUIT(msg *Message) (server, reason string, err error) {
	if len(msg.Params) < 2 {
		return "", "", fmt.Errorf("SQUIT requires 2 parameters")
	}
	
	return msg.Params[0], msg.Params[1], nil
}

// BuildERROR creates an ERROR message
func BuildERROR(reason string) *Message {
	return &Message{
		Command: "ERROR",
		Params:  []string{reason},
	}
}
