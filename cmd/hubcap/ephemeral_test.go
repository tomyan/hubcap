package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestEphemeralProfile_NoAutoLaunch(t *testing.T) {
	// Given — an ephemeral profile on an unused port
	dir := t.TempDir()
	t.Setenv("HUBCAP_CONFIG_DIR", dir)

	pf := &ProfilesFile{
		Default: "eph",
		Profiles: map[string]Profile{
			"eph": {
				Host:             "localhost",
				Port:             19891,
				Headless:         true,
				Ephemeral:        true,
				EphemeralTimeout: "10m",
			},
		},
	}
	saveProfilesFile(dir, pf)

	cfg := &Config{
		Host:    "localhost",
		Port:    9222,
		Timeout: 5 * time.Second,
		Output:  "json",
		Stdout:  &bytes.Buffer{},
		Stderr:  &bytes.Buffer{},
	}

	// When — running a command with the ephemeral profile
	code := run([]string{"--profile", "eph", "tabs"}, cfg)

	// Then — should fail to connect (no auto-launch)
	if code != ExitConnFailed {
		t.Errorf("expected ExitConnFailed, got %d", code)
	}
	stderr := cfg.Stderr.(*bytes.Buffer).String()
	if !strings.Contains(stderr, "hubcap setup launch") {
		t.Errorf("expected hint about 'hubcap setup launch', got: %s", stderr)
	}
}

func TestEphemeralTouch(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HUBCAP_CONFIG_DIR", dir)

	// Create ephemeral session file with old mtime
	ephDir := filepath.Join(dir, "ephemeral")
	os.MkdirAll(ephDir, 0755)

	sess := ephemeralSession{
		PID:     99999,
		Port:    19882,
		DataDir: "/tmp/fake",
		Timeout: "10m",
	}
	data, _ := json.Marshal(sess)
	sessFile := filepath.Join(ephDir, "touchtest.json")
	os.WriteFile(sessFile, data, 0644)

	// Set old mtime
	oldTime := time.Now().Add(-5 * time.Minute)
	os.Chtimes(sessFile, oldTime, oldTime)

	// Touch the session
	touchEphemeralSession(dir, "touchtest")

	// Mtime should be updated
	info, _ := os.Stat(sessFile)
	if info.ModTime().Before(time.Now().Add(-1 * time.Second)) {
		t.Error("mtime should be updated after touch")
	}
}

func TestCleanupStaleEphemeral(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HUBCAP_CONFIG_DIR", dir)

	ephDir := filepath.Join(dir, "ephemeral")
	os.MkdirAll(ephDir, 0755)

	// Create a stale session (timeout expired, process doesn't exist)
	sess := ephemeralSession{
		PID:     99999, // non-existent PID
		Port:    19883,
		DataDir: t.TempDir(), // temp dir that should get cleaned
		Timeout: "1s",        // 1 second timeout
	}
	data, _ := json.Marshal(sess)
	sessFile := filepath.Join(ephDir, "stale.json")
	os.WriteFile(sessFile, data, 0644)

	// Set old mtime (older than timeout)
	oldTime := time.Now().Add(-10 * time.Second)
	os.Chtimes(sessFile, oldTime, oldTime)

	// Run cleanup
	cleanupStaleEphemeral(dir)

	// Session file should be removed
	if _, err := os.Stat(sessFile); !os.IsNotExist(err) {
		t.Error("stale session file should be removed")
	}
}

func TestCleanupStaleEphemeral_ActiveSession(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HUBCAP_CONFIG_DIR", dir)

	ephDir := filepath.Join(dir, "ephemeral")
	os.MkdirAll(ephDir, 0755)

	// Create a fresh session (not stale)
	sess := ephemeralSession{
		PID:     os.Getpid(), // current process — definitely exists
		Port:    19884,
		DataDir: t.TempDir(),
		Timeout: "10m",
	}
	data, _ := json.Marshal(sess)
	sessFile := filepath.Join(ephDir, "active.json")
	os.WriteFile(sessFile, data, 0644)

	// mtime is now (fresh) — should not be cleaned
	cleanupStaleEphemeral(dir)

	if _, err := os.Stat(sessFile); os.IsNotExist(err) {
		t.Error("active session file should NOT be removed")
	}
}
