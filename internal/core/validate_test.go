package core

import (
	"strings"
	"testing"
)

func TestValidateURL(t *testing.T) {
	makeTooLong := func() string {
		// построим URL длиннее MaxURLLen
		base := "https://example.com/"
		if len(base) >= MaxURLLen {
			t.Fatalf("test base longer than MaxURLLen")
		}
		return base + strings.Repeat("a", MaxURLLen-len(base)+1)
	}

	tests := []struct {
		name    string
		in      string
		wantOK  bool
	}{
		{"https_ok", "https://example.com", true},
		{"http_with_path_and_query", "http://a.b/c?x=1&y=2", true},
		{"trim_spaces_ok", "   https://golang.org  ", true},

		{"empty", "", false},
		{"spaces_only", "   ", false},
		{"ftp_scheme", "ftp://example.com", false},
		{"no_host", "https://", false},
		{"too_long", makeTooLong(), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValidateURL(tt.in)
			if tt.wantOK {
				if err != nil {
					t.Fatalf("expected OK, got err: %v", err)
				}
				// быстрая sanity-проверка: результат не пустой и начинается с http/https
				if got == "" || !(strings.HasPrefix(got, "http://") || strings.HasPrefix(got, "https://")) {
					t.Fatalf("unexpected normalized URL: %q", got)
				}
			} else {
				if err == nil {
					t.Fatalf("expected error, got OK: %q", got)
				}
			}
		})
	}
}
