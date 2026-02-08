package launcher

import (
	"os"
	"testing"
	"time"
)

func TestFindChrome(t *testing.T) {
	t.Parallel()

	path := FindChrome("")
	if path == "" {
		t.Skip("Chrome not found on this system")
	}

	// Path should exist on disk
	if _, err := os.Stat(path); err != nil {
		t.Errorf("FindChrome returned path that doesn't exist: %s", path)
	}
}

func TestFindChrome_ExplicitPath(t *testing.T) {
	t.Parallel()

	// If an explicit path is given and exists, use it
	path := FindChrome("/bin/sh")
	if path != "/bin/sh" {
		t.Errorf("FindChrome with explicit path: want /bin/sh, got %s", path)
	}
}

func TestFindChrome_ExplicitPath_NotFound(t *testing.T) {
	t.Parallel()

	path := FindChrome("/nonexistent/chrome")
	if path != "" {
		t.Errorf("FindChrome with nonexistent explicit path: want empty, got %s", path)
	}
}

func TestIsPortOpen_ClosedPort(t *testing.T) {
	t.Parallel()

	// Port 19999 should not be open
	if IsPortOpen("localhost", 19999) {
		t.Error("expected port 19999 to be closed")
	}
}

func TestWaitForPort_Timeout(t *testing.T) {
	t.Parallel()

	err := WaitForPort("localhost", 19999, 100*time.Millisecond)
	if err == nil {
		t.Error("expected timeout error for closed port")
	}
}

func TestLaunchAndStop(t *testing.T) {
	t.Parallel()

	chromePath := FindChrome("")
	if chromePath == "" {
		t.Skip("Chrome not found on this system")
	}

	opts := LaunchOptions{
		ChromePath: chromePath,
		Port:       19876,
		Headless:   true,
	}

	inst, err := Launch(opts)
	if err != nil {
		t.Fatalf("Launch failed: %v", err)
	}
	defer inst.Stop()

	// Port should be open after launch
	if !IsPortOpen("localhost", 19876) {
		t.Error("port should be open after launch")
	}

	// Stop should close the port
	if err := inst.Stop(); err != nil {
		t.Errorf("Stop failed: %v", err)
	}

	// Give it a moment to release the port
	time.Sleep(200 * time.Millisecond)

	if IsPortOpen("localhost", 19876) {
		t.Error("port should be closed after stop")
	}
}

func TestLaunch_InvalidChromePath(t *testing.T) {
	t.Parallel()

	opts := LaunchOptions{
		ChromePath: "/nonexistent/chrome",
		Port:       19877,
		Headless:   true,
	}

	_, err := Launch(opts)
	if err == nil {
		t.Error("expected error for invalid Chrome path")
	}
}

func TestLaunch_CustomDataDir(t *testing.T) {
	t.Parallel()

	chromePath := FindChrome("")
	if chromePath == "" {
		t.Skip("Chrome not found on this system")
	}

	dataDir, err := os.MkdirTemp("", "hubcap-launcher-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(dataDir)

	opts := LaunchOptions{
		ChromePath: chromePath,
		Port:       19878,
		Headless:   true,
		DataDir:    dataDir,
	}

	inst, err := Launch(opts)
	if err != nil {
		t.Fatalf("Launch failed: %v", err)
	}
	defer inst.Stop()

	// Data dir should not be cleaned up by Stop when user-provided
	inst.Stop()
	if _, err := os.Stat(dataDir); err != nil {
		t.Error("user-provided data dir should not be removed on stop")
	}
}
