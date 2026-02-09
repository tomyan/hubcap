package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestEphemeralAutoLaunch(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HUBCAP_CONFIG_DIR", dir)

	// Create an ephemeral profile
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
		Timeout: 30 * time.Second,
		Output:  "json",
		Stdout:  &bytes.Buffer{},
		Stderr:  &bytes.Buffer{},
	}

	// Using --profile eph should auto-launch Chrome and connect on its port
	code := run([]string{"--profile", "eph", "tabs"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("ephemeral auto-launch failed (exit %d): %s", code, stderr)
	}

	// Verify an ephemeral session file was created
	ephDir := filepath.Join(dir, "ephemeral")
	files, _ := os.ReadDir(ephDir)
	found := false
	for _, f := range files {
		if f.Name() == "eph.json" {
			found = true
		}
	}
	if !found {
		t.Error("ephemeral session file eph.json should exist")
	}

	// Clean up: read session file to get PID, kill Chrome
	data, err := os.ReadFile(filepath.Join(ephDir, "eph.json"))
	if err == nil {
		var sess ephemeralSession
		json.Unmarshal(data, &sess)
		if sess.PID > 0 {
			proc, _ := os.FindProcess(sess.PID)
			if proc != nil {
				proc.Kill()
			}
		}
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
