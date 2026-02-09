// Package launcher provides Chrome browser discovery, launching, and lifecycle management.
package launcher

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"time"
)

// LaunchOptions configures Chrome launching.
type LaunchOptions struct {
	ChromePath string // Path to Chrome binary (auto-detected if empty)
	Port       int    // Remote debugging port
	Headless   bool   // Run in headless mode
	DataDir    string // User data directory (temp dir created if empty)
}

// Instance represents a running Chrome instance.
type Instance struct {
	cmd        *exec.Cmd
	Port       int
	PID        int
	DataDir    string
	ownsData   bool // true if we created the data dir and should clean it up
}

// FindChrome locates Chrome on the system. If chromePath is non-empty and exists,
// it is returned directly. Otherwise, searches PATH and known install locations.
func FindChrome(chromePath string) string {
	if chromePath != "" {
		if _, err := os.Stat(chromePath); err == nil {
			return chromePath
		}
		return ""
	}

	// Check PATH first
	if path, err := exec.LookPath("google-chrome"); err == nil {
		return path
	}
	if path, err := exec.LookPath("chromium"); err == nil {
		return path
	}

	// Check known locations
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

	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}

	return ""
}

// IsPortOpen checks if a TCP port is accepting connections.
func IsPortOpen(host string, port int) bool {
	addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))
	conn, err := net.DialTimeout("tcp", addr, 100*time.Millisecond)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// WaitForPort waits for a TCP port to become available.
func WaitForPort(host string, port int, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for %s", net.JoinHostPort(host, fmt.Sprintf("%d", port)))
		case <-ticker.C:
			if IsPortOpen(host, port) {
				return nil
			}
		}
	}
}

// Launch starts a Chrome instance with the given options.
func Launch(opts LaunchOptions) (*Instance, error) {
	chromePath := FindChrome(opts.ChromePath)
	if chromePath == "" {
		return nil, fmt.Errorf("Chrome not found")
	}

	ownsData := false
	dataDir := opts.DataDir
	if dataDir == "" {
		var err error
		dataDir, err = os.MkdirTemp("", "hubcap-chrome-*")
		if err != nil {
			return nil, fmt.Errorf("failed to create temp dir: %w", err)
		}
		ownsData = true
	}

	args := []string{
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
		fmt.Sprintf("--remote-debugging-port=%d", opts.Port),
		fmt.Sprintf("--user-data-dir=%s", dataDir),
		"about:blank",
	}
	if opts.Headless {
		args = append([]string{"--headless"}, args...)
	}

	cmd := exec.Command(chromePath, args...)
	cmd.Stdout = nil
	cmd.Stderr = nil

	if err := cmd.Start(); err != nil {
		if ownsData {
			os.RemoveAll(dataDir)
		}
		return nil, fmt.Errorf("failed to start Chrome: %w", err)
	}

	inst := &Instance{
		cmd:      cmd,
		Port:     opts.Port,
		PID:      cmd.Process.Pid,
		DataDir:  dataDir,
		ownsData: ownsData,
	}

	if err := WaitForPort("localhost", opts.Port, 30*time.Second); err != nil {
		inst.Stop()
		return nil, fmt.Errorf("Chrome failed to start: %w", err)
	}

	return inst, nil
}

// ChromeInfo contains version information from a running Chrome instance.
type ChromeInfo struct {
	Browser  string `json:"Browser"`
	Protocol string `json:"Protocol-Version"`
	V8       string `json:"V8-Version"`
	WebKit   string `json:"WebKit-Version"`
}

// DetectRunning checks if a Chrome debug port is responding and returns version info.
func DetectRunning(host string, port int) (*ChromeInfo, error) {
	url := fmt.Sprintf("http://%s/json/version", net.JoinHostPort(host, fmt.Sprintf("%d", port)))
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("Chrome not reachable at %s:%d: %w", host, port, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	var info ChromeInfo
	if err := json.Unmarshal(body, &info); err != nil {
		return nil, fmt.Errorf("parsing version info: %w", err)
	}
	return &info, nil
}

// Stop terminates the Chrome instance and cleans up.
func (inst *Instance) Stop() error {
	if inst.cmd != nil && inst.cmd.Process != nil {
		inst.cmd.Process.Kill()
		inst.cmd.Wait()

		// Kill orphaned child processes
		if inst.DataDir != "" {
			killCmd := exec.Command("pkill", "-9", "-f", inst.DataDir)
			killCmd.Run()
		}
		inst.cmd = nil
	}
	if inst.ownsData && inst.DataDir != "" {
		time.Sleep(100 * time.Millisecond)
		os.RemoveAll(inst.DataDir)
		inst.DataDir = ""
	}
	return nil
}
