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

func setupTestConfig(t *testing.T) (*Config, string) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HUBCAP_CONFIG_DIR", dir)
	cfg := &Config{
		Port:    testChromePort,
		Host:    "localhost",
		Timeout: 5 * time.Second,
		Output:  "json",
		Stdout:  &bytes.Buffer{},
		Stderr:  &bytes.Buffer{},
	}
	return cfg, dir
}

func TestSetupAdd(t *testing.T) {
	cfg, dir := setupTestConfig(t)

	code := run([]string{"setup", "add", "myprofile", "--host", "example.com", "--port", "9333"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("setup add failed: %s", stderr)
	}

	// Verify profile was saved
	pf, err := loadProfilesFile(dir)
	if err != nil {
		t.Fatalf("load profiles: %v", err)
	}
	p, ok := pf.Profiles["myprofile"]
	if !ok {
		t.Fatal("profile 'myprofile' not found after add")
	}
	if p.Host != "example.com" {
		t.Errorf("Host = %q, want example.com", p.Host)
	}
	if p.Port != 9333 {
		t.Errorf("Port = %d, want 9333", p.Port)
	}
}

func TestSetupAdd_SetDefault(t *testing.T) {
	cfg, dir := setupTestConfig(t)

	code := run([]string{"setup", "add", "newdefault", "--port", "9222", "--set-default"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("setup add with --set-default failed: %s", stderr)
	}

	pf, _ := loadProfilesFile(dir)
	if pf.Default != "newdefault" {
		t.Errorf("Default = %q, want newdefault", pf.Default)
	}
}

func TestSetupAdd_MissingName(t *testing.T) {
	cfg, _ := setupTestConfig(t)

	code := run([]string{"setup", "add"}, cfg)
	if code != ExitError {
		t.Errorf("setup add without name: want ExitError, got %d", code)
	}
}

func TestSetupAdd_DuplicateName(t *testing.T) {
	cfg, dir := setupTestConfig(t)

	// Pre-create a profile
	pf := &ProfilesFile{
		Profiles: map[string]Profile{"existing": {Port: 1111}},
	}
	saveProfilesFile(dir, pf)

	code := run([]string{"setup", "add", "existing", "--port", "2222"}, cfg)
	if code != ExitError {
		t.Errorf("setup add duplicate: want ExitError, got %d", code)
	}
}

func TestSetupList_Empty(t *testing.T) {
	cfg, _ := setupTestConfig(t)

	code := run([]string{"setup", "list"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("setup list failed: %s", stderr)
	}
}

func TestSetupList_WithProfiles(t *testing.T) {
	cfg, dir := setupTestConfig(t)

	pf := &ProfilesFile{
		Default: "local",
		Profiles: map[string]Profile{
			"local": {Host: "localhost", Port: 9222},
			"ci":    {Host: "ci-host", Port: 9333},
		},
	}
	saveProfilesFile(dir, pf)

	code := run([]string{"setup", "list"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("setup list failed: %s", stderr)
	}

	stdout := cfg.Stdout.(*bytes.Buffer).String()

	// JSON output should contain both profiles
	var result interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("output not valid JSON: %v\n%s", err, stdout)
	}
}

func TestSetupShow(t *testing.T) {
	cfg, dir := setupTestConfig(t)

	pf := &ProfilesFile{
		Default: "local",
		Profiles: map[string]Profile{
			"local": {Host: "localhost", Port: 9222},
		},
	}
	saveProfilesFile(dir, pf)

	code := run([]string{"setup", "show", "local"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("setup show failed: %s", stderr)
	}

	stdout := cfg.Stdout.(*bytes.Buffer).String()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("output not valid JSON: %v", err)
	}
	if result["host"] != "localhost" {
		t.Errorf("show host = %v, want localhost", result["host"])
	}
}

func TestSetupShow_DefaultProfile(t *testing.T) {
	cfg, dir := setupTestConfig(t)

	pf := &ProfilesFile{
		Default: "mydefault",
		Profiles: map[string]Profile{
			"mydefault": {Host: "default-host", Port: 4444},
		},
	}
	saveProfilesFile(dir, pf)

	// No name argument — should show the default profile
	code := run([]string{"setup", "show"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("setup show (default) failed: %s", stderr)
	}

	stdout := cfg.Stdout.(*bytes.Buffer).String()
	var result map[string]interface{}
	json.Unmarshal([]byte(stdout), &result)
	if result["host"] != "default-host" {
		t.Errorf("show default host = %v, want default-host", result["host"])
	}
}

func TestSetupShow_NotFound(t *testing.T) {
	cfg, _ := setupTestConfig(t)

	code := run([]string{"setup", "show", "nonexistent"}, cfg)
	if code != ExitError {
		t.Errorf("setup show nonexistent: want ExitError, got %d", code)
	}
}

func TestSetupEdit(t *testing.T) {
	cfg, dir := setupTestConfig(t)

	pf := &ProfilesFile{
		Profiles: map[string]Profile{
			"editable": {Host: "old-host", Port: 1111},
		},
	}
	saveProfilesFile(dir, pf)

	code := run([]string{"setup", "edit", "editable", "--host", "new-host", "--port", "2222"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("setup edit failed: %s", stderr)
	}

	pf, _ = loadProfilesFile(dir)
	p := pf.Profiles["editable"]
	if p.Host != "new-host" {
		t.Errorf("Host after edit = %q, want new-host", p.Host)
	}
	if p.Port != 2222 {
		t.Errorf("Port after edit = %d, want 2222", p.Port)
	}
}

func TestSetupEdit_NotFound(t *testing.T) {
	cfg, _ := setupTestConfig(t)

	code := run([]string{"setup", "edit", "nonexistent", "--port", "1234"}, cfg)
	if code != ExitError {
		t.Errorf("setup edit nonexistent: want ExitError, got %d", code)
	}
}

func TestSetupRemove(t *testing.T) {
	cfg, dir := setupTestConfig(t)

	pf := &ProfilesFile{
		Profiles: map[string]Profile{
			"removeme": {Port: 1111},
		},
	}
	saveProfilesFile(dir, pf)

	code := run([]string{"setup", "remove", "removeme", "--force"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("setup remove failed: %s", stderr)
	}

	pf, _ = loadProfilesFile(dir)
	if _, ok := pf.Profiles["removeme"]; ok {
		t.Error("profile 'removeme' should be removed")
	}
}

func TestSetupRemove_NotFound(t *testing.T) {
	cfg, _ := setupTestConfig(t)

	code := run([]string{"setup", "remove", "nonexistent", "--force"}, cfg)
	if code != ExitError {
		t.Errorf("setup remove nonexistent: want ExitError, got %d", code)
	}
}

func TestSetupRemove_ClearsDefault(t *testing.T) {
	cfg, dir := setupTestConfig(t)

	pf := &ProfilesFile{
		Default: "removeme",
		Profiles: map[string]Profile{
			"removeme": {Port: 1111},
		},
	}
	saveProfilesFile(dir, pf)

	code := run([]string{"setup", "remove", "removeme", "--force"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("setup remove failed: %s", stderr)
	}

	pf, _ = loadProfilesFile(dir)
	if pf.Default != "" {
		t.Errorf("Default should be cleared after removing default profile, got %q", pf.Default)
	}
}

func TestSetupDefault_Get(t *testing.T) {
	cfg, dir := setupTestConfig(t)

	pf := &ProfilesFile{
		Default:  "current",
		Profiles: map[string]Profile{"current": {Port: 1111}},
	}
	saveProfilesFile(dir, pf)

	code := run([]string{"setup", "default"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("setup default failed: %s", stderr)
	}

	stdout := cfg.Stdout.(*bytes.Buffer).String()
	if !strings.Contains(stdout, "current") {
		t.Errorf("setup default output should contain 'current', got: %s", stdout)
	}
}

func TestSetupDefault_Set(t *testing.T) {
	cfg, dir := setupTestConfig(t)

	pf := &ProfilesFile{
		Profiles: map[string]Profile{
			"one": {Port: 1111},
			"two": {Port: 2222},
		},
	}
	saveProfilesFile(dir, pf)

	code := run([]string{"setup", "default", "two"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("setup default set failed: %s", stderr)
	}

	pf, _ = loadProfilesFile(dir)
	if pf.Default != "two" {
		t.Errorf("Default = %q, want two", pf.Default)
	}
}

func TestSetupDefault_SetNotFound(t *testing.T) {
	cfg, _ := setupTestConfig(t)

	code := run([]string{"setup", "default", "nonexistent"}, cfg)
	if code != ExitError {
		t.Errorf("setup default nonexistent: want ExitError, got %d", code)
	}
}

func TestSetupStatus(t *testing.T) {
	cfg, dir := setupTestConfig(t)

	pf := &ProfilesFile{
		Default: "test",
		Profiles: map[string]Profile{
			"test": {Host: "localhost", Port: testChromePort},
		},
	}
	saveProfilesFile(dir, pf)

	code := run([]string{"setup", "status", "test"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("setup status failed: %s", stderr)
	}

	stdout := cfg.Stdout.(*bytes.Buffer).String()
	// Should indicate Chrome is reachable
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("output not valid JSON: %v", err)
	}
	if result["connected"] != true {
		t.Errorf("expected connected=true, got %v", result["connected"])
	}
}

func TestSetupStatus_Unreachable(t *testing.T) {
	cfg, dir := setupTestConfig(t)

	pf := &ProfilesFile{
		Profiles: map[string]Profile{
			"bad": {Host: "localhost", Port: 19999},
		},
	}
	saveProfilesFile(dir, pf)

	code := run([]string{"setup", "status", "bad"}, cfg)
	// Should succeed but report not connected
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("setup status failed: %s", stderr)
	}

	stdout := cfg.Stdout.(*bytes.Buffer).String()
	var result map[string]interface{}
	json.Unmarshal([]byte(stdout), &result)
	if result["connected"] != false {
		t.Errorf("expected connected=false, got %v", result["connected"])
	}
}

func TestSetupAdd_AllFields(t *testing.T) {
	cfg, dir := setupTestConfig(t)

	code := run([]string{
		"setup", "add", "full",
		"--host", "myhost",
		"--port", "9555",
		"--timeout", "30s",
		"--output", "text",
		"--chrome-path", "/usr/bin/chrome",
		"--headless",
		"--chrome-data-dir", "/tmp/data",
		"--ephemeral",
		"--ephemeral-timeout", "10m",
	}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("setup add all fields failed: %s", stderr)
	}

	pf, _ := loadProfilesFile(dir)
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

func TestSetup_NoSubcommand_NonTTY(t *testing.T) {
	cfg, dir := setupTestConfig(t)

	pf := &ProfilesFile{
		Default: "local",
		Profiles: map[string]Profile{
			"local": {Host: "localhost", Port: 9222},
		},
	}
	saveProfilesFile(dir, pf)

	// Non-TTY stdin — should show config summary
	cfg.Stdin = strings.NewReader("")
	code := run([]string{"setup"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("setup (no subcommand) failed: %s", stderr)
	}
}

func TestSetup_UnknownSubcommand(t *testing.T) {
	cfg, _ := setupTestConfig(t)

	code := run([]string{"setup", "bogus"}, cfg)
	if code != ExitError {
		t.Errorf("setup bogus: want ExitError, got %d", code)
	}

	stderr := cfg.Stderr.(*bytes.Buffer).String()
	if !strings.Contains(stderr, "bogus") {
		t.Errorf("stderr should mention 'bogus', got: %s", stderr)
	}
}

func TestSetupLaunch(t *testing.T) {
	cfg, dir := setupTestConfig(t)

	pf := &ProfilesFile{
		Profiles: map[string]Profile{
			"launchtest": {
				Port:     19880,
				Headless: true,
			},
		},
	}
	saveProfilesFile(dir, pf)

	code := run([]string{"setup", "launch", "launchtest"}, cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("setup launch failed: %s", stderr)
	}

	stdout := cfg.Stdout.(*bytes.Buffer).String()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("output not valid JSON: %v\n%s", err, stdout)
	}

	// Should have launched Chrome
	if result["port"] != float64(19880) {
		t.Errorf("port = %v, want 19880", result["port"])
	}
	pid, ok := result["pid"]
	if !ok || pid == float64(0) {
		t.Error("should report a pid")
	}

	// Clean up: stop via setup remove or kill directly
	// The launched Chrome should be reachable
	cfg2, _ := setupTestConfig(t)
	t.Setenv("HUBCAP_CONFIG_DIR", dir)
	code = run([]string{"setup", "status", "launchtest"}, cfg2)
	if code != ExitSuccess {
		t.Fatal("status check failed after launch")
	}
	statusOut := cfg2.Stdout.(*bytes.Buffer).String()
	var statusResult map[string]interface{}
	json.Unmarshal([]byte(statusOut), &statusResult)
	if statusResult["connected"] != true {
		t.Error("Chrome should be connected after launch")
	}

	// Kill the process we launched
	if pidFloat, ok := pid.(float64); ok {
		proc, err := os.FindProcess(int(pidFloat))
		if err == nil {
			proc.Kill()
		}
	}
}

// Ensure profiles.json file permissions
func TestSetupAdd_FilePermissions(t *testing.T) {
	cfg, dir := setupTestConfig(t)

	code := run([]string{"setup", "add", "permtest", "--port", "1234"}, cfg)
	if code != ExitSuccess {
		t.Fatal("setup add failed")
	}

	info, err := os.Stat(filepath.Join(dir, "profiles.json"))
	if err != nil {
		t.Fatal(err)
	}
	perm := info.Mode().Perm()
	if perm != 0644 {
		t.Errorf("file permissions = %o, want 0644", perm)
	}
}
