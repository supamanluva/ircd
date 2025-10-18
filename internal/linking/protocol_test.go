package linking

import (
	"testing"
	"time"
)

func TestParseMessage(t *testing.T) {
	tests := []struct {
		input   string
		wantMsg *Message
		wantErr bool
	}{
		{
			input: "PASS secret TS 6 0AA",
			wantMsg: &Message{
				Command: "PASS",
				Params:  []string{"secret", "TS", "6", "0AA"},
			},
		},
		{
			input: ":0AA SERVER hub.test 1 :Test Server",
			wantMsg: &Message{
				Source:  "0AA",
				Command: "SERVER",
				Params:  []string{"hub.test", "1", "Test Server"},
			},
		},
		{
			input: "CAPAB :QS EX CHW IE",
			wantMsg: &Message{
				Command: "CAPAB",
				Params:  []string{"QS EX CHW IE"},
			},
		},
		{
			input:   "",
			wantErr: true,
		},
		{
			input:   ":0AA",
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		msg, err := ParseMessage(tt.input)
		
		if tt.wantErr {
			if err == nil {
				t.Errorf("ParseMessage(%q) expected error, got nil", tt.input)
			}
			continue
		}
		
		if err != nil {
			t.Errorf("ParseMessage(%q) unexpected error: %v", tt.input, err)
			continue
		}
		
		if msg.Source != tt.wantMsg.Source {
			t.Errorf("ParseMessage(%q) source = %q, want %q", tt.input, msg.Source, tt.wantMsg.Source)
		}
		
		if msg.Command != tt.wantMsg.Command {
			t.Errorf("ParseMessage(%q) command = %q, want %q", tt.input, msg.Command, tt.wantMsg.Command)
		}
		
		if len(msg.Params) != len(tt.wantMsg.Params) {
			t.Errorf("ParseMessage(%q) param count = %d, want %d", tt.input, len(msg.Params), len(tt.wantMsg.Params))
			continue
		}
		
		for i, param := range msg.Params {
			if param != tt.wantMsg.Params[i] {
				t.Errorf("ParseMessage(%q) param[%d] = %q, want %q", tt.input, i, param, tt.wantMsg.Params[i])
			}
		}
	}
}

func TestMessageString(t *testing.T) {
	tests := []struct {
		msg  *Message
		want string
	}{
		{
			msg: &Message{
				Command: "PASS",
				Params:  []string{"secret", "TS", "6", "0AA"},
			},
			want: "PASS secret TS 6 0AA",
		},
		{
			msg: &Message{
				Source:  "0AA",
				Command: "SERVER",
				Params:  []string{"hub.test", "1", "Test Server"},
			},
			want: ":0AA SERVER hub.test 1 :Test Server",
		},
		{
			msg: &Message{
				Command: "CAPAB",
				Params:  []string{"QS EX CHW"},
			},
			want: "CAPAB :QS EX CHW",
		},
	}
	
	for _, tt := range tests {
		got := tt.msg.String()
		if got != tt.want {
			t.Errorf("Message.String() = %q, want %q", got, tt.want)
		}
	}
}

func TestBuildParsePASS(t *testing.T) {
	password := "secret123"
	sid := "0AA"
	
	msg := BuildPASS(password, sid)
	
	gotPass, gotSID, gotVersion, err := ParsePASS(msg)
	if err != nil {
		t.Fatalf("ParsePASS failed: %v", err)
	}
	
	if gotPass != password {
		t.Errorf("password = %q, want %q", gotPass, password)
	}
	
	if gotSID != sid {
		t.Errorf("SID = %q, want %q", gotSID, sid)
	}
	
	if gotVersion != TS6Version {
		t.Errorf("version = %d, want %d", gotVersion, TS6Version)
	}
}

func TestBuildParseCAPAB(t *testing.T) {
	caps := []string{"QS", "EX", "CHW", "IE"}
	
	msg := BuildCAPAB(caps)
	
	gotCaps, err := ParseCAPAB(msg)
	if err != nil {
		t.Fatalf("ParseCAPAB failed: %v", err)
	}
	
	if len(gotCaps) != len(caps) {
		t.Errorf("capability count = %d, want %d", len(gotCaps), len(caps))
	}
	
	for i, cap := range caps {
		if gotCaps[i] != cap {
			t.Errorf("capability[%d] = %q, want %q", i, gotCaps[i], cap)
		}
	}
}

func TestBuildParseSERVER(t *testing.T) {
	name := "hub.example.net"
	hopcount := 1
	description := "Test IRC Hub"
	
	msg := BuildSERVER(name, hopcount, description)
	
	gotName, gotHopcount, gotDesc, err := ParseSERVER(msg)
	if err != nil {
		t.Fatalf("ParseSERVER failed: %v", err)
	}
	
	if gotName != name {
		t.Errorf("name = %q, want %q", gotName, name)
	}
	
	if gotHopcount != hopcount {
		t.Errorf("hopcount = %d, want %d", gotHopcount, hopcount)
	}
	
	if gotDesc != description {
		t.Errorf("description = %q, want %q", gotDesc, description)
	}
}

func TestBuildParseSVINFO(t *testing.T) {
	msg := BuildSVINFO()
	
	gotTSVersion, gotMinVersion, gotTime, err := ParseSVINFO(msg)
	if err != nil {
		t.Fatalf("ParseSVINFO failed: %v", err)
	}
	
	if gotTSVersion != TS6Version {
		t.Errorf("TS version = %d, want %d", gotTSVersion, TS6Version)
	}
	
	if gotMinVersion != MinTSVersion {
		t.Errorf("min TS version = %d, want %d", gotMinVersion, MinTSVersion)
	}
	
	// Check time is reasonable (within last 5 seconds)
	now := time.Now().Unix()
	if gotTime < now-5 || gotTime > now+5 {
		t.Errorf("server time = %d, outside reasonable range of %d", gotTime, now)
	}
}

func TestBuildParseUID(t *testing.T) {
	sid := "0AA"
	nick := "TestUser"
	user := "testuser"
	host := "example.com"
	ip := "192.168.1.1"
	uid := "0AAAAAAAA"
	modes := "+i"
	realname := "Test User"
	timestamp := time.Now().Unix()
	
	msg := BuildUID(sid, nick, user, host, ip, uid, modes, realname, timestamp)
	
	gotUser, err := ParseUID(msg)
	if err != nil {
		t.Fatalf("ParseUID failed: %v", err)
	}
	
	if gotUser.UID != uid {
		t.Errorf("UID = %q, want %q", gotUser.UID, uid)
	}
	
	if gotUser.Nick != nick {
		t.Errorf("nick = %q, want %q", gotUser.Nick, nick)
	}
	
	if gotUser.User != user {
		t.Errorf("user = %q, want %q", gotUser.User, user)
	}
	
	if gotUser.Host != host {
		t.Errorf("host = %q, want %q", gotUser.Host, host)
	}
	
	if gotUser.IP != ip {
		t.Errorf("IP = %q, want %q", gotUser.IP, ip)
	}
	
	if gotUser.Modes != modes {
		t.Errorf("modes = %q, want %q", gotUser.Modes, modes)
	}
	
	if gotUser.RealName != realname {
		t.Errorf("realname = %q, want %q", gotUser.RealName, realname)
	}
	
	if gotUser.Timestamp != timestamp {
		t.Errorf("timestamp = %d, want %d", gotUser.Timestamp, timestamp)
	}
}

func TestBuildParseSJOIN(t *testing.T) {
	sid := "0AA"
	channel := "#test"
	ts := time.Now().Unix()
	modes := "+nt"
	members := map[string]string{
		"0AAAAAAAA": "@",
		"0AAAAAAAB": "+",
		"0AAAAAAAC": "",
	}
	
	msg := BuildSJOIN(sid, channel, ts, modes, members)
	
	gotChannel, gotTS, gotModes, gotMembers, err := ParseSJOIN(msg)
	if err != nil {
		t.Fatalf("ParseSJOIN failed: %v", err)
	}
	
	if gotChannel != channel {
		t.Errorf("channel = %q, want %q", gotChannel, channel)
	}
	
	if gotTS != ts {
		t.Errorf("timestamp = %d, want %d", gotTS, ts)
	}
	
	if gotModes != modes {
		t.Errorf("modes = %q, want %q", gotModes, modes)
	}
	
	if len(gotMembers) != len(members) {
		t.Errorf("member count = %d, want %d", len(gotMembers), len(members))
	}
	
	for uid, mode := range members {
		gotMode, ok := gotMembers[uid]
		if !ok {
			t.Errorf("member %q not found", uid)
			continue
		}
		if gotMode != mode {
			t.Errorf("member %q mode = %q, want %q", uid, gotMode, mode)
		}
	}
}

func TestBuildParseSQUIT(t *testing.T) {
	source := "0AA"
	server := "1BB"
	reason := "Connection lost"
	
	msg := BuildSQUIT(source, server, reason)
	
	gotServer, gotReason, err := ParseSQUIT(msg)
	if err != nil {
		t.Fatalf("ParseSQUIT failed: %v", err)
	}
	
	if gotServer != server {
		t.Errorf("server = %q, want %q", gotServer, server)
	}
	
	if gotReason != reason {
		t.Errorf("reason = %q, want %q", gotReason, reason)
	}
}

func TestParsePASSInvalid(t *testing.T) {
	tests := []struct {
		name string
		msg  *Message
	}{
		{
			name: "too few params",
			msg: &Message{
				Command: "PASS",
				Params:  []string{"secret"},
			},
		},
		{
			name: "invalid protocol",
			msg: &Message{
				Command: "PASS",
				Params:  []string{"secret", "IRC", "6", "0AA"},
			},
		},
		{
			name: "invalid version",
			msg: &Message{
				Command: "PASS",
				Params:  []string{"secret", "TS", "abc", "0AA"},
			},
		},
		{
			name: "invalid SID",
			msg: &Message{
				Command: "PASS",
				Params:  []string{"secret", "TS", "6", "AAA"},
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, _, err := ParsePASS(tt.msg)
			if err == nil {
				t.Errorf("ParsePASS expected error, got nil")
			}
		})
	}
}
