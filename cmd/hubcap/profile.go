package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tomyan/hubcap/internal/chrome/launcher"
)

// Profile represents a named connection profile.
type Profile struct {
	Host             string `json:"host,omitempty"`
	Port             int    `json:"port,omitempty"`
	Timeout          string `json:"timeout,omitempty"`
	Output           string `json:"output,omitempty"`
	Target           string `json:"target,omitempty"`
	ChromePath       string `json:"chrome_path,omitempty"`
	Headless         bool   `json:"headless,omitempty"`
	ChromeDataDir    string `json:"chrome_data_dir,omitempty"`
	Ephemeral        bool   `json:"ephemeral,omitempty"`
	EphemeralTimeout string `json:"ephemeral_timeout,omitempty"`
}

// ProfilesFile represents the on-disk profiles.json structure.
type ProfilesFile struct {
	Default  string             `json:"default"`
	Profiles map[string]Profile `json:"profiles"`
}

// configDir returns the hubcap config directory.
// Uses $HUBCAP_CONFIG_DIR if set, otherwise ~/.config/hubcap.
func configDir() string {
	if dir := os.Getenv("HUBCAP_CONFIG_DIR"); dir != "" {
		return dir
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", ".hubcap")
	}
	return filepath.Join(home, ".config", "hubcap")
}

// loadProfilesFile loads profiles.json from the given directory.
// Returns an empty but valid ProfilesFile if the file doesn't exist.
func loadProfilesFile(dir string) (*ProfilesFile, error) {
	path := filepath.Join(dir, "profiles.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &ProfilesFile{Profiles: make(map[string]Profile)}, nil
		}
		return nil, fmt.Errorf("reading profiles: %w", err)
	}

	var pf ProfilesFile
	if err := json.Unmarshal(data, &pf); err != nil {
		return nil, fmt.Errorf("parsing profiles.json: %w", err)
	}
	if pf.Profiles == nil {
		pf.Profiles = make(map[string]Profile)
	}
	return &pf, nil
}

// saveProfilesFile writes profiles.json to the given directory,
// creating intermediate directories as needed.
func saveProfilesFile(dir string, pf *ProfilesFile) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}

	data, err := json.MarshalIndent(pf, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling profiles: %w", err)
	}

	path := filepath.Join(dir, "profiles.json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing profiles.json: %w", err)
	}
	return nil
}

// ephemeralSession represents a running ephemeral Chrome session.
type ephemeralSession struct {
	PID     int    `json:"pid"`
	Port    int    `json:"port"`
	DataDir string `json:"data_dir"`
	Timeout string `json:"timeout"`
}

// loadEphemeralSession loads an ephemeral session file.
func loadEphemeralSession(dir, name string) (*ephemeralSession, error) {
	path := filepath.Join(dir, "ephemeral", name+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var sess ephemeralSession
	if err := json.Unmarshal(data, &sess); err != nil {
		return nil, err
	}
	return &sess, nil
}

// saveEphemeralSession writes an ephemeral session file.
func saveEphemeralSession(dir, name string, sess *ephemeralSession) error {
	ephDir := filepath.Join(dir, "ephemeral")
	if err := os.MkdirAll(ephDir, 0755); err != nil {
		return err
	}
	data, err := json.Marshal(sess)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(ephDir, name+".json"), data, 0644)
}

// touchEphemeralSession updates the mtime of an ephemeral session file.
func touchEphemeralSession(dir, name string) {
	path := filepath.Join(dir, "ephemeral", name+".json")
	now := time.Now()
	os.Chtimes(path, now, now)
}

// ensureEphemeralRunning checks if an ephemeral profile's Chrome is running,
// and launches it if not. Returns the port to connect to.
func ensureEphemeralRunning(dir string, name string, p Profile) (int, error) {
	port := p.Port
	if port == 0 {
		port = 9222
	}

	host := p.Host
	if host == "" {
		host = "localhost"
	}

	// Check if we have an existing session
	sess, err := loadEphemeralSession(dir, name)
	if err == nil && launcher.IsPortOpen(host, sess.Port) {
		touchEphemeralSession(dir, name)
		return sess.Port, nil
	}

	// Launch Chrome
	opts := launcher.LaunchOptions{
		ChromePath: p.ChromePath,
		Port:       port,
		Headless:   p.Headless,
		DataDir:    p.ChromeDataDir,
	}

	inst, err := launcher.Launch(opts)
	if err != nil {
		return 0, fmt.Errorf("launching ephemeral Chrome: %w", err)
	}

	timeout := p.EphemeralTimeout
	if timeout == "" {
		timeout = "10m"
	}

	sess = &ephemeralSession{
		PID:     inst.PID,
		Port:    port,
		DataDir: inst.DataDir,
		Timeout: timeout,
	}
	if err := saveEphemeralSession(dir, name, sess); err != nil {
		return 0, fmt.Errorf("saving ephemeral session: %w", err)
	}

	return port, nil
}

// cleanupStaleEphemeral removes ephemeral sessions that have exceeded their timeout.
func cleanupStaleEphemeral(dir string) {
	ephDir := filepath.Join(dir, "ephemeral")
	entries, err := os.ReadDir(ephDir)
	if err != nil {
		return // no ephemeral dir — nothing to clean
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		path := filepath.Join(ephDir, entry.Name())
		info, err := entry.Info()
		if err != nil {
			continue
		}

		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		var sess ephemeralSession
		if err := json.Unmarshal(data, &sess); err != nil {
			continue
		}

		timeout, err := time.ParseDuration(sess.Timeout)
		if err != nil {
			timeout = 10 * time.Minute
		}

		// Check if session has exceeded its timeout (based on mtime)
		if time.Since(info.ModTime()) <= timeout {
			continue // still active
		}

		// Stale session — kill Chrome and clean up
		if sess.PID > 0 {
			proc, err := os.FindProcess(sess.PID)
			if err == nil {
				proc.Kill()
			}
		}

		// Remove the session file
		os.Remove(path)
	}
}
