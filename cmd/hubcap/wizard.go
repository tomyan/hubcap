package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/tomyan/hubcap/internal/chrome/launcher"
	"golang.org/x/term"
)

// isTerminal checks if the given reader is a terminal.
func isTerminal(r io.Reader) bool {
	f, ok := r.(*os.File)
	if !ok {
		return false
	}
	return term.IsTerminal(int(f.Fd()))
}

// promptChoice shows numbered options and returns the 0-based index of the chosen option.
func promptChoice(scanner *bufio.Scanner, w io.Writer, prompt string, options []string, defaultVal string) (int, error) {
	defaultIdx := -1
	for i, opt := range options {
		if opt == defaultVal {
			defaultIdx = i
		}
	}

	for {
		fmt.Fprintln(w, prompt)
		for i, opt := range options {
			marker := "  "
			if i == defaultIdx {
				marker = "* "
			}
			fmt.Fprintf(w, "%s%d) %s\n", marker, i+1, opt)
		}

		if defaultIdx >= 0 {
			fmt.Fprintf(w, "Choice [%d]: ", defaultIdx+1)
		} else {
			fmt.Fprint(w, "Choice: ")
		}

		if !scanner.Scan() {
			return 0, fmt.Errorf("no input")
		}
		line := strings.TrimSpace(scanner.Text())

		if line == "" && defaultIdx >= 0 {
			return defaultIdx, nil
		}

		n, err := strconv.Atoi(line)
		if err != nil || n < 1 || n > len(options) {
			fmt.Fprintf(w, "Please enter a number between 1 and %d.\n\n", len(options))
			continue
		}

		return n - 1, nil
	}
}

// promptString prompts for a string value with an optional default.
func promptString(scanner *bufio.Scanner, w io.Writer, prompt string, defaultVal string) (string, error) {
	if defaultVal != "" {
		fmt.Fprintf(w, "%s [%s]: ", prompt, defaultVal)
	} else {
		fmt.Fprintf(w, "%s ", prompt)
	}

	if !scanner.Scan() {
		return "", fmt.Errorf("no input")
	}
	line := strings.TrimSpace(scanner.Text())
	if line == "" {
		return defaultVal, nil
	}
	return line, nil
}

// promptConfirm prompts for a yes/no answer.
func promptConfirm(scanner *bufio.Scanner, w io.Writer, prompt string) (bool, error) {
	fmt.Fprintf(w, "%s (y/n): ", prompt)

	if !scanner.Scan() {
		return false, fmt.Errorf("no input")
	}
	line := strings.TrimSpace(strings.ToLower(scanner.Text()))
	return line == "y" || line == "yes", nil
}

// runSetupWizard runs the interactive setup wizard.
// Happy path: detect Chrome → save profile (2 prompts).
// If Chrome isn't running: offer to relaunch it with CDP enabled.
// Falls through to advanced setup for remote/custom configurations.
func runSetupWizard(cfg *Config) int {
	r := bufio.NewScanner(cfg.Stdin)
	w := cfg.Stderr

	fmt.Fprintln(w, "Hubcap Setup")
	fmt.Fprintln(w)

	// Step 1: Check if Chrome CDP is already running on the default port
	detected := launcher.IsPortOpen("localhost", 9222)

	if detected {
		// Happy path: Chrome is already listening
		info, _ := launcher.DetectRunning("localhost", 9222)
		if info != nil && info.Browser != "" {
			fmt.Fprintf(w, "Found %s on localhost:9222\n\n", info.Browser)
		} else {
			fmt.Fprintln(w, "Chrome detected on localhost:9222")
		fmt.Fprintln(w)
		}

		name, err := promptString(r, w, "Profile name:", "default")
		if err != nil {
			fmt.Fprintf(w, "error: %v\n", err)
			return ExitError
		}
		if name == "" {
			name = "default"
		}

		return saveWizardProfile(w, name, "localhost", 9222, false)
	}

	// Chrome not detected — offer options
	fmt.Fprintln(w, "No Chrome detected on localhost:9222")
	fmt.Fprintln(w)

	idx, err := promptChoice(r, w, "What would you like to do?", []string{
		"Relaunch Chrome with CDP enabled",
		"Connect to a different port",
		"Connect to a remote Chrome",
	}, "Relaunch Chrome with CDP enabled")
	if err != nil {
		fmt.Fprintf(w, "error: %v\n", err)
		return ExitError
	}

	switch idx {
	case 0:
		return wizardRelaunchChrome(r, w)
	case 1:
		return wizardCustomPort(r, w)
	case 2:
		return wizardRemote(r, w)
	}

	return ExitError
}

// wizardRelaunchChrome finds Chrome and relaunches it with CDP on port 9222.
func wizardRelaunchChrome(r *bufio.Scanner, w io.Writer) int {
	chromePath := launcher.FindChrome("")
	if chromePath == "" {
		fmt.Fprintln(w, "Could not find Chrome on this system.")
		fmt.Fprintln(w, "Install Chrome or use 'hubcap setup add' with --chrome-path.")
		return ExitError
	}

	fmt.Fprintf(w, "Found Chrome: %s\n", chromePath)
	fmt.Fprintln(w, "This will launch Chrome with remote debugging on port 9222.")
	fmt.Fprintln(w, "If Chrome is already running, close it first.")
	fmt.Fprintln(w)

	ok, err := promptConfirm(r, w, "Launch Chrome now?")
	if err != nil {
		fmt.Fprintf(w, "error: %v\n", err)
		return ExitError
	}
	if !ok {
		return ExitSuccess
	}

	fmt.Fprintln(w, "Launching Chrome...")
	inst, err := launcher.Launch(launcher.LaunchOptions{
		ChromePath: chromePath,
		Port:       9222,
		Headless:   false,
	})
	if err != nil {
		fmt.Fprintf(w, "error: %v\n", err)
		return ExitError
	}

	fmt.Fprintf(w, "Chrome launched on localhost:9222 (pid %d)\n\n", inst.PID)

	name, err := promptString(r, w, "Profile name:", "default")
	if err != nil {
		fmt.Fprintf(w, "error: %v\n", err)
		return ExitError
	}
	if name == "" {
		name = "default"
	}

	return saveWizardProfile(w, name, "localhost", 9222, false)
}

// wizardCustomPort sets up a local profile on a non-default port.
func wizardCustomPort(r *bufio.Scanner, w io.Writer) int {
	portStr, err := promptString(r, w, "Debug port:", "")
	if err != nil {
		fmt.Fprintf(w, "error: %v\n", err)
		return ExitError
	}
	port, err := strconv.Atoi(portStr)
	if err != nil || port < 1 || port > 65535 {
		fmt.Fprintf(w, "Invalid port: %s\n", portStr)
		return ExitError
	}

	name, err := promptString(r, w, "Profile name:", "default")
	if err != nil {
		fmt.Fprintf(w, "error: %v\n", err)
		return ExitError
	}
	if name == "" {
		name = "default"
	}

	return saveWizardProfile(w, name, "localhost", port, false)
}

// wizardRemote sets up a profile for a remote Chrome instance.
func wizardRemote(r *bufio.Scanner, w io.Writer) int {
	host, err := promptString(r, w, "Chrome host:", "")
	if err != nil {
		fmt.Fprintf(w, "error: %v\n", err)
		return ExitError
	}
	if host == "" {
		fmt.Fprintln(w, "Host is required.")
		return ExitError
	}

	portStr, err := promptString(r, w, "Debug port:", "9222")
	if err != nil {
		fmt.Fprintf(w, "error: %v\n", err)
		return ExitError
	}
	port, err := strconv.Atoi(portStr)
	if err != nil || port < 1 || port > 65535 {
		fmt.Fprintf(w, "Invalid port: %s\n", portStr)
		return ExitError
	}

	name, err := promptString(r, w, "Profile name:", "remote")
	if err != nil {
		fmt.Fprintf(w, "error: %v\n", err)
		return ExitError
	}
	if name == "" {
		name = "remote"
	}

	return saveWizardProfile(w, name, host, port, false)
}

// saveWizardProfile saves a profile and sets it as default.
func saveWizardProfile(w io.Writer, name, host string, port int, headless bool) int {
	dir := configDir()
	pf, err := loadProfilesFile(dir)
	if err != nil {
		fmt.Fprintf(w, "error: %v\n", err)
		return ExitError
	}

	pf.Profiles[name] = Profile{
		Host:     host,
		Port:     port,
		Headless: headless,
	}
	pf.Default = name

	if err := saveProfilesFile(dir, pf); err != nil {
		fmt.Fprintf(w, "error: %v\n", err)
		return ExitError
	}

	fmt.Fprintf(w, "Profile %q saved as default.\n", name)
	fmt.Fprintln(w)
	fmt.Fprintln(w, "You're all set! Try:")
	fmt.Fprintf(w, "  hubcap tabs\n")
	fmt.Fprintf(w, "  hubcap title\n")
	return ExitSuccess
}
