package main

import (
	"flag"
	"fmt"
	"strings"

	"github.com/tomyan/hubcap/internal/chrome/launcher"
)

func init() {
	commands["setup"] = CommandInfo{
		Name:     "setup",
		Desc:     "Configure profiles and Chrome connection",
		Category: "Utility",
		Run:      func(cfg *Config, args []string) int { return cmdSetup(cfg, args) },
	}
}

var setupSubcommands = map[string]string{
	"list":    "List all profiles",
	"show":    "Show profile details",
	"add":     "Add a new profile",
	"edit":    "Edit an existing profile",
	"remove":  "Remove a profile",
	"default": "Get or set the default profile",
	"status":  "Check Chrome connectivity",
	"launch":  "Launch Chrome for a profile",
}

func cmdSetup(cfg *Config, args []string) int {
	if len(args) == 0 {
		return cmdSetupShow(cfg, nil)
	}

	sub := args[0]
	subArgs := args[1:]

	switch sub {
	case "list":
		return cmdSetupList(cfg)
	case "show":
		return cmdSetupShow(cfg, subArgs)
	case "add":
		return cmdSetupAdd(cfg, subArgs)
	case "edit":
		return cmdSetupEdit(cfg, subArgs)
	case "remove":
		return cmdSetupRemove(cfg, subArgs)
	case "default":
		return cmdSetupDefault(cfg, subArgs)
	case "status":
		return cmdSetupStatus(cfg, subArgs)
	case "launch":
		return cmdSetupLaunch(cfg, subArgs)
	default:
		fmt.Fprintf(cfg.Stderr, "unknown setup subcommand: %s\n", sub)
		fmt.Fprintln(cfg.Stderr, "subcommands: list, show, add, edit, remove, default, status, launch")
		return ExitError
	}
}

func cmdSetupList(cfg *Config) int {
	dir := configDir()
	pf, err := loadProfilesFile(dir)
	if err != nil {
		fmt.Fprintf(cfg.Stderr, "error: %v\n", err)
		return ExitError
	}

	type profileEntry struct {
		Name      string `json:"name"`
		Host      string `json:"host,omitempty"`
		Port      int    `json:"port,omitempty"`
		IsDefault bool   `json:"is_default,omitempty"`
	}

	var entries []profileEntry
	for name, p := range pf.Profiles {
		entries = append(entries, profileEntry{
			Name:      name,
			Host:      p.Host,
			Port:      p.Port,
			IsDefault: name == pf.Default,
		})
	}
	if entries == nil {
		entries = []profileEntry{}
	}

	return outputResult(cfg, entries)
}

func cmdSetupShow(cfg *Config, args []string) int {
	dir := configDir()
	pf, err := loadProfilesFile(dir)
	if err != nil {
		fmt.Fprintf(cfg.Stderr, "error: %v\n", err)
		return ExitError
	}

	name := ""
	if len(args) > 0 {
		name = args[0]
	}
	if name == "" {
		name = pf.Default
	}
	if name == "" {
		// Show overview: default + profile count
		type overview struct {
			Default      string `json:"default"`
			ProfileCount int    `json:"profile_count"`
		}
		return outputResult(cfg, overview{
			Default:      pf.Default,
			ProfileCount: len(pf.Profiles),
		})
	}

	p, ok := pf.Profiles[name]
	if !ok {
		fmt.Fprintf(cfg.Stderr, "error: profile %q not found\n", name)
		return ExitError
	}

	// Include the name in output
	type profileShow struct {
		Name             string `json:"name"`
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
		IsDefault        bool   `json:"is_default,omitempty"`
	}

	return outputResult(cfg, profileShow{
		Name:             name,
		Host:             p.Host,
		Port:             p.Port,
		Timeout:          p.Timeout,
		Output:           p.Output,
		Target:           p.Target,
		ChromePath:       p.ChromePath,
		Headless:         p.Headless,
		ChromeDataDir:    p.ChromeDataDir,
		Ephemeral:        p.Ephemeral,
		EphemeralTimeout: p.EphemeralTimeout,
		IsDefault:        name == pf.Default,
	})
}

func cmdSetupAdd(cfg *Config, args []string) int {
	fs := flag.NewFlagSet("setup add", flag.ContinueOnError)
	fs.SetOutput(cfg.Stderr)

	p, setDefault := registerProfileFlags(fs)

	if err := fs.Parse(args); err != nil {
		return ExitError
	}

	remaining := fs.Args()
	if len(remaining) < 1 {
		fmt.Fprintln(cfg.Stderr, "usage: hubcap setup add <name> [flags]")
		return ExitError
	}
	name := remaining[0]

	// Re-parse with name removed from args
	fs = flag.NewFlagSet("setup add", flag.ContinueOnError)
	fs.SetOutput(cfg.Stderr)
	p, setDefault = registerProfileFlags(fs)

	// Build args without the name
	var flagArgs []string
	for _, a := range args {
		if a != name {
			flagArgs = append(flagArgs, a)
		}
	}
	if err := fs.Parse(flagArgs); err != nil {
		return ExitError
	}

	dir := configDir()
	pf, err := loadProfilesFile(dir)
	if err != nil {
		fmt.Fprintf(cfg.Stderr, "error: %v\n", err)
		return ExitError
	}

	if _, exists := pf.Profiles[name]; exists {
		fmt.Fprintf(cfg.Stderr, "error: profile %q already exists (use 'setup edit' to modify)\n", name)
		return ExitError
	}

	pf.Profiles[name] = buildProfile(p)

	if *setDefault || (pf.Default == "" && len(pf.Profiles) == 1) {
		pf.Default = name
	}

	if err := saveProfilesFile(dir, pf); err != nil {
		fmt.Fprintf(cfg.Stderr, "error: %v\n", err)
		return ExitError
	}

	return outputResult(cfg, map[string]interface{}{
		"added":      name,
		"is_default": pf.Default == name,
	})
}

func cmdSetupEdit(cfg *Config, args []string) int {
	fs := flag.NewFlagSet("setup edit", flag.ContinueOnError)
	fs.SetOutput(cfg.Stderr)

	p, setDefault := registerProfileFlags(fs)

	if err := fs.Parse(args); err != nil {
		return ExitError
	}

	remaining := fs.Args()
	if len(remaining) < 1 {
		fmt.Fprintln(cfg.Stderr, "usage: hubcap setup edit <name> [flags]")
		return ExitError
	}
	name := remaining[0]

	// Re-parse with name removed
	fs = flag.NewFlagSet("setup edit", flag.ContinueOnError)
	fs.SetOutput(cfg.Stderr)
	p, setDefault = registerProfileFlags(fs)
	var flagArgs []string
	for _, a := range args {
		if a != name {
			flagArgs = append(flagArgs, a)
		}
	}
	fs.Parse(flagArgs)

	// Track which flags were set
	editFlags := map[string]bool{}
	fs.Visit(func(f *flag.Flag) {
		editFlags[f.Name] = true
	})

	dir := configDir()
	pf, err := loadProfilesFile(dir)
	if err != nil {
		fmt.Fprintf(cfg.Stderr, "error: %v\n", err)
		return ExitError
	}

	existing, ok := pf.Profiles[name]
	if !ok {
		fmt.Fprintf(cfg.Stderr, "error: profile %q not found\n", name)
		return ExitError
	}

	// Merge only explicitly-set flags
	if editFlags["host"] {
		existing.Host = *p.host
	}
	if editFlags["port"] {
		existing.Port = *p.port
	}
	if editFlags["timeout"] {
		existing.Timeout = *p.timeout
	}
	if editFlags["output"] {
		existing.Output = *p.output
	}
	if editFlags["chrome-path"] {
		existing.ChromePath = *p.chromePath
	}
	if editFlags["headless"] {
		existing.Headless = *p.headless
	}
	if editFlags["chrome-data-dir"] {
		existing.ChromeDataDir = *p.chromeDataDir
	}
	if editFlags["ephemeral"] {
		existing.Ephemeral = *p.ephemeral
	}
	if editFlags["ephemeral-timeout"] {
		existing.EphemeralTimeout = *p.ephemeralTimeout
	}

	pf.Profiles[name] = existing

	if *setDefault {
		pf.Default = name
	}

	if err := saveProfilesFile(dir, pf); err != nil {
		fmt.Fprintf(cfg.Stderr, "error: %v\n", err)
		return ExitError
	}

	return outputResult(cfg, map[string]string{"edited": name})
}

func cmdSetupRemove(cfg *Config, args []string) int {
	// Extract the name (first non-flag arg) and pass remaining to flag parser
	var name string
	var flagArgs []string
	for _, a := range args {
		if !strings.HasPrefix(a, "-") && name == "" {
			name = a
		} else {
			flagArgs = append(flagArgs, a)
		}
	}
	if name == "" {
		fmt.Fprintln(cfg.Stderr, "usage: hubcap setup remove <name> [--force]")
		return ExitError
	}

	fs := flag.NewFlagSet("setup remove", flag.ContinueOnError)
	fs.SetOutput(cfg.Stderr)
	force := fs.Bool("force", false, "Skip confirmation")

	if err := fs.Parse(flagArgs); err != nil {
		return ExitError
	}

	dir := configDir()
	pf, err := loadProfilesFile(dir)
	if err != nil {
		fmt.Fprintf(cfg.Stderr, "error: %v\n", err)
		return ExitError
	}

	if _, ok := pf.Profiles[name]; !ok {
		fmt.Fprintf(cfg.Stderr, "error: profile %q not found\n", name)
		return ExitError
	}

	if !*force {
		fmt.Fprintf(cfg.Stderr, "error: use --force to confirm removal of profile %q\n", name)
		return ExitError
	}

	delete(pf.Profiles, name)
	if pf.Default == name {
		pf.Default = ""
	}

	if err := saveProfilesFile(dir, pf); err != nil {
		fmt.Fprintf(cfg.Stderr, "error: %v\n", err)
		return ExitError
	}

	return outputResult(cfg, map[string]string{"removed": name})
}

func cmdSetupDefault(cfg *Config, args []string) int {
	dir := configDir()
	pf, err := loadProfilesFile(dir)
	if err != nil {
		fmt.Fprintf(cfg.Stderr, "error: %v\n", err)
		return ExitError
	}

	// Get mode
	if len(args) == 0 {
		return outputResult(cfg, map[string]string{"default": pf.Default})
	}

	// Set mode
	name := args[0]
	if _, ok := pf.Profiles[name]; !ok {
		fmt.Fprintf(cfg.Stderr, "error: profile %q not found\n", name)
		return ExitError
	}

	pf.Default = name
	if err := saveProfilesFile(dir, pf); err != nil {
		fmt.Fprintf(cfg.Stderr, "error: %v\n", err)
		return ExitError
	}

	return outputResult(cfg, map[string]string{"default": name})
}

func cmdSetupStatus(cfg *Config, args []string) int {
	dir := configDir()
	pf, err := loadProfilesFile(dir)
	if err != nil {
		fmt.Fprintf(cfg.Stderr, "error: %v\n", err)
		return ExitError
	}

	name := ""
	if len(args) > 0 {
		name = args[0]
	}
	if name == "" {
		name = pf.Default
	}
	if name == "" {
		fmt.Fprintln(cfg.Stderr, "error: no profile specified and no default set")
		return ExitError
	}

	p, ok := pf.Profiles[name]
	if !ok {
		fmt.Fprintf(cfg.Stderr, "error: profile %q not found\n", name)
		return ExitError
	}

	host := p.Host
	if host == "" {
		host = "localhost"
	}
	port := p.Port
	if port == 0 {
		port = 9222
	}

	connected := launcher.IsPortOpen(host, port)

	type statusResult struct {
		Profile   string `json:"profile"`
		Host      string `json:"host"`
		Port      int    `json:"port"`
		Connected bool   `json:"connected"`
	}

	return outputResult(cfg, statusResult{
		Profile:   name,
		Host:      host,
		Port:      port,
		Connected: connected,
	})
}

func cmdSetupLaunch(cfg *Config, args []string) int {
	fmt.Fprintln(cfg.Stderr, "setup launch not yet implemented")
	return ExitError
}

// profileFlags holds pointers to flag values for add/edit commands.
type profileFlags struct {
	host             *string
	port             *int
	timeout          *string
	output           *string
	chromePath       *string
	headless         *bool
	chromeDataDir    *string
	ephemeral        *bool
	ephemeralTimeout *string
}

func registerProfileFlags(fs *flag.FlagSet) (*profileFlags, *bool) {
	p := &profileFlags{
		host:             fs.String("host", "", "Chrome debug host"),
		port:             fs.Int("port", 0, "Chrome debug port"),
		timeout:          fs.String("timeout", "", "Command timeout"),
		output:           fs.String("output", "", "Output format"),
		chromePath:       fs.String("chrome-path", "", "Chrome binary path"),
		headless:         fs.Bool("headless", false, "Run headless"),
		chromeDataDir:    fs.String("chrome-data-dir", "", "Chrome data directory"),
		ephemeral:        fs.Bool("ephemeral", false, "Auto-launch and cleanup Chrome"),
		ephemeralTimeout: fs.String("ephemeral-timeout", "", "Ephemeral session timeout"),
	}
	setDefault := fs.Bool("set-default", false, "Set as default profile")
	return p, setDefault
}

func buildProfile(p *profileFlags) Profile {
	return Profile{
		Host:             *p.host,
		Port:             *p.port,
		Timeout:          *p.timeout,
		Output:           *p.output,
		ChromePath:       *p.chromePath,
		Headless:         *p.headless,
		ChromeDataDir:    *p.chromeDataDir,
		Ephemeral:        *p.ephemeral,
		EphemeralTimeout: *p.ephemeralTimeout,
	}
}

