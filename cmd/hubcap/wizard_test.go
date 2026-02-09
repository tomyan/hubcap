package main

import (
	"bufio"
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestPromptChoice(t *testing.T) {
	t.Parallel()

	scanner := bufio.NewScanner(strings.NewReader("2\n"))
	var output bytes.Buffer

	choice, err := promptChoice(scanner, &output, "Pick one:", []string{"Apple", "Banana", "Cherry"}, "")
	if err != nil {
		t.Fatalf("promptChoice error: %v", err)
	}
	if choice != 1 { // 0-indexed, user typed "2"
		t.Errorf("choice = %d, want 1", choice)
	}

	// Output should show numbered options
	out := output.String()
	if !strings.Contains(out, "1) Apple") {
		t.Errorf("output should show '1) Apple', got: %s", out)
	}
	if !strings.Contains(out, "2) Banana") {
		t.Errorf("output should show '2) Banana', got: %s", out)
	}
}

func TestPromptChoice_Default(t *testing.T) {
	t.Parallel()

	scanner := bufio.NewScanner(strings.NewReader("\n"))
	var output bytes.Buffer

	choice, err := promptChoice(scanner, &output, "Pick one:", []string{"Apple", "Banana"}, "Apple")
	if err != nil {
		t.Fatalf("promptChoice error: %v", err)
	}
	if choice != 0 { // default is "Apple" = index 0
		t.Errorf("choice = %d, want 0 (default)", choice)
	}
}

func TestPromptChoice_InvalidThenValid(t *testing.T) {
	t.Parallel()

	scanner := bufio.NewScanner(strings.NewReader("99\n1\n"))
	var output bytes.Buffer

	choice, err := promptChoice(scanner, &output, "Pick:", []string{"A", "B"}, "")
	if err != nil {
		t.Fatalf("promptChoice error: %v", err)
	}
	if choice != 0 {
		t.Errorf("choice = %d, want 0", choice)
	}
}

func TestPromptString(t *testing.T) {
	t.Parallel()

	scanner := bufio.NewScanner(strings.NewReader("hello world\n"))
	var output bytes.Buffer

	result, err := promptString(scanner, &output, "Enter value:", "default")
	if err != nil {
		t.Fatalf("promptString error: %v", err)
	}
	if result != "hello world" {
		t.Errorf("result = %q, want 'hello world'", result)
	}
}

func TestPromptString_Default(t *testing.T) {
	t.Parallel()

	scanner := bufio.NewScanner(strings.NewReader("\n"))
	var output bytes.Buffer

	result, err := promptString(scanner, &output, "Enter value:", "mydefault")
	if err != nil {
		t.Fatalf("promptString error: %v", err)
	}
	if result != "mydefault" {
		t.Errorf("result = %q, want 'mydefault'", result)
	}
}

func TestPromptConfirm_Yes(t *testing.T) {
	t.Parallel()

	scanner := bufio.NewScanner(strings.NewReader("y\n"))
	var output bytes.Buffer

	result, err := promptConfirm(scanner, &output, "Continue?")
	if err != nil {
		t.Fatalf("promptConfirm error: %v", err)
	}
	if !result {
		t.Error("expected true for 'y'")
	}
}

func TestPromptConfirm_No(t *testing.T) {
	t.Parallel()

	scanner := bufio.NewScanner(strings.NewReader("n\n"))
	var output bytes.Buffer

	result, err := promptConfirm(scanner, &output, "Continue?")
	if err != nil {
		t.Fatalf("promptConfirm error: %v", err)
	}
	if result {
		t.Error("expected false for 'n'")
	}
}

func TestSetupWizard_CustomPort(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HUBCAP_CONFIG_DIR", dir)

	// No Chrome on 9222, so wizard shows options.
	// Input:
	// 1) "2" = Connect to a different port
	// 2) port: "9301" (our test chrome)
	// 3) profile name: "mylocal"
	input := strings.NewReader("2\n9301\nmylocal\n")

	cfg := &Config{
		Host:    "localhost",
		Port:    9222,
		Timeout: 5 * time.Second,
		Output:  "json",
		Stdin:   input,
		Stdout:  &bytes.Buffer{},
		Stderr:  &bytes.Buffer{},
	}

	code := runSetupWizard(cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("wizard failed (exit %d):\nstderr: %s", code, stderr)
	}

	// Verify profile was created
	pf, err := loadProfilesFile(dir)
	if err != nil {
		t.Fatalf("load profiles: %v", err)
	}

	p, ok := pf.Profiles["mylocal"]
	if !ok {
		t.Fatal("profile 'mylocal' should exist after wizard")
	}
	if p.Port != 9301 {
		t.Errorf("Port = %d, want 9301", p.Port)
	}
	if pf.Default != "mylocal" {
		t.Errorf("Default = %q, want mylocal", pf.Default)
	}
}

func TestSetupWizard_RemoteHost(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HUBCAP_CONFIG_DIR", dir)

	// No Chrome on 9222, pick remote.
	// Input:
	// 1) "3" = Connect to a remote Chrome
	// 2) host: "ci-box"
	// 3) port: "9222" (default)
	// 4) profile name: "ci"
	input := strings.NewReader("3\nci-box\n9222\nci\n")

	cfg := &Config{
		Host:    "localhost",
		Port:    9222,
		Timeout: 5 * time.Second,
		Output:  "json",
		Stdin:   input,
		Stdout:  &bytes.Buffer{},
		Stderr:  &bytes.Buffer{},
	}

	code := runSetupWizard(cfg)
	if code != ExitSuccess {
		stderr := cfg.Stderr.(*bytes.Buffer).String()
		t.Fatalf("wizard failed (exit %d):\nstderr: %s", code, stderr)
	}

	pf, _ := loadProfilesFile(dir)
	p, ok := pf.Profiles["ci"]
	if !ok {
		t.Fatal("profile 'ci' should exist after wizard")
	}
	if p.Host != "ci-box" {
		t.Errorf("Host = %q, want ci-box", p.Host)
	}
	if p.Port != 9222 {
		t.Errorf("Port = %d, want 9222", p.Port)
	}
}
