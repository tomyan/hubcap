// Package testutil provides test utilities for Chrome tests.
package testutil

import (
	"fmt"
	"os/exec"
	"time"

	"github.com/tomyan/hubcap/internal/chrome/launcher"
)

// ChromeInstance represents a running Chrome instance for testing.
type ChromeInstance struct {
	inst *launcher.Instance
	Port int
}

// StartChrome starts a headless Chrome instance on the specified port.
// Returns a ChromeInstance that must be stopped with Stop().
func StartChrome(port int) (*ChromeInstance, error) {
	// Kill any stale Chrome processes that might be using this port
	killStale := exec.Command("pkill", "-9", "-f", fmt.Sprintf("remote-debugging-port=%d", port))
	killStale.Run()
	time.Sleep(200 * time.Millisecond)

	inst, err := launcher.Launch(launcher.LaunchOptions{
		Port:     port,
		Headless: true,
	})
	if err != nil {
		return nil, err
	}

	return &ChromeInstance{inst: inst, Port: port}, nil
}

// Stop terminates the Chrome instance and cleans up.
func (c *ChromeInstance) Stop() error {
	return c.inst.Stop()
}
