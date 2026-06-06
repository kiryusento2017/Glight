package main

import (
	"testing"
)

func TestParseSessionID(t *testing.T) {
	tests := []struct {
		name string
		json string
		want string
	}{
		{"正常 hook JSON", `{"session_id":"abc-123"}`, "abc-123"},
		{"空 JSON", `{}`, ""},
		{"无 session_id 字段", `{"other":"x"}`, ""},
		{"坏 JSON", `not json`, ""},
	}
	for _, tt := range tests {
		if got := parseSessionID([]byte(tt.json)); got != tt.want {
			t.Errorf("%s: got=%q want=%q", tt.name, got, tt.want)
		}
	}
}

func TestSessionFileName(t *testing.T) {
	tests := []struct {
		sid  string
		want string
	}{
		{"abc-123", "agent-light-state-abc-123"},
		{"", "agent-light-state-default"},
		{"../../../etc/passwd", "agent-light-state-etcpasswd"},
		{"session_42!", "agent-light-state-session42"},
	}
	for _, tt := range tests {
		if got := sessionFileName(tt.sid); got != tt.want {
			t.Errorf("sessionFileName(%q)=%q want %q", tt.sid, got, tt.want)
		}
	}
}
