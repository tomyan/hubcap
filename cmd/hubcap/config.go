package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// fileConfig represents the JSON config file structure.
type fileConfig struct {
	Port    *int    `json:"port,omitempty"`
	Host    *string `json:"host,omitempty"`
	Timeout *string `json:"timeout,omitempty"` // duration string, e.g. "30s"
	Output  *string `json:"output,omitempty"`
	Target  *string `json:"target,omitempty"`
}

// loadConfigFile loads a .hubcaprc file and applies it to cfg.
// It checks CWD first, then home directory. Values in the file
// override defaults but are themselves overridden by CLI flags.
func loadConfigFile(cfg *Config) {
	paths := []string{
		filepath.Join(".", ".hubcaprc"),
	}
	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(home, ".hubcaprc"))
	}

	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		var fc fileConfig
		if err := json.Unmarshal(data, &fc); err != nil {
			continue // silently skip malformed config
		}
		applyFileConfig(cfg, &fc)
		return // use first file found
	}
}

func applyFileConfig(cfg *Config, fc *fileConfig) {
	if fc.Port != nil {
		cfg.Port = *fc.Port
	}
	if fc.Host != nil {
		cfg.Host = *fc.Host
	}
	if fc.Timeout != nil {
		if d, err := time.ParseDuration(*fc.Timeout); err == nil {
			cfg.Timeout = d
		}
	}
	if fc.Output != nil {
		cfg.Output = *fc.Output
	}
	if fc.Target != nil {
		cfg.Target = *fc.Target
	}
}
