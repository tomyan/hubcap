// Package testutil provides test utilities for CDP tests.
package testutil

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"runtime"
	"time"
)

// ChromeInstance represents a running Chrome instance for testing.
type ChromeInstance struct {
	cmd     *exec.Cmd
	Port    int
	dataDir string
}

// StartChrome starts a headless Chrome instance on the specified port.
// Returns a ChromeInstance that must be stopped with Stop().
func StartChrome(port int) (*ChromeInstance, error) {
	// Find Chrome binary
	chromePath := findChrome()
	if chromePath == "" {
		return nil, fmt.Errorf("Chrome not found")
	}

	// Create temp directory for user data
	dataDir, err := os.MkdirTemp("", "cdp-test-chrome-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}

	// Start Chrome with remote debugging
	args := []string{
		"--headless",
		"--disable-gpu",
		"--no-sandbox",
		"--disable-dev-shm-usage",
		"--disable-extensions",
		"--disable-background-networking",
		"--disable-sync",
		"--disable-translate",
		"--mute-audio",
		"--no-first-run",
		"--disable-default-apps",
		fmt.Sprintf("--remote-debugging-port=%d", port),
		fmt.Sprintf("--user-data-dir=%s", dataDir),
		"about:blank",
	}

	cmd := exec.Command(chromePath, args...)
	cmd.Stdout = nil
	cmd.Stderr = nil

	if err := cmd.Start(); err != nil {
		os.RemoveAll(dataDir)
		return nil, fmt.Errorf("failed to start Chrome: %w", err)
	}

	instance := &ChromeInstance{
		cmd:     cmd,
		Port:    port,
		dataDir: dataDir,
	}

	// Wait for Chrome to be ready
	if err := waitForPort(port, 10*time.Second); err != nil {
		instance.Stop()
		return nil, fmt.Errorf("Chrome failed to start: %w", err)
	}

	return instance, nil
}

// Stop terminates the Chrome instance and cleans up.
func (c *ChromeInstance) Stop() error {
	if c.cmd != nil && c.cmd.Process != nil {
		c.cmd.Process.Kill()
		c.cmd.Wait()
	}
	if c.dataDir != "" {
		os.RemoveAll(c.dataDir)
	}
	return nil
}

// findChrome locates the Chrome binary on the system.
func findChrome() string {
	var paths []string

	switch runtime.GOOS {
	case "darwin":
		paths = []string{
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			"/Applications/Chromium.app/Contents/MacOS/Chromium",
		}
	case "linux":
		paths = []string{
			"/usr/bin/google-chrome",
			"/usr/bin/google-chrome-stable",
			"/usr/bin/chromium",
			"/usr/bin/chromium-browser",
			"/snap/bin/chromium",
		}
	case "windows":
		paths = []string{
			`C:\Program Files\Google\Chrome\Application\chrome.exe`,
			`C:\Program Files (x86)\Google\Chrome\Application\chrome.exe`,
		}
	}

	// Check PATH first
	if path, err := exec.LookPath("google-chrome"); err == nil {
		return path
	}
	if path, err := exec.LookPath("chromium"); err == nil {
		return path
	}

	// Check known locations
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}

	return ""
}

// waitForPort waits for a TCP port to become available.
func waitForPort(port int, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	addr := fmt.Sprintf("localhost:%d", port)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for port %d", port)
		case <-ticker.C:
			conn, err := net.DialTimeout("tcp", addr, 100*time.Millisecond)
			if err == nil {
				conn.Close()
				return nil
			}
		}
	}
}
