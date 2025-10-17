package security

import (
	"strings"
	"testing"
)

func TestValidateInput(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		maxLength int
		wantValid bool
	}{
		{"Valid input", "hello world", 20, true},
		{"Too long", "verylonginputstring", 10, false},
		{"Null byte", "hello\x00world", 20, false},
		{"Empty", "", 10, true},
		{"At limit", "exactly10!", 10, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, valid := ValidateInput(tt.input, tt.maxLength)
			if valid != tt.wantValid {
				t.Errorf("ValidateInput() valid = %v, want %v", valid, tt.wantValid)
			}
		})
	}
}

func TestSanitizeNickname(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"alice", "alice"},
		{"alice123", "alice123"},
		{"alice!@#", "alice"},
		{"alice_bot", "alice_bot"},
		{"alice[home]", "alice[home]"},
		{"test space", "testspace"},
		{"тест", ""}, // Non-ASCII removed
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := SanitizeNickname(tt.input)
			if got != tt.want {
				t.Errorf("SanitizeNickname(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSanitizeChannelName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"#general", "#general"},
		{"#test channel", "#testchannel"},
		{"#no,commas", "#nocommas"},
		{"&local", "&local"},
		{"#café", "#café"}, // Non-control chars allowed
		{"notchannel", ""},  // Must start with # or &
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := SanitizeChannelName(tt.input)
			if got != tt.want {
				t.Errorf("SanitizeChannelName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsValidMessage(t *testing.T) {
	tests := []struct {
		name  string
		input string
		valid bool
	}{
		{"Normal message", "Hello, world!", true},
		{"With tab", "Hello\tworld", true},
		{"With newline", "Hello\nworld", true},
		{"Control char", "Hello\x01world", false},
		{"Bell char", "Hello\x07world", false},
		{"Empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidMessage(tt.input); got != tt.valid {
				t.Errorf("IsValidMessage() = %v, want %v", got, tt.valid)
			}
		})
	}
}

func TestStripControlCodes(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"No codes", "plain text", "plain text"},
		{"Bold", "text\x02bold\x02normal", "textboldnormal"},
		{"Color removed", "text\x0312,04colored\x03normal", "textcolorednormal"},
		{"Multiple codes", "\x02bold\x1funderline\x02", "boldunderline"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StripControlCodes(tt.input)
			if got != tt.want {
				t.Errorf("StripControlCodes() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		input     string
		maxLength int
		want      string
	}{
		{"short", 10, "short"},
		{"exactly10!", 10, "exactly10!"},
		{"toolongstring", 7, "toolong"},
		{"", 5, ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := TruncateString(tt.input, tt.maxLength)
			if got != tt.want {
				t.Errorf("TruncateString() = %q, want %q", got, tt.want)
			}
			if len(got) > tt.maxLength {
				t.Errorf("TruncateString() length = %d, want <= %d", len(got), tt.maxLength)
			}
		})
	}
}

func BenchmarkSanitizeNickname(b *testing.B) {
	input := "alice!@#$%^&*()_+123"
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		SanitizeNickname(input)
	}
}

func BenchmarkStripControlCodes(b *testing.B) {
	input := strings.Repeat("\x02bold\x03colored\x0fnormal ", 100)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		StripControlCodes(input)
	}
}
