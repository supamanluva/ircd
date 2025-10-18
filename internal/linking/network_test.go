package linking

import (
	"fmt"
	"testing"
)

func TestGenerateSID(t *testing.T) {
	// Test multiple SID generations
	for i := 0; i < 100; i++ {
		sid, err := GenerateSID()
		if err != nil {
			t.Fatalf("GenerateSID failed: %v", err)
		}
		
		if !ValidateSID(sid) {
			t.Errorf("Generated invalid SID: %s", sid)
		}
		
		// Check format
		if len(sid) != 3 {
			t.Errorf("SID %s has wrong length: %d", sid, len(sid))
		}
		
		// First char must be 0-9
		if sid[0] < '0' || sid[0] > '9' {
			t.Errorf("SID %s first char not a digit: %c", sid, sid[0])
		}
	}
}

func TestGenerateSpecificSID(t *testing.T) {
	tests := []struct {
		first, second, third int
		expected             string
	}{
		{0, 0, 0, "000"},
		{1, 10, 10, "1AA"},
		{9, 35, 35, "9ZZ"},
		{5, 15, 20, "5FK"},
	}
	
	for _, tt := range tests {
		sid := GenerateSpecificSID(tt.first, tt.second, tt.third)
		if sid != tt.expected {
			t.Errorf("GenerateSpecificSID(%d, %d, %d) = %s, want %s",
				tt.first, tt.second, tt.third, sid, tt.expected)
		}
		
		if !ValidateSID(sid) {
			t.Errorf("Generated invalid SID: %s", sid)
		}
	}
}

func TestValidateSID(t *testing.T) {
	tests := []struct {
		sid   string
		valid bool
	}{
		{"0AA", true},
		{"1BC", true},
		{"9ZZ", true},
		{"000", true},
		{"5K9", true},
		
		// Invalid
		{"AA0", false},  // First char not digit
		{"A00", false},  // First char not digit
		{"0aa", false},  // Lowercase not allowed
		{"00", false},   // Too short
		{"0000", false}, // Too long
		{"", false},     // Empty
		{"0A!", false},  // Invalid char
	}
	
	for _, tt := range tests {
		result := ValidateSID(tt.sid)
		if result != tt.valid {
			t.Errorf("ValidateSID(%q) = %v, want %v", tt.sid, result, tt.valid)
		}
	}
}

func TestValidateUID(t *testing.T) {
	tests := []struct {
		uid   string
		valid bool
	}{
		{"0AAAAAAAA", true},
		{"1BCAAAAAB", true},
		{"9ZZZZZZZZ", true},
		{"5K9000000", true},
		
		// Invalid
		{"0AA", false},        // Too short
		{"0AAAAAA", false},    // Too short
		{"0AAAAAAAAAA", false}, // Too long
		{"AA0AAAAAA", false},  // Invalid SID
		{"0AAaaaaaa", false},  // Lowercase not allowed
		{"0AA!AAAAA", false},  // Invalid char
		{"", false},           // Empty
	}
	
	for _, tt := range tests {
		result := ValidateUID(tt.uid)
		if result != tt.valid {
			t.Errorf("ValidateUID(%q) = %v, want %v", tt.uid, result, tt.valid)
		}
	}
}

func TestNetworkGenerateUID(t *testing.T) {
	net := NewNetwork("0AA", "test.server")
	
	// Generate multiple UIDs
	uids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		uid := net.GenerateUID()
		
		// Check format
		if !ValidateUID(uid) {
			t.Errorf("Generated invalid UID: %s", uid)
		}
		
		// Check SID prefix
		if uid[:3] != "0AA" {
			t.Errorf("UID %s has wrong SID prefix", uid)
		}
		
		// Check uniqueness
		if uids[uid] {
			t.Errorf("Duplicate UID generated: %s", uid)
		}
		uids[uid] = true
	}
}

func TestEncodeBase36(t *testing.T) {
	tests := []struct {
		n        uint32
		expected string
	}{
		{0, "000000"},
		{1, "000001"},
		{10, "00000A"},
		{35, "00000Z"},
		{36, "000010"},
		{1295, "0000ZZ"}, // 35*36 + 35
	}
	
	for _, tt := range tests {
		result := encodeBase36(tt.n)
		if result != tt.expected {
			t.Errorf("encodeBase36(%d) = %s, want %s", tt.n, result, tt.expected)
		}
	}
}

func TestNetworkAddServer(t *testing.T) {
	net := NewNetwork("0AA", "hub.test")
	
	srv := &Server{
		SID:  "1BB",
		Name: "leaf.test",
	}
	
	err := net.AddServer(srv)
	if err != nil {
		t.Fatalf("AddServer failed: %v", err)
	}
	
	// Try to add again (should fail)
	err = net.AddServer(srv)
	if err == nil {
		t.Error("AddServer should fail for duplicate SID")
	}
	
	// Check retrieval
	retrieved, ok := net.GetServer("1BB")
	if !ok {
		t.Error("GetServer failed to find server")
	}
	if retrieved.SID != srv.SID {
		t.Errorf("Retrieved server SID = %s, want %s", retrieved.SID, srv.SID)
	}
}

func TestNetworkAddUser(t *testing.T) {
	net := NewNetwork("0AA", "hub.test")
	
	srv := &Server{
		SID:  "1BB",
		Name: "leaf.test",
	}
	net.AddServer(srv)
	
	user := &RemoteUser{
		UID:       "1BBAAAAAA",
		Nick:      "TestUser",
		Server:    srv,
		Timestamp: 1234567890,
		Channels:  make(map[string]bool),
	}
	
	err := net.AddUser(user)
	if err != nil {
		t.Fatalf("AddUser failed: %v", err)
	}
	
	// Try to add duplicate UID
	err = net.AddUser(user)
	if err == nil {
		t.Error("AddUser should fail for duplicate UID")
	}
	
	// Check retrieval by UID
	retrieved, ok := net.GetUserByUID("1BBAAAAAA")
	if !ok {
		t.Error("GetUserByUID failed")
	}
	if retrieved.Nick != "TestUser" {
		t.Errorf("Retrieved user nick = %s, want TestUser", retrieved.Nick)
	}
	
	// Check retrieval by nick
	retrieved2, ok := net.GetUserByNick("TestUser")
	if !ok {
		t.Error("GetUserByNick failed")
	}
	if retrieved2.UID != "1BBAAAAAA" {
		t.Errorf("Retrieved user UID = %s, want 1BBAAAAAA", retrieved2.UID)
	}
}

func TestNetworkUpdateNick(t *testing.T) {
	net := NewNetwork("0AA", "hub.test")
	
	srv := &Server{
		SID:  "1BB",
		Name: "leaf.test",
	}
	net.AddServer(srv)
	
	user := &RemoteUser{
		UID:       "1BBAAAAAA",
		Nick:      "OldNick",
		Server:    srv,
		Timestamp: 1234567890,
		Channels:  make(map[string]bool),
	}
	net.AddUser(user)
	
	// Update nick
	err := net.UpdateNick("1BBAAAAAA", "NewNick", 1234567900)
	if err != nil {
		t.Fatalf("UpdateNick failed: %v", err)
	}
	
	// Old nick should not exist
	_, ok := net.GetUserByNick("OldNick")
	if ok {
		t.Error("Old nick still exists")
	}
	
	// New nick should exist
	retrieved, ok := net.GetUserByNick("NewNick")
	if !ok {
		t.Error("New nick not found")
	}
	if retrieved.UID != "1BBAAAAAA" {
		t.Errorf("Wrong user for new nick")
	}
}

func TestNetworkNickCollision(t *testing.T) {
	net := NewNetwork("0AA", "hub.test")
	
	srv1 := &Server{SID: "1BB", Name: "server1"}
	srv2 := &Server{SID: "2CC", Name: "server2"}
	net.AddServer(srv1)
	net.AddServer(srv2)
	
	// Add first user
	user1 := &RemoteUser{
		UID:       "1BBAAAAAA",
		Nick:      "SameNick",
		Server:    srv1,
		Timestamp: 1000, // Older timestamp
		Channels:  make(map[string]bool),
	}
	net.AddUser(user1)
	
	// Try to add second user with same nick but newer timestamp
	user2 := &RemoteUser{
		UID:       "2CCAAAAAA",
		Nick:      "SameNick",
		Server:    srv2,
		Timestamp: 2000, // Newer timestamp
		Channels:  make(map[string]bool),
	}
	err := net.AddUser(user2)
	if err == nil {
		t.Error("AddUser should fail for nick collision with newer timestamp")
	}
	
	// First user should still exist
	retrieved, ok := net.GetUserByNick("SameNick")
	if !ok {
		t.Fatal("Original user not found")
	}
	if retrieved.UID != "1BBAAAAAA" {
		t.Error("Wrong user retained after collision")
	}
}

func TestNetworkRemoveServer(t *testing.T) {
	net := NewNetwork("0AA", "hub.test")
	
	srv := &Server{
		SID:     "1BB",
		Name:    "leaf.test",
		Users:   make(map[string]*RemoteUser),
		Channels: make(map[string]*RemoteChannel),
	}
	net.AddServer(srv)
	
	// Add users to server
	for i := 0; i < 5; i++ {
		uid := net.GenerateUID()
		user := &RemoteUser{
			UID:       uid,
			Nick:      fmt.Sprintf("User%d", i),
			Server:    srv,
			Timestamp: int64(i),
			Channels:  make(map[string]bool),
		}
		net.AddUser(user)
	}
	
	initialUsers := net.GetUserCount()
	if initialUsers != 5 {
		t.Errorf("Expected 5 users, got %d", initialUsers)
	}
	
	// Remove server
	net.RemoveServer("1BB")
	
	// Server should be gone
	_, ok := net.GetServer("1BB")
	if ok {
		t.Error("Server still exists after removal")
	}
	
	// All users should be gone
	if net.GetUserCount() != 0 {
		t.Errorf("Expected 0 users after server removal, got %d", net.GetUserCount())
	}
}

func TestNetworkAddChannel(t *testing.T) {
	net := NewNetwork("0AA", "hub.test")
	
	ch := &RemoteChannel{
		Name:    "#test",
		TS:      1234567890,
		Modes:   "nt",
		Members: make(map[string]string),
	}
	
	net.AddChannel(ch)
	
	// Check retrieval
	retrieved, ok := net.GetChannel("#test")
	if !ok {
		t.Error("Channel not found")
	}
	if retrieved.Name != "#test" {
		t.Errorf("Wrong channel name: %s", retrieved.Name)
	}
}

func TestNetworkChannelTSConflict(t *testing.T) {
	net := NewNetwork("0AA", "hub.test")
	
	// Add channel with TS 1000
	ch1 := &RemoteChannel{
		Name:    "#test",
		TS:      1000,
		Modes:   "nt",
		Members: map[string]string{"user1": "@"},
	}
	net.AddChannel(ch1)
	
	// Try to add same channel with older TS (should win)
	ch2 := &RemoteChannel{
		Name:    "#test",
		TS:      500, // Older
		Modes:   "s",
		Members: map[string]string{"user2": "+"},
	}
	net.AddChannel(ch2)
	
	// Check that older TS won
	retrieved, _ := net.GetChannel("#test")
	if retrieved.TS != 500 {
		t.Errorf("Channel TS = %d, want 500", retrieved.TS)
	}
	if retrieved.Modes != "s" {
		t.Errorf("Channel modes = %s, want s", retrieved.Modes)
	}
}
