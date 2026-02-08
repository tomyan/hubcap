package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestConfigDir_Default(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()
	t.Setenv("HUBCAP_CONFIG_DIR", "")

	dir := configDir()
	home, _ := os.UserHomeDir()
	want := filepath.Join(home, ".config", "hubcap")
	if dir != want {
		t.Errorf("configDir() = %q, want %q", dir, want)
	}
}

func TestConfigDir_EnvOverride(t *testing.T) {
	t.Setenv("HUBCAP_CONFIG_DIR", "/tmp/hubcap-test-config")

	dir := configDir()
	if dir != "/tmp/hubcap-test-config" {
		t.Errorf("configDir() = %q, want /tmp/hubcap-test-config", dir)
	}
}

func TestLoadProfilesFile_NotFound(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	pf, err := loadProfilesFile(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should return empty but valid struct
	if pf.Default != "" {
		t.Errorf("Default = %q, want empty", pf.Default)
	}
	if pf.Profiles == nil {
		t.Error("Profiles should be initialized (not nil)")
	}
	if len(pf.Profiles) != 0 {
		t.Errorf("Profiles should be empty, got %d", len(pf.Profiles))
	}
}

func TestLoadProfilesFile_Valid(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	data := `{
		"default": "local",
		"profiles": {
			"local": {
				"host": "localhost",
				"port": 9222
			},
			"ci": {
				"host": "ci-host",
				"port": 9333,
				"headless": true
			}
		}
	}`
	if err := os.WriteFile(filepath.Join(dir, "profiles.json"), []byte(data), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	pf, err := loadProfilesFile(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if pf.Default != "local" {
		t.Errorf("Default = %q, want local", pf.Default)
	}
	if len(pf.Profiles) != 2 {
		t.Fatalf("want 2 profiles, got %d", len(pf.Profiles))
	}

	local := pf.Profiles["local"]
	if local.Host != "localhost" {
		t.Errorf("local.Host = %q, want localhost", local.Host)
	}
	if local.Port != 9222 {
		t.Errorf("local.Port = %d, want 9222", local.Port)
	}

	ci := pf.Profiles["ci"]
	if ci.Host != "ci-host" {
		t.Errorf("ci.Host = %q, want ci-host", ci.Host)
	}
	if ci.Headless != true {
		t.Error("ci.Headless should be true")
	}
}

func TestLoadProfilesFile_Invalid(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "profiles.json"), []byte("not json"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := loadProfilesFile(dir)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestSaveProfilesFile_RoundTrip(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	pf := &ProfilesFile{
		Default: "myprofile",
		Profiles: map[string]Profile{
			"myprofile": {
				Host: "example.com",
				Port: 9444,
			},
		},
	}

	if err := saveProfilesFile(dir, pf); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	// Verify the file exists and is valid JSON
	data, err := os.ReadFile(filepath.Join(dir, "profiles.json"))
	if err != nil {
		t.Fatalf("failed to read saved file: %v", err)
	}

	var decoded ProfilesFile
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("saved file is not valid JSON: %v", err)
	}

	if decoded.Default != "myprofile" {
		t.Errorf("round-trip Default = %q, want myprofile", decoded.Default)
	}
	p, ok := decoded.Profiles["myprofile"]
	if !ok {
		t.Fatal("round-trip missing profile 'myprofile'")
	}
	if p.Host != "example.com" {
		t.Errorf("round-trip Host = %q, want example.com", p.Host)
	}
	if p.Port != 9444 {
		t.Errorf("round-trip Port = %d, want 9444", p.Port)
	}
}

func TestSaveProfilesFile_CreatesDir(t *testing.T) {
	t.Parallel()

	dir := filepath.Join(t.TempDir(), "nested", "dir")

	pf := &ProfilesFile{
		Default:  "test",
		Profiles: map[string]Profile{"test": {Port: 1234}},
	}

	if err := saveProfilesFile(dir, pf); err != nil {
		t.Fatalf("save should create intermediate dirs: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "profiles.json")); err != nil {
		t.Error("profiles.json should exist after save")
	}
}

func TestProfileAllFields(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	data := `{
		"default": "full",
		"profiles": {
			"full": {
				"host": "myhost",
				"port": 9555,
				"timeout": "30s",
				"output": "text",
				"target": "ABCD",
				"chrome_path": "/usr/bin/chrome",
				"headless": true,
				"chrome_data_dir": "/tmp/data",
				"ephemeral": true,
				"ephemeral_timeout": "10m"
			}
		}
	}`
	if err := os.WriteFile(filepath.Join(dir, "profiles.json"), []byte(data), 0644); err != nil {
		t.Fatal(err)
	}

	pf, err := loadProfilesFile(dir)
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	p := pf.Profiles["full"]
	if p.Host != "myhost" {
		t.Errorf("Host = %q", p.Host)
	}
	if p.Port != 9555 {
		t.Errorf("Port = %d", p.Port)
	}
	if p.Timeout != "30s" {
		t.Errorf("Timeout = %q", p.Timeout)
	}
	if p.Output != "text" {
		t.Errorf("Output = %q", p.Output)
	}
	if p.Target != "ABCD" {
		t.Errorf("Target = %q", p.Target)
	}
	if p.ChromePath != "/usr/bin/chrome" {
		t.Errorf("ChromePath = %q", p.ChromePath)
	}
	if !p.Headless {
		t.Error("Headless should be true")
	}
	if p.ChromeDataDir != "/tmp/data" {
		t.Errorf("ChromeDataDir = %q", p.ChromeDataDir)
	}
	if !p.Ephemeral {
		t.Error("Ephemeral should be true")
	}
	if p.EphemeralTimeout != "10m" {
		t.Errorf("EphemeralTimeout = %q", p.EphemeralTimeout)
	}
}
