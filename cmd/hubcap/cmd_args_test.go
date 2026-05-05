package main

import (
	"bytes"
	"strings"
	"testing"
)

// commandsRequiringInteraction lists commands that should be skipped from generic
// arg-handling tests because their normal execution would block (read stdin,
// open a TUI, run continuously, etc.). The arg-handling itself is exercised
// for every command because help/unknown-flag requests must short-circuit
// before any of that work happens.
var commandsRequiringInteraction = map[string]bool{
	"pipe":    true, // reads from stdin
	"shell":   true, // interactive REPL
	"inspect": true, // terminal TUI
	"record":  true, // long-running capture
}

// TestRun_PositionalHelp_ForAllCommands verifies that `hubcap <cmd> help` shows
// help for the command (and short-circuits before any work happens).
func TestRun_PositionalHelp_ForAllCommands(t *testing.T) {
	t.Parallel()
	for name := range commands {
		name := name
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			// Given
			cfg := testConfig()
			cfg.Port = 1 // Unused: help must short-circuit before any Chrome connection

			// When
			code := run([]string{name, "help"}, cfg)

			// Then
			if code != ExitSuccess {
				stderr := cfg.Stderr.(*bytes.Buffer).String()
				stdout := cfg.Stdout.(*bytes.Buffer).String()
				t.Fatalf("hubcap %s help: expected ExitSuccess, got %d\nstdout: %s\nstderr: %s", name, code, stdout, stderr)
			}
			stdout := cfg.Stdout.(*bytes.Buffer).String()
			if !strings.Contains(stdout, "hubcap "+name) {
				t.Errorf("hubcap %s help: expected output to mention 'hubcap %s', got: %s", name, name, stdout)
			}
		})
	}
}

// TestRun_HelpFlag_ForAllCommands verifies that `hubcap <cmd> --help` shows
// help for the command (and short-circuits before any work happens).
func TestRun_HelpFlag_ForAllCommands(t *testing.T) {
	t.Parallel()
	for name := range commands {
		name := name
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			// Given
			cfg := testConfig()
			cfg.Port = 1 // Unused: help must short-circuit before any Chrome connection

			// When
			code := run([]string{name, "--help"}, cfg)

			// Then
			if code != ExitSuccess {
				stderr := cfg.Stderr.(*bytes.Buffer).String()
				stdout := cfg.Stdout.(*bytes.Buffer).String()
				t.Fatalf("hubcap %s --help: expected ExitSuccess, got %d\nstdout: %s\nstderr: %s", name, code, stdout, stderr)
			}
			stdout := cfg.Stdout.(*bytes.Buffer).String()
			if !strings.Contains(stdout, "hubcap "+name) {
				t.Errorf("hubcap %s --help: expected output to mention 'hubcap %s', got: %s", name, name, stdout)
			}
		})
	}
}

// TestRun_ShortHelpFlag_ForAllCommands verifies that `hubcap <cmd> -h` shows help.
func TestRun_ShortHelpFlag_ForAllCommands(t *testing.T) {
	t.Parallel()
	for name := range commands {
		name := name
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			// Given
			cfg := testConfig()
			cfg.Port = 1

			// When
			code := run([]string{name, "-h"}, cfg)

			// Then
			if code != ExitSuccess {
				stderr := cfg.Stderr.(*bytes.Buffer).String()
				stdout := cfg.Stdout.(*bytes.Buffer).String()
				t.Fatalf("hubcap %s -h: expected ExitSuccess, got %d\nstdout: %s\nstderr: %s", name, code, stdout, stderr)
			}
		})
	}
}

// TestRun_UnknownFlag_RejectedByAllCommands verifies that an unknown long flag
// returns ExitError with a usage message and never contacts Chrome.
func TestRun_UnknownFlag_RejectedByAllCommands(t *testing.T) {
	t.Parallel()
	for name := range commands {
		if commandsRequiringInteraction[name] {
			continue
		}
		name := name
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			// Given
			cfg := testConfig()
			cfg.Port = 1 // Unused: unknown-flag rejection must short-circuit before any Chrome connection

			// When
			code := run([]string{name, "--definitely-not-a-real-flag"}, cfg)

			// Then
			if code == ExitSuccess {
				stdout := cfg.Stdout.(*bytes.Buffer).String()
				stderr := cfg.Stderr.(*bytes.Buffer).String()
				t.Fatalf("hubcap %s --definitely-not-a-real-flag: expected non-success exit, got ExitSuccess\nstdout: %s\nstderr: %s", name, stdout, stderr)
			}
			if code == ExitConnFailed {
				stderr := cfg.Stderr.(*bytes.Buffer).String()
				t.Fatalf("hubcap %s --definitely-not-a-real-flag: should reject unknown flag before connecting to Chrome, got ExitConnFailed\nstderr: %s", name, stderr)
			}
			stderr := cfg.Stderr.(*bytes.Buffer).String()
			if !strings.Contains(stderr, "definitely-not-a-real-flag") {
				t.Errorf("hubcap %s --definitely-not-a-real-flag: expected stderr to mention the unknown flag, got: %s", name, stderr)
			}
		})
	}
}

// TestRun_New_PositionalHelp is a focused regression test for the user-reported
// bug: `hubcap new help` previously navigated to https://help.
func TestRun_New_PositionalHelp(t *testing.T) {
	t.Parallel()
	// Given
	cfg := testConfig()
	cfg.Port = 1 // Unused: must short-circuit

	// When
	code := run([]string{"new", "help"}, cfg)

	// Then
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("hubcap new help: expected ExitSuccess, got %d, stderr: %s", code, stderr)
	}
	stdout := cfg.Stdout.(*bytes.Buffer).String()
	if !strings.Contains(stdout, "hubcap new") {
		t.Errorf("hubcap new help: expected docs output, got: %s", stdout)
	}
}

// TestRun_Click_UnknownFlag is a focused regression test: `hubcap click --foo`
// must reject --foo rather than treating it as a CSS selector.
func TestRun_Click_UnknownFlag(t *testing.T) {
	t.Parallel()
	// Given
	cfg := testConfig()
	cfg.Port = 1

	// When
	code := run([]string{"click", "--foo"}, cfg)

	// Then
	if code == ExitSuccess {
		t.Fatalf("hubcap click --foo: expected error exit, got ExitSuccess")
	}
	if code == ExitConnFailed {
		t.Fatalf("hubcap click --foo: should reject before connecting to Chrome")
	}
	stderr := cfg.Stderr.(*bytes.Buffer).String()
	if !strings.Contains(stderr, "foo") {
		t.Errorf("hubcap click --foo: expected stderr to mention 'foo', got: %s", stderr)
	}
}
