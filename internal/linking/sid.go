package linking

import (
	"crypto/rand"
	"fmt"
)

// GenerateSID generates a random Server ID in TS6 format: [0-9][A-Z0-9][A-Z0-9]
// Examples: 0AA, 1BC, 9ZZ
func GenerateSID() (string, error) {
	const (
		firstChars  = "0123456789"          // First char: digit
		otherChars  = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ" // Other chars: alphanumeric
	)
	
	// Generate random bytes
	bytes := make([]byte, 3)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate SID: %v", err)
	}
	
	// First char: 0-9
	sid := make([]byte, 3)
	sid[0] = firstChars[int(bytes[0])%len(firstChars)]
	
	// Second and third chars: 0-9A-Z
	sid[1] = otherChars[int(bytes[1])%len(otherChars)]
	sid[2] = otherChars[int(bytes[2])%len(otherChars)]
	
	return string(sid), nil
}

// GenerateSpecificSID generates a SID from specific values (for testing)
func GenerateSpecificSID(first int, second, third int) string {
	const chars = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	
	if first < 0 || first > 9 {
		first = 0
	}
	if second < 0 || second >= len(chars) {
		second = 0
	}
	if third < 0 || third >= len(chars) {
		third = 0
	}
	
	return fmt.Sprintf("%c%c%c", 
		'0'+first,
		chars[second],
		chars[third])
}
