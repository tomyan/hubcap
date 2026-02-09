package launcher

import (
	"fmt"
	"os/exec"
	"runtime"
	"time"
)

// CommandRunner abstracts command execution for testability.
type CommandRunner interface {
	Run(name string, args ...string) ([]byte, error)
}

// DefaultCommandRunner executes commands via os/exec.
type DefaultCommandRunner struct{}

// Run executes a command and returns its combined output.
func (d DefaultCommandRunner) Run(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).CombinedOutput()
}

// quitChromeDarwin gracefully quits Chrome on macOS.
// If Chrome isn't running, returns immediately.
// Falls back to pkill if Chrome doesn't quit within maxWaitMs milliseconds.
func quitChromeDarwin(runner CommandRunner, maxWaitMs int) error {
	// Check if Chrome is running
	_, err := runner.Run("pgrep", "-x", "Google Chrome")
	if err != nil {
		// Chrome not running — nothing to quit
		return nil
	}

	// Ask Chrome to quit gracefully via AppleScript
	_, err = runner.Run("osascript", "-e", `tell application "Google Chrome" to quit`)
	if err != nil {
		return fmt.Errorf("osascript quit failed: %w", err)
	}

	// Poll until Chrome exits or timeout
	deadline := time.Now().Add(time.Duration(maxWaitMs) * time.Millisecond)
	for time.Now().Before(deadline) {
		_, err := runner.Run("pgrep", "-x", "Google Chrome")
		if err != nil {
			// Chrome has exited
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}

	// Fallback: force kill
	runner.Run("pkill", "-x", "Google Chrome")
	return nil
}

// relaunchChromeDarwin launches Chrome on macOS using `open -a` with only
// the remote debugging port flag. This preserves the user's default profile,
// tabs, and extensions.
func relaunchChromeDarwin(runner CommandRunner, port int) error {
	_, err := runner.Run("open", "-a", "Google Chrome", "--args",
		fmt.Sprintf("--remote-debugging-port=%d", port))
	if err != nil {
		return fmt.Errorf("failed to launch Chrome: %w", err)
	}
	return nil
}

const defaultQuitTimeoutMs = 5000

// RelaunchOptions configures the relaunch behaviour.
type RelaunchOptions struct {
	Port     int           // Remote debugging port (default 9222)
	GOOS     string        // Override runtime.GOOS for testing
	Runner   CommandRunner // Override command runner for testing
	WaitFunc func() error  // Override wait-for-port for testing
}

// RelaunchUserChrome gracefully quits the user's Chrome and relaunches it
// with remote debugging enabled, preserving their profile, tabs, and extensions.
func RelaunchUserChrome(opts RelaunchOptions) error {
	goos := opts.GOOS
	if goos == "" {
		goos = runtime.GOOS
	}

	runner := opts.Runner
	if runner == nil {
		runner = DefaultCommandRunner{}
	}

	port := opts.Port
	if port == 0 {
		port = 9222
	}

	switch goos {
	case "darwin":
		if err := quitChromeDarwin(runner, defaultQuitTimeoutMs); err != nil {
			return err
		}
		if err := relaunchChromeDarwin(runner, port); err != nil {
			return err
		}
	case "linux":
		if err := quitChromeLinux(runner, defaultQuitTimeoutMs); err != nil {
			return err
		}
		if err := relaunchChromeLinux(runner, port); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported platform: %s", goos)
	}

	if opts.WaitFunc != nil {
		return opts.WaitFunc()
	}
	return WaitForPort("localhost", port, 30*time.Second)
}

// quitChromeLinux gracefully quits Chrome on Linux using SIGTERM.
func quitChromeLinux(runner CommandRunner, maxWaitMs int) error {
	_, err := runner.Run("pgrep", "-x", "chrome")
	if err != nil {
		return nil // not running
	}

	runner.Run("pkill", "-TERM", "-x", "chrome")

	deadline := time.Now().Add(time.Duration(maxWaitMs) * time.Millisecond)
	for time.Now().Before(deadline) {
		_, err := runner.Run("pgrep", "-x", "chrome")
		if err != nil {
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}

	runner.Run("pkill", "-KILL", "-x", "chrome")
	return nil
}

// relaunchChromeLinux launches Chrome on Linux directly with the debugging port.
func relaunchChromeLinux(runner CommandRunner, port int) error {
	chromePath := FindChrome("")
	if chromePath == "" {
		return fmt.Errorf("Chrome not found")
	}
	_, err := runner.Run(chromePath, fmt.Sprintf("--remote-debugging-port=%d", port))
	if err != nil {
		return fmt.Errorf("failed to launch Chrome: %w", err)
	}
	return nil
}
