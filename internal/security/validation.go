package security

import (
	"strings"
)

// ValidateInput sanitizes and validates input strings
func ValidateInput(input string, maxLength int) (string, bool) {
	// Check length
	if len(input) > maxLength {
		return "", false
	}

	// Check for null bytes
	if strings.Contains(input, "\x00") {
		return "", false
	}

	return input, true
}

// SanitizeNickname ensures nickname only contains valid characters
func SanitizeNickname(nick string) string {
	var result strings.Builder
	for _, ch := range nick {
		// Allow ASCII letters, digits, and special IRC chars (RFC 2812)
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') ||
		   (ch >= '0' && ch <= '9') || 
		   strings.ContainsRune("[]\\`_^{|}-", ch) {
			result.WriteRune(ch)
		}
	}
	return result.String()
}

// SanitizeChannelName ensures channel name only contains valid characters
func SanitizeChannelName(name string) string {
	if len(name) == 0 {
		return ""
	}
	
	// First char must be # or &
	if name[0] != '#' && name[0] != '&' {
		return ""
	}
	
	var result strings.Builder
	result.WriteByte(name[0])
	
	// Rest: no spaces, commas, or control chars
	for i := 1; i < len(name); i++ {
		ch := name[i]
		if ch != ' ' && ch != ',' && ch >= 32 && ch != 7 {
			result.WriteByte(ch)
		}
	}
	return result.String()
}

// IsValidMessage checks if a message contains only valid characters
func IsValidMessage(msg string) bool {
	for _, ch := range msg {
		// Reject control characters except tab and newline
		if ch < 32 && ch != '\t' && ch != '\n' && ch != '\r' {
			return false
		}
	}
	return true
}

// StripControlCodes removes IRC color codes and control characters
func StripControlCodes(msg string) string {
	var result strings.Builder
	
	for i := 0; i < len(msg); i++ {
		ch := msg[i]
		
		// Handle IRC color codes (^C = 3)
		if ch == 3 {
			i++ // Skip the color code marker
			// Skip foreground color digits (up to 2)
			for j := 0; j < 2 && i < len(msg) && msg[i] >= '0' && msg[i] <= '9'; j++ {
				i++
			}
			// Skip comma and background color digits (up to 2)
			if i < len(msg) && msg[i] == ',' {
				i++
				for j := 0; j < 2 && i < len(msg) && msg[i] >= '0' && msg[i] <= '9'; j++ {
					i++
				}
			}
			i-- // Adjust because loop will increment
			continue
		}
		
		// Skip other control codes (bold, reset, underline, etc.)
		if ch == 2 || ch == 15 || ch == 22 || ch == 29 || ch == 31 {
			continue
		}
		
		result.WriteByte(ch)
	}
	
	return result.String()
}

// TruncateString safely truncates a string to maxLength
func TruncateString(s string, maxLength int) string {
	if len(s) <= maxLength {
		return s
	}
	return s[:maxLength]
}
