package core

import (
	"testing"
)

func TestNewCode_LengthAndAlphabet(t *testing.T) {
	const N = 1000
	for i := 0; i < N; i++ {
		id, err := NewCode(CodeLen)
		if err != nil {
			t.Fatalf("NewCode error: %v", err)
		}
		if len(id) != CodeLen {
			t.Fatalf("bad length: got %d want %d", len(id), CodeLen)
		}
		if !IsValidCode(id) {
			t.Fatalf("id contains invalid chars: %q", id)
		}
	}
}

func TestIsValidCode(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{"Abcdef_123", true},       // 10, в алфавите
		{"short", false},           // мало
		{"this_is_11ch", false},    // много (11)
		{"абвгдежзий", false},      // кириллица
		{"ABC-123_def", false},     // недопустимый '-'
	}
	for _, tt := range tests {
		if got := IsValidCode(tt.in); got != tt.want {
			t.Fatalf("IsValidCode(%q)=%v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestNewCode_UniquenessBestEffort(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping uniqueness test in short mode")
	}
	const N = 20000
	seen := make(map[string]struct{}, N)
	for i := 0; i < N; i++ {
		id, err := NewCode(CodeLen)
		if err != nil {
			t.Fatalf("NewCode error: %v", err)
		}
		if _, ok := seen[id]; ok {
			t.Fatalf("unexpected collision for %q at i=%d", id, i)
		}
		seen[id] = struct{}{}
	}
}
