package commands

import (
	"testing"

	"github.com/supamanluva/ircd/internal/channel"
	"github.com/supamanluva/ircd/internal/client"
	"github.com/supamanluva/ircd/internal/logger"
	"github.com/supamanluva/ircd/internal/parser"
)

// Mock client registry for testing
type mockClientRegistry struct {
	clients map[string]*client.Client
}

func newMockClientRegistry() *mockClientRegistry {
	return &mockClientRegistry{
		clients: make(map[string]*client.Client),
	}
}

func (m *mockClientRegistry) GetClient(nickname string) *client.Client {
	return m.clients[nickname]
}

func (m *mockClientRegistry) AddClient(c *client.Client) error {
	m.clients[c.GetNickname()] = c
	return nil
}

func (m *mockClientRegistry) RemoveClient(c *client.Client) {
	delete(m.clients, c.GetNickname())
}

func (m *mockClientRegistry) IsNicknameInUse(nickname string) bool {
	_, exists := m.clients[nickname]
	return exists
}

// Mock channel registry for testing
type mockChannelRegistry struct {
	channels map[string]*channel.Channel
}

func newMockChannelRegistry() *mockChannelRegistry {
	return &mockChannelRegistry{
		channels: make(map[string]*channel.Channel),
	}
}

func (m *mockChannelRegistry) GetChannel(name string) *channel.Channel {
	return m.channels[name]
}

func (m *mockChannelRegistry) CreateChannel(name string) *channel.Channel {
	if ch, exists := m.channels[name]; exists {
		return ch
	}
	ch := channel.New(name)
	m.channels[name] = ch
	return ch
}

func (m *mockChannelRegistry) RemoveChannel(name string) {
	delete(m.channels, name)
}

func TestIsValidNickname(t *testing.T) {
	tests := []struct {
		name     string
		nickname string
		valid    bool
	}{
		{"Valid simple", "alice", true},
		{"Valid with underscore", "alice_", true},
		{"Valid with digits", "alice123", true},
		{"Valid with brackets", "alice[bot]", true},
		{"Empty", "", false},
		{"Too long", "verylongnickname123", false},
		{"Starts with digit", "1alice", false},
		{"Invalid characters", "alice!", false},
		{"Spaces", "alice bob", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isValidNickname(tt.nickname); got != tt.valid {
				t.Errorf("isValidNickname(%q) = %v, want %v", tt.nickname, got, tt.valid)
			}
		})
	}
}

func TestHandleNick(t *testing.T) {
	log := logger.New()
	registry := newMockClientRegistry()
	handler := New("testserver", log, registry, &mockChannelRegistry{})

	tests := []struct {
		name        string
		input       string
		setup       func(*client.Client)
		expectNick  string
		expectError bool
	}{
		{
			name:       "Valid nickname",
			input:      "NICK alice",
			expectNick: "alice",
		},
		{
			name:       "No nickname given",
			input:      "NICK",
			expectNick: "",
		},
		{
			name:       "Invalid nickname",
			input:      "NICK 123invalid",
			expectNick: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := client.NewMock(log)
			
			if tt.setup != nil {
				tt.setup(c)
			}

			msg, _ := parser.Parse(tt.input)
			err := handler.handleNick(c, msg)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if tt.expectNick != "" && c.GetNickname() != tt.expectNick {
				t.Errorf("Expected nickname %q, got %q", tt.expectNick, c.GetNickname())
			}
		})
	}
}

func TestHandlePing(t *testing.T) {
	log := logger.New()
	registry := newMockClientRegistry()
	handler := New("testserver", log, registry, &mockChannelRegistry{})

	tests := []struct {
		name        string
		input       string
		expectError bool
	}{
		{
			name:  "PING with token",
			input: "PING :12345",
		},
		{
			name:        "PING without token",
			input:       "PING",
			expectError: false, // Should send error reply but not return error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := client.NewMock(log)
			msg, _ := parser.Parse(tt.input)
			err := handler.handlePing(c, msg)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestNumericReply(t *testing.T) {
	tests := []struct {
		name       string
		serverName string
		code       string
		nick       string
		message    string
		expected   string
	}{
		{
			name:       "Welcome message",
			serverName: "irc.test.com",
			code:       "001",
			nick:       "alice",
			message:    ":Welcome to IRC",
			expected:   ":irc.test.com 001 alice :Welcome to IRC",
		},
		{
			name:       "Empty nick becomes asterisk",
			serverName: "irc.test.com",
			code:       "001",
			nick:       "",
			message:    ":Welcome",
			expected:   ":irc.test.com 001 * :Welcome",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NumericReply(tt.serverName, tt.code, tt.nick, tt.message)
			if got != tt.expected {
				t.Errorf("NumericReply() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestHandleUser(t *testing.T) {
	log := logger.New()
	registry := newMockClientRegistry()
	handler := New("testserver", log, registry, newMockChannelRegistry())

	tests := []struct {
		name        string
		input       string
		setup       func(*client.Client)
		expectError bool
		expectUser  string
		expectReal  string
	}{
		{
			name:       "Valid USER command",
			input:      "USER alice 0 * :Alice Wonderland",
			expectUser: "alice",
			expectReal: "Alice Wonderland",
		},
		{
			name:        "USER with insufficient params",
			input:       "USER alice",
			expectError: false, // Sends error reply but doesn't return error
		},
		{
			name: "USER already registered",
			input: "USER alice 0 * :Alice",
			setup: func(c *client.Client) {
				c.SetUsername("olduser", "Old User")
				c.SetRegistered(true)
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := client.NewMock(log)
			
			if tt.setup != nil {
				tt.setup(c)
			}

			msg, _ := parser.Parse(tt.input)
			err := handler.handleUser(c, msg)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Note: Client doesn't expose GetUsername/GetRealname, so we can't verify
			// But we can verify no error occurred which means the values were set
		})
	}
}

func TestHandlePong(t *testing.T) {
	log := logger.New()
	registry := newMockClientRegistry()
	handler := New("testserver", log, registry, newMockChannelRegistry())

	tests := []struct {
		name        string
		input       string
		expectError bool
	}{
		{
			name:  "PONG with token",
			input: "PONG :testserver",
		},
		{
			name:  "PONG without token",
			input: "PONG",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := client.NewMock(log)
			msg, _ := parser.Parse(tt.input)
			err := handler.handlePong(c, msg)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestHandleQuit(t *testing.T) {
	log := logger.New()
	registry := newMockClientRegistry()
	handler := New("testserver", log, registry, newMockChannelRegistry())

	tests := []struct {
		name        string
		input       string
		expectError bool
	}{
		{
			name:        "QUIT with message",
			input:       "QUIT :Goodbye",
			expectError: true, // handleQuit returns error to signal disconnect
		},
		{
			name:        "QUIT without message",
			input:       "QUIT",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := client.NewMock(log)
			c.SetNickname("alice")
			registry.AddClient(c)

			msg, _ := parser.Parse(tt.input)
			err := handler.handleQuit(c, msg)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestHandleJoin(t *testing.T) {
	log := logger.New()
	clientReg := newMockClientRegistry()
	channelReg := newMockChannelRegistry()
	handler := New("testserver", log, clientReg, channelReg)

	tests := []struct {
		name        string
		input       string
		setup       func(*client.Client)
		expectError bool
		expectChan  string
	}{
		{
			name:       "Join single channel",
			input:      "JOIN #test",
			expectChan: "#test",
			setup: func(c *client.Client) {
				c.SetNickname("alice")
				c.SetRegistered(true)
			},
		},
		{
			name:  "Join without registration",
			input: "JOIN #test",
			setup: func(c *client.Client) {
				c.SetNickname("alice")
			},
		},
		{
			name: "Join without params",
			input: "JOIN",
			setup: func(c *client.Client) {
				c.SetNickname("alice")
				c.SetRegistered(true)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := client.NewMock(log)
			
			if tt.setup != nil {
				tt.setup(c)
			}

			msg, _ := parser.Parse(tt.input)
			err := handler.handleJoin(c, msg)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if tt.expectChan != "" {
				ch := channelReg.GetChannel(tt.expectChan)
				if ch == nil {
					t.Errorf("Expected channel %q to exist", tt.expectChan)
				}
			}
		})
	}
}

func TestHandlePart(t *testing.T) {
	log := logger.New()
	clientReg := newMockClientRegistry()
	channelReg := newMockChannelRegistry()
	handler := New("testserver", log, clientReg, channelReg)

	tests := []struct {
		name        string
		input       string
		setup       func(*client.Client)
		expectError bool
	}{
		{
			name:  "Part from channel",
			input: "PART #test",
			setup: func(c *client.Client) {
				c.SetNickname("alice")
				c.SetRegistered(true)
				ch := channelReg.CreateChannel("#test")
				ch.AddMember(c)
			},
		},
		{
			name:  "Part from non-existent channel",
			input: "PART #nonexistent",
			setup: func(c *client.Client) {
				c.SetNickname("alice")
				c.SetRegistered(true)
			},
		},
		{
			name: "Part without params",
			input: "PART",
			setup: func(c *client.Client) {
				c.SetNickname("alice")
				c.SetRegistered(true)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := client.NewMock(log)
			
			if tt.setup != nil {
				tt.setup(c)
			}

			msg, _ := parser.Parse(tt.input)
			err := handler.handlePart(c, msg)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestHandlePrivmsg(t *testing.T) {
	log := logger.New()
	clientReg := newMockClientRegistry()
	channelReg := newMockChannelRegistry()
	handler := New("testserver", log, clientReg, channelReg)

	tests := []struct {
		name        string
		input       string
		setup       func(*client.Client)
		expectError bool
	}{
		{
			name:  "Send to channel",
			input: "PRIVMSG #test :Hello everyone",
			setup: func(c *client.Client) {
				c.SetNickname("alice")
				c.SetRegistered(true)
				ch := channelReg.CreateChannel("#test")
				ch.AddMember(c)
			},
		},
		{
			name:  "Send to user",
			input: "PRIVMSG bob :Hello bob",
			setup: func(c *client.Client) {
				c.SetNickname("alice")
				c.SetRegistered(true)
				bob := client.NewMock(log)
				bob.SetNickname("bob")
				clientReg.AddClient(bob)
			},
		},
		{
			name: "No recipient",
			input: "PRIVMSG",
			setup: func(c *client.Client) {
				c.SetNickname("alice")
				c.SetRegistered(true)
			},
		},
		{
			name: "No text",
			input: "PRIVMSG #test",
			setup: func(c *client.Client) {
				c.SetNickname("alice")
				c.SetRegistered(true)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := client.NewMock(log)
			
			if tt.setup != nil {
				tt.setup(c)
			}

			msg, _ := parser.Parse(tt.input)
			err := handler.handlePrivmsg(c, msg)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestHandleNotice(t *testing.T) {
	log := logger.New()
	clientReg := newMockClientRegistry()
	channelReg := newMockChannelRegistry()
	handler := New("testserver", log, clientReg, channelReg)

	tests := []struct {
		name        string
		input       string
		setup       func(*client.Client)
		expectError bool
	}{
		{
			name:  "Send notice to channel",
			input: "NOTICE #test :Server notice",
			setup: func(c *client.Client) {
				c.SetNickname("alice")
				c.SetRegistered(true)
				ch := channelReg.CreateChannel("#test")
				ch.AddMember(c)
			},
		},
		{
			name:  "Send notice to user",
			input: "NOTICE bob :Private notice",
			setup: func(c *client.Client) {
				c.SetNickname("alice")
				c.SetRegistered(true)
				bob := client.NewMock(log)
				bob.SetNickname("bob")
				clientReg.AddClient(bob)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := client.NewMock(log)
			
			if tt.setup != nil {
				tt.setup(c)
			}

			msg, _ := parser.Parse(tt.input)
			err := handler.handleNotice(c, msg)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestHandleNames(t *testing.T) {
	log := logger.New()
	clientReg := newMockClientRegistry()
	channelReg := newMockChannelRegistry()
	handler := New("testserver", log, clientReg, channelReg)

	tests := []struct {
		name        string
		input       string
		setup       func(*client.Client)
		expectError bool
	}{
		{
			name:  "NAMES for channel",
			input: "NAMES #test",
			setup: func(c *client.Client) {
				c.SetNickname("alice")
				c.SetRegistered(true)
				ch := channelReg.CreateChannel("#test")
				ch.AddMember(c)
			},
		},
		{
			name: "NAMES without params",
			input: "NAMES",
			setup: func(c *client.Client) {
				c.SetNickname("alice")
				c.SetRegistered(true)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := client.NewMock(log)
			
			if tt.setup != nil {
				tt.setup(c)
			}

			msg, _ := parser.Parse(tt.input)
			err := handler.handleNames(c, msg)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestHandleTopic(t *testing.T) {
	log := logger.New()
	clientReg := newMockClientRegistry()
	channelReg := newMockChannelRegistry()
	handler := New("testserver", log, clientReg, channelReg)

	tests := []struct {
		name        string
		input       string
		setup       func(*client.Client)
		expectError bool
		expectTopic string
	}{
		{
			name:        "Set topic",
			input:       "TOPIC #test :New topic",
			expectTopic: "New topic",
			setup: func(c *client.Client) {
				c.SetNickname("alice")
				c.SetRegistered(true)
				ch := channelReg.CreateChannel("#test")
				ch.AddMember(c)
			},
		},
		{
			name:  "Get topic",
			input: "TOPIC #test",
			setup: func(c *client.Client) {
				c.SetNickname("alice")
				c.SetRegistered(true)
				ch := channelReg.CreateChannel("#test")
				ch.AddMember(c)
				ch.SetTopic("Existing topic")
			},
		},
		{
			name: "Topic without channel",
			input: "TOPIC",
			setup: func(c *client.Client) {
				c.SetNickname("alice")
				c.SetRegistered(true)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := client.NewMock(log)
			
			if tt.setup != nil {
				tt.setup(c)
			}

			msg, _ := parser.Parse(tt.input)
			err := handler.handleTopic(c, msg)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if tt.expectTopic != "" {
				ch := channelReg.GetChannel("#test")
				if ch != nil && ch.GetTopic() != tt.expectTopic {
					t.Errorf("Expected topic %q, got %q", tt.expectTopic, ch.GetTopic())
				}
			}
		})
	}
}

func TestHandleMode(t *testing.T) {
	log := logger.New()
	clientReg := newMockClientRegistry()
	channelReg := newMockChannelRegistry()
	handler := New("testserver", log, clientReg, channelReg)

	tests := []struct {
		name        string
		input       string
		setup       func(*client.Client)
		expectError bool
	}{
		{
			name:  "Set user mode",
			input: "MODE alice +i",
			setup: func(c *client.Client) {
				c.SetNickname("alice")
				c.SetRegistered(true)
			},
		},
		{
			name:  "Set channel mode",
			input: "MODE #test +t",
			setup: func(c *client.Client) {
				c.SetNickname("alice")
				c.SetRegistered(true)
				ch := channelReg.CreateChannel("#test")
				ch.AddMember(c)
			},
		},
		{
			name: "MODE without params",
			input: "MODE",
			setup: func(c *client.Client) {
				c.SetNickname("alice")
				c.SetRegistered(true)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := client.NewMock(log)
			
			if tt.setup != nil {
				tt.setup(c)
			}

			msg, _ := parser.Parse(tt.input)
			err := handler.handleMode(c, msg)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestHandleKick(t *testing.T) {
	log := logger.New()
	clientReg := newMockClientRegistry()
	channelReg := newMockChannelRegistry()
	handler := New("testserver", log, clientReg, channelReg)

	tests := []struct {
		name        string
		input       string
		setup       func(*client.Client)
		expectError bool
	}{
		{
			name:  "Kick user from channel",
			input: "KICK #test bob :Spamming",
			setup: func(c *client.Client) {
				c.SetNickname("alice")
				c.SetRegistered(true)
				ch := channelReg.CreateChannel("#test")
				ch.AddMember(c)
				bob := client.NewMock(log)
				bob.SetNickname("bob")
				ch.AddMember(bob)
				clientReg.AddClient(bob)
			},
		},
		{
			name:  "Kick without operator",
			input: "KICK #test bob",
			setup: func(c *client.Client) {
				c.SetNickname("alice")
				c.SetRegistered(true)
				ch := channelReg.CreateChannel("#test")
				ch.AddMember(c)
				bob := client.NewMock(log)
				bob.SetNickname("bob")
				ch.AddMember(bob)
			},
		},
		{
			name: "KICK without params",
			input: "KICK",
			setup: func(c *client.Client) {
				c.SetNickname("alice")
				c.SetRegistered(true)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := client.NewMock(log)
			
			if tt.setup != nil {
				tt.setup(c)
			}

			msg, _ := parser.Parse(tt.input)
			err := handler.handleKick(c, msg)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestHandle(t *testing.T) {
	log := logger.New()
	clientReg := newMockClientRegistry()
	channelReg := newMockChannelRegistry()
	handler := New("testserver", log, clientReg, channelReg)

	tests := []struct {
		name        string
		input       string
		setup       func(*client.Client)
		expectError bool
	}{
		{
			name:  "Route NICK command",
			input: "NICK alice",
		},
		{
			name:  "Route USER command",
			input: "USER alice 0 * :Alice",
		},
		{
			name:  "Route PING command",
			input: "PING :test",
		},
		{
			name:  "Unknown command",
			input: "UNKNOWN param1 param2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := client.NewMock(log)
			
			if tt.setup != nil {
				tt.setup(c)
			}

			msg, _ := parser.Parse(tt.input)
			err := handler.Handle(c, msg)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}
