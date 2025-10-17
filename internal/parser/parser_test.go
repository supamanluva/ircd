package parser

import (
	"testing"
)

func TestParseSimpleCommand(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantCmd  string
		wantParams []string
	}{
		{
			name:     "NICK command",
			input:    "NICK alice",
			wantCmd:  "NICK",
			wantParams: []string{"alice"},
		},
		{
			name:     "JOIN command",
			input:    "JOIN #channel",
			wantCmd:  "JOIN",
			wantParams: []string{"#channel"},
		},
		{
			name:     "PRIVMSG with trailing",
			input:    "PRIVMSG #channel :Hello, world!",
			wantCmd:  "PRIVMSG",
			wantParams: []string{"#channel", "Hello, world!"},
		},
		{
			name:     "Command with prefix",
			input:    ":server 001 alice :Welcome",
			wantCmd:  "001",
			wantParams: []string{"alice", "Welcome"},
		},
		{
			name:     "PING",
			input:    "PING",
			wantCmd:  "PING",
			wantParams: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			if msg.Command != tt.wantCmd {
				t.Errorf("Command = %v, want %v", msg.Command, tt.wantCmd)
			}

			if len(msg.Params) != len(tt.wantParams) {
				t.Errorf("Params length = %v, want %v", len(msg.Params), len(tt.wantParams))
				return
			}

			for i, param := range msg.Params {
				if param != tt.wantParams[i] {
					t.Errorf("Params[%d] = %v, want %v", i, param, tt.wantParams[i])
				}
			}
		})
	}
}

func TestParseWithPrefix(t *testing.T) {
	input := ":nick!user@host PRIVMSG #channel :Hello"
	msg, err := Parse(input)
	
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if msg.Prefix != "nick!user@host" {
		t.Errorf("Prefix = %v, want %v", msg.Prefix, "nick!user@host")
	}

	if msg.Command != "PRIVMSG" {
		t.Errorf("Command = %v, want PRIVMSG", msg.Command)
	}
}

func TestIsValid(t *testing.T) {
	tests := []struct {
		name  string
		input string
		valid bool
	}{
		{"Valid command", "NICK alice", true},
		{"Empty string", "", false},
		{"Only spaces", "   ", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg, _ := Parse(tt.input)
			if msg.IsValid() != tt.valid {
				t.Errorf("IsValid() = %v, want %v", msg.IsValid(), tt.valid)
			}
		})
	}
}
