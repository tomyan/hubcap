package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRun_WithProfile(t *testing.T) {
	// Create a temp config dir with a profile
	dir := t.TempDir()
	t.Setenv("HUBCAP_CONFIG_DIR", dir)

	pf := &ProfilesFile{
		Default: "test",
		Profiles: map[string]Profile{
			"test": {
				Host: "localhost",
				Port: testChromePort,
			},
		},
	}
	if err := saveProfilesFile(dir, pf); err != nil {
		t.Fatalf("save profiles: %v", err)
	}

	cfg := &Config{
		Timeout: 5 * time.Second,
		Output:  "json",
		Stdout:  &bytes.Buffer{},
		Stderr:  &bytes.Buffer{},
	}

	// Run with --profile test — should use the profile's port
	code := run([]string{"--profile", "test", "title"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Errorf("expected success, got %d: %s", code, stderr)
	}
}

func TestRun_ProfileFlagOverride(t *testing.T) {
	// Profile sets port 1111, but --port flag overrides it
	dir := t.TempDir()
	t.Setenv("HUBCAP_CONFIG_DIR", dir)

	pf := &ProfilesFile{
		Default: "bad",
		Profiles: map[string]Profile{
			"bad": {
				Host: "localhost",
				Port: 1111, // wrong port
			},
		},
	}
	if err := saveProfilesFile(dir, pf); err != nil {
		t.Fatalf("save profiles: %v", err)
	}

	cfg := &Config{
		Timeout: 5 * time.Second,
		Output:  "json",
		Stdout:  &bytes.Buffer{},
		Stderr:  &bytes.Buffer{},
	}

	// --port flag should override profile port
	code := run([]string{"--profile", "bad", "--port", intToStr(testChromePort), "title"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Errorf("expected success (flag override), got %d: %s", code, stderr)
	}
}

func TestRun_ProfileEnvVar(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HUBCAP_CONFIG_DIR", dir)
	t.Setenv("HUBCAP_PROFILE", "envprofile")

	pf := &ProfilesFile{
		Profiles: map[string]Profile{
			"envprofile": {
				Host: "localhost",
				Port: testChromePort,
			},
		},
	}
	if err := saveProfilesFile(dir, pf); err != nil {
		t.Fatalf("save profiles: %v", err)
	}

	cfg := &Config{
		Timeout: 5 * time.Second,
		Output:  "json",
		Stdout:  &bytes.Buffer{},
		Stderr:  &bytes.Buffer{},
	}

	code := run([]string{"title"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Errorf("expected success via HUBCAP_PROFILE, got %d: %s", code, stderr)
	}
}

func TestRun_NoProfileBackwardCompat(t *testing.T) {
	// No profiles.json, no --profile flag — should work as before
	dir := t.TempDir()
	t.Setenv("HUBCAP_CONFIG_DIR", dir)
	t.Setenv("HUBCAP_PROFILE", "")

	cfg := &Config{
		Port:    testChromePort,
		Host:    "localhost",
		Timeout: 5 * time.Second,
		Output:  "json",
		Stdout:  &bytes.Buffer{},
		Stderr:  &bytes.Buffer{},
	}

	code := run([]string{"title"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Errorf("expected success (backward compat), got %d: %s", code, stderr)
	}
}

func TestRun_ProfileNotFound(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HUBCAP_CONFIG_DIR", dir)

	cfg := &Config{
		Timeout: 5 * time.Second,
		Output:  "json",
		Stdout:  &bytes.Buffer{},
		Stderr:  &bytes.Buffer{},
	}

	code := run([]string{"--profile", "nonexistent", "title"}, cfg)
	if code != ExitError {
		t.Errorf("expected ExitError for nonexistent profile, got %d", code)
	}
	stderr := cfg.Stderr.(*bytes.Buffer).String()
	if !contains(stderr, "nonexistent") {
		t.Errorf("stderr should mention profile name, got: %s", stderr)
	}
}

func TestRun_HubcaprcOverridesProfile(t *testing.T) {
	// Profile sets port 1111, .hubcaprc in CWD sets the correct port
	dir := t.TempDir()
	t.Setenv("HUBCAP_CONFIG_DIR", dir)

	pf := &ProfilesFile{
		Default: "bad",
		Profiles: map[string]Profile{
			"bad": {
				Host: "localhost",
				Port: 1111,
			},
		},
	}
	if err := saveProfilesFile(dir, pf); err != nil {
		t.Fatal(err)
	}

	// Create .hubcaprc in a temp working directory
	workDir := t.TempDir()
	rc := []byte(`{"port": ` + intToStr(testChromePort) + `}`)
	if err := os.WriteFile(filepath.Join(workDir, ".hubcaprc"), rc, 0644); err != nil {
		t.Fatal(err)
	}

	// Change to workDir so loadConfigFile picks up the .hubcaprc
	origDir, _ := os.Getwd()
	os.Chdir(workDir)
	defer os.Chdir(origDir)

	cfg := &Config{
		Timeout: 5 * time.Second,
		Output:  "json",
		Stdout:  &bytes.Buffer{},
		Stderr:  &bytes.Buffer{},
	}

	code := run([]string{"title"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Errorf("expected success (.hubcaprc overrides profile), got %d: %s", code, stderr)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsImpl(s, substr))
}

func containsImpl(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func intToStr(i int) string {
	return fmt.Sprintf("%d", i)
}
