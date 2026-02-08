package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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
