package main

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestNormalizeURL(t *testing.T) {
	tests := []struct {
		name string
		input string
		want string
	}{
		{
			name:  "bare domain gets https",
			input: "example.com",
			want:  "https://example.com",
		},
		{
			name:  "http scheme unchanged",
			input: "http://foo.com",
			want:  "http://foo.com",
		},
		{
			name:  "https scheme unchanged",
			input: "https://bar.com",
			want:  "https://bar.com",
		},
		{
			name:  "about:blank unchanged",
			input: "about:blank",
			want:  "about:blank",
		},
		{
			name:  "data URL unchanged",
			input: "data:text/html,<h1>hi</h1>",
			want:  "data:text/html,<h1>hi</h1>",
		},
		{
			name:  "javascript URL unchanged",
			input: "javascript:void(0)",
			want:  "javascript:void(0)",
		},
		{
			name:  "file URL unchanged",
			input: "file:///tmp/test.html",
			want:  "file:///tmp/test.html",
		},
		{
			name:  "localhost with port gets https",
			input: "localhost:3000",
			want:  "https://localhost:3000",
		},
		{
			name:  "IP with port and path gets https",
			input: "192.168.1.1:8080/path",
			want:  "https://192.168.1.1:8080/path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When
			got := normalizeURL(tt.input)

			// Then
			if got != tt.want {
				t.Errorf("normalizeURL(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestRun_Goto_SchemelessURL(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Given
	tabID, cleanup := createTestTabCLI(t)
	defer cleanup()

	// When
	cfg := testConfig()
	code := run([]string{"--target", tabID, "goto", "example.com"}, cfg)

	// Then
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("expected exit code %d, got %d, stderr: %s", ExitSuccess, code, stderr)
	}

	stdout := cfg.Stdout.(*bytes.Buffer).String()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	if result["url"] == nil {
		t.Error("expected 'url' field in output")
	}
}
