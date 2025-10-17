package parser

import (
	"strings"
)

// Message represents a parsed IRC message
type Message struct {
	Prefix  string   // Optional prefix (sender)
	Command string   // IRC command (e.g., PRIVMSG, JOIN)
	Params  []string // Command parameters
	Raw     string   // Raw message string
}

// Parse parses a raw IRC protocol message
// Format: [:prefix] <command> [params] [:trailing]
func Parse(raw string) (*Message, error) {
	msg := &Message{
		Raw:    raw,
		Params: make([]string, 0),
	}

	// Handle empty message
	if raw == "" {
		return msg, nil
	}

	pos := 0
	
	// Parse prefix (optional, starts with :)
	if raw[0] == ':' {
		end := strings.Index(raw, " ")
		if end == -1 {
			// Malformed message
			return msg, nil
		}
		msg.Prefix = raw[1:end]
		pos = end + 1
	}

	// Skip spaces
	for pos < len(raw) && raw[pos] == ' ' {
		pos++
	}

	// Parse command
	end := strings.Index(raw[pos:], " ")
	if end == -1 {
		// Command with no parameters
		msg.Command = strings.ToUpper(raw[pos:])
		return msg, nil
	}
	
	msg.Command = strings.ToUpper(raw[pos : pos+end])
	pos += end + 1

	// Parse parameters
	for pos < len(raw) {
		// Skip spaces
		for pos < len(raw) && raw[pos] == ' ' {
			pos++
		}
		
		if pos >= len(raw) {
			break
		}

		// Trailing parameter (starts with :)
		if raw[pos] == ':' {
			msg.Params = append(msg.Params, raw[pos+1:])
			break
		}

		// Regular parameter
		end := strings.Index(raw[pos:], " ")
		if end == -1 {
			msg.Params = append(msg.Params, raw[pos:])
			break
		}
		
		msg.Params = append(msg.Params, raw[pos:pos+end])
		pos += end + 1
	}

	return msg, nil
}

// IsValid checks if the message has a valid command
func (m *Message) IsValid() bool {
	return m.Command != ""
}

// GetParam returns the parameter at the given index, or empty string if not found
func (m *Message) GetParam(index int) string {
	if index < 0 || index >= len(m.Params) {
		return ""
	}
	return m.Params[index]
}

// HasParam checks if a parameter exists at the given index
func (m *Message) HasParam(index int) bool {
	return index >= 0 && index < len(m.Params)
}
