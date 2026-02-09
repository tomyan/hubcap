package launcher

import (
	"fmt"
	"os/exec"
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
