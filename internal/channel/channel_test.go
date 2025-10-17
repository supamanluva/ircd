package channel

import (
	"net"
	"testing"
	"time"

	"github.com/supamanluva/ircd/internal/client"
	"github.com/supamanluva/ircd/internal/logger"
)

// Mock connection for testing
type mockConn struct{}

func (m *mockConn) Read(b []byte) (n int, err error)         { return 0, nil }
func (m *mockConn) Write(b []byte) (n int, err error)        { return len(b), nil }
func (m *mockConn) Close() error                             { return nil }
func (m *mockConn) LocalAddr() net.Addr                      { return &mockAddr{} }
func (m *mockConn) RemoteAddr() net.Addr                     { return &mockAddr{} }
func (m *mockConn) SetDeadline(t time.Time) error            { return nil }
func (m *mockConn) SetReadDeadline(t time.Time) error        { return nil }
func (m *mockConn) SetWriteDeadline(t time.Time) error       { return nil }

type mockAddr struct{}

func (m *mockAddr) Network() string { return "tcp" }
func (m *mockAddr) String() string  { return "127.0.0.1:12345" }

func createTestClient(nick string) *client.Client {
	log := logger.New()
	c := client.New(&mockConn{}, log)
	c.SetNickname(nick)
	c.SetUsername(nick, "Test User")
	c.SetRegistered(true)
	return c
}

func TestNewChannel(t *testing.T) {
	ch := New("#test")
	
	if ch.GetName() != "#test" {
		t.Errorf("Expected name #test, got %s", ch.GetName())
	}
	
	if ch.GetMemberCount() != 0 {
		t.Errorf("Expected 0 members, got %d", ch.GetMemberCount())
	}
	
	if ch.GetTopic() != "" {
		t.Errorf("Expected empty topic, got %s", ch.GetTopic())
	}
	
	// Check default modes
	if !ch.HasMode('n') {
		t.Error("Expected +n (no external) mode by default")
	}
	
	if !ch.HasMode('t') {
		t.Error("Expected +t (topic lock) mode by default")
	}
}

func TestAddMember(t *testing.T) {
	ch := New("#test")
	c1 := createTestClient("alice")
	
	ch.AddMember(c1)
	
	if ch.GetMemberCount() != 1 {
		t.Errorf("Expected 1 member, got %d", ch.GetMemberCount())
	}
	
	if !ch.HasMember(c1) {
		t.Error("Expected alice to be a member")
	}
	
	// First member should be operator
	if !ch.IsOperator(c1) {
		t.Error("Expected first member to be operator")
	}
}

func TestRemoveMember(t *testing.T) {
	ch := New("#test")
	c1 := createTestClient("alice")
	
	ch.AddMember(c1)
	ch.RemoveMember(c1)
	
	if ch.GetMemberCount() != 0 {
		t.Errorf("Expected 0 members, got %d", ch.GetMemberCount())
	}
	
	if ch.HasMember(c1) {
		t.Error("Expected alice to not be a member")
	}
	
	if ch.IsOperator(c1) {
		t.Error("Expected alice to not be operator")
	}
}

func TestMultipleMembers(t *testing.T) {
	ch := New("#test")
	c1 := createTestClient("alice")
	c2 := createTestClient("bob")
	c3 := createTestClient("charlie")
	
	ch.AddMember(c1)
	ch.AddMember(c2)
	ch.AddMember(c3)
	
	if ch.GetMemberCount() != 3 {
		t.Errorf("Expected 3 members, got %d", ch.GetMemberCount())
	}
	
	// Only first member should be operator
	if !ch.IsOperator(c1) {
		t.Error("Expected alice to be operator")
	}
	
	if ch.IsOperator(c2) {
		t.Error("Expected bob to not be operator")
	}
	
	if ch.IsOperator(c3) {
		t.Error("Expected charlie to not be operator")
	}
}

func TestSetOperator(t *testing.T) {
	ch := New("#test")
	c1 := createTestClient("alice")
	c2 := createTestClient("bob")
	
	ch.AddMember(c1)
	ch.AddMember(c2)
	
	// Give bob operator status
	ch.SetOperator(c2, true)
	
	if !ch.IsOperator(c2) {
		t.Error("Expected bob to be operator after SetOperator(true)")
	}
	
	// Remove alice's operator status
	ch.SetOperator(c1, false)
	
	if ch.IsOperator(c1) {
		t.Error("Expected alice to not be operator after SetOperator(false)")
	}
}

func TestTopic(t *testing.T) {
	ch := New("#test")
	
	if ch.GetTopic() != "" {
		t.Error("Expected empty initial topic")
	}
	
	ch.SetTopic("Welcome to the test channel!")
	
	if ch.GetTopic() != "Welcome to the test channel!" {
		t.Errorf("Expected topic to be set, got %s", ch.GetTopic())
	}
}

func TestChannelModes(t *testing.T) {
	ch := New("#test")
	
	// Test setting modes
	ch.SetMode('i', true)
	if !ch.HasMode('i') {
		t.Error("Expected +i mode to be set")
	}
	
	ch.SetMode('m', true)
	if !ch.HasMode('m') {
		t.Error("Expected +m mode to be set")
	}
	
	// Test unsetting modes
	ch.SetMode('i', false)
	if ch.HasMode('i') {
		t.Error("Expected +i mode to be unset")
	}
	
	// Test GetModes
	modes := ch.GetModes()
	if modes != "+mnt" && modes != "+ntm" && modes != "+tmn" {
		t.Errorf("Expected modes to contain m, n, t in some order, got %s", modes)
	}
}

func TestBanList(t *testing.T) {
	ch := New("#test")
	
	// Test adding bans
	ch.AddBan("*!*@evil.com")
	ch.AddBan("baduser!*@*")
	
	bans := ch.GetBanList()
	if len(bans) != 2 {
		t.Errorf("Expected 2 bans, got %d", len(bans))
	}
	
	// Test duplicate ban
	ch.AddBan("*!*@evil.com")
	bans = ch.GetBanList()
	if len(bans) != 2 {
		t.Errorf("Expected 2 bans (no duplicate), got %d", len(bans))
	}
	
	// Test removing ban
	removed := ch.RemoveBan("*!*@evil.com")
	if !removed {
		t.Error("Expected ban to be removed")
	}
	
	bans = ch.GetBanList()
	if len(bans) != 1 {
		t.Errorf("Expected 1 ban after removal, got %d", len(bans))
	}
	
	// Test removing non-existent ban
	removed = ch.RemoveBan("nonexistent!*@*")
	if removed {
		t.Error("Expected false when removing non-existent ban")
	}
}

func TestGetMemberByNick(t *testing.T) {
	ch := New("#test")
	c1 := createTestClient("alice")
	c2 := createTestClient("bob")
	
	ch.AddMember(c1)
	ch.AddMember(c2)
	
	// Test getting existing member
	member := ch.GetMemberByNick("alice")
	if member == nil {
		t.Error("Expected to find alice")
	}
	if member != nil && member.GetNickname() != "alice" {
		t.Errorf("Expected alice, got %s", member.GetNickname())
	}
	
	// Test getting non-existent member
	member = ch.GetMemberByNick("charlie")
	if member != nil {
		t.Error("Expected nil for non-existent member")
	}
}

func TestGetMemberNicks(t *testing.T) {
	ch := New("#test")
	c1 := createTestClient("alice")
	c2 := createTestClient("bob")
	
	ch.AddMember(c1)
	ch.AddMember(c2)
	
	nicks := ch.GetMemberNicks()
	if len(nicks) != 2 {
		t.Errorf("Expected 2 nicks, got %d", len(nicks))
	}
	
	// Check that alice has @ prefix (operator)
	hasAliceOp := false
	hasBob := false
	for _, nick := range nicks {
		if nick == "@alice" {
			hasAliceOp = true
		}
		if nick == "bob" {
			hasBob = true
		}
	}
	
	if !hasAliceOp {
		t.Error("Expected @alice in nick list")
	}
	if !hasBob {
		t.Error("Expected bob in nick list")
	}
}

func TestIsEmpty(t *testing.T) {
	ch := New("#test")
	
	if !ch.IsEmpty() {
		t.Error("Expected empty channel")
	}
	
	c1 := createTestClient("alice")
	ch.AddMember(c1)
	
	if ch.IsEmpty() {
		t.Error("Expected non-empty channel")
	}
	
	ch.RemoveMember(c1)
	
	if !ch.IsEmpty() {
		t.Error("Expected empty channel after removing member")
	}
}

func TestConcurrentAccess(t *testing.T) {
	ch := New("#test")
	done := make(chan bool)
	
	// Add members concurrently
	for i := 0; i < 10; i++ {
		go func(n int) {
			c := createTestClient("user" + string(rune('0'+n)))
			ch.AddMember(c)
			done <- true
		}(i)
	}
	
	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
	
	count := ch.GetMemberCount()
	if count != 10 {
		t.Errorf("Expected 10 members after concurrent adds, got %d", count)
	}
}
