package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
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
	"stop":    "Stop Chrome for a profile",
}

func cmdSetup(cfg *Config, args []string) int {
	if err := ensureDefaultProfile(configDir()); err != nil {
		fmt.Fprintf(cfg.Stderr, "error: %v\n", err)
		return ExitError
	}

	if len(args) == 0 {
		return cmdSetupDashboard(cfg)
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
	case "stop":
		return cmdSetupStop(cfg, subArgs)
	default:
		fmt.Fprintf(cfg.Stderr, "unknown setup subcommand: %s\n", sub)
		fmt.Fprintln(cfg.Stderr, "subcommands: list, show, add, edit, remove, default, status, launch, stop")
		return ExitError
	}
}

func cmdSetupDashboard(cfg *Config) int {
	dir := configDir()
	pf, err := loadProfilesFile(dir)
	if err != nil {
		fmt.Fprintf(cfg.Stderr, "error: %v\n", err)
		return ExitError
	}

	if len(pf.Profiles) == 0 {
		printFirstRunMessage(cfg)
		return ExitSuccess
	}

	printProfileTable(cfg, pf)
	return ExitSuccess
}

func printFirstRunMessage(cfg *Config) {
	w := cfg.Stdout
	fmt.Fprintln(w, "Default profile created.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Quick start:")
	fmt.Fprintln(w, "  hubcap setup launch           Start Chrome")
	fmt.Fprintln(w, "  hubcap tabs                   List open tabs")
	fmt.Fprintln(w, "  hubcap goto https://example   Navigate to a URL")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Run 'hubcap setup' to see profile status.")
}

func printProfileTable(cfg *Config, pf *ProfilesFile) {
	portChecker := cfg.PortChecker
	if portChecker == nil {
		portChecker = launcher.IsPortOpen
	}

	type row struct {
		marker string
		name   string
		host   string
		port   int
		status string
	}

	var rows []row
	for name, p := range pf.Profiles {
		host := p.Host
		if host == "" {
			host = "localhost"
		}
		port := p.Port
		if port == 0 {
			port = 9222
		}
		status := "not connected"
		if portChecker(host, port) {
			status = "connected"
		}
		marker := " "
		if name == pf.Default {
			marker = "*"
		}
		rows = append(rows, row{marker, name, host, port, status})
	}

	sort.Slice(rows, func(i, j int) bool {
		return rows[i].name < rows[j].name
	})

	w := cfg.Stdout
	fmt.Fprintf(w, "  %-12s %-16s %-6s %s\n", "PROFILE", "HOST", "PORT", "STATUS")
	for _, r := range rows {
		fmt.Fprintf(w, "%s %-12s %-16s %-6d %s\n", r.marker, r.name, r.host, r.port, r.status)
	}

	printSubcommandGuide(w)
}

func printSubcommandGuide(w io.Writer) {
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Manage profiles:")
	fmt.Fprintln(w, "  hubcap setup add <name>     Add a profile")
	fmt.Fprintln(w, "  hubcap setup edit <name>    Edit a profile")
	fmt.Fprintln(w, "  hubcap setup remove <name>  Remove a profile")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Start/stop Chrome:")
	fmt.Fprintln(w, "  hubcap setup launch [name]  Start Chrome for a profile")
	fmt.Fprintln(w, "  hubcap setup stop [name]    Stop Chrome for a profile")
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

	// Check if already running
	if launcher.IsPortOpen(host, port) {
		fmt.Fprintf(cfg.Stderr, "Chrome already running on %s:%d\n", host, port)
		return ExitError
	}

	opts := launcher.LaunchOptions{
		ChromePath: p.ChromePath,
		Port:       port,
		Headless:   p.Headless,
		DataDir:    p.ChromeDataDir,
	}

	inst, err := launcher.Launch(opts)
	if err != nil {
		fmt.Fprintf(cfg.Stderr, "error: %v\n", err)
		return ExitError
	}

	type launchResult struct {
		Profile string `json:"profile"`
		Host    string `json:"host"`
		Port    int    `json:"port"`
		PID     int    `json:"pid"`
		DataDir string `json:"data_dir"`
	}

	// Save session file so 'setup stop' can find the process
	sess := &ephemeralSession{
		PID:     inst.PID,
		Port:    port,
		DataDir: inst.DataDir,
		Timeout: "0",
	}
	saveEphemeralSession(dir, name, sess)

	return outputResult(cfg, launchResult{
		Profile: name,
		Host:    host,
		Port:    port,
		PID:     inst.PID,
		DataDir: inst.DataDir,
	})
}

func cmdSetupStop(cfg *Config, args []string) int {
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

	if _, ok := pf.Profiles[name]; !ok {
		fmt.Fprintf(cfg.Stderr, "error: profile %q not found\n", name)
		return ExitError
	}

	sess, err := loadEphemeralSession(dir, name)
	if err != nil {
		fmt.Fprintf(cfg.Stderr, "error: Chrome is not running for profile %q\n", name)
		return ExitError
	}

	// Kill the Chrome process
	if sess.PID > 0 {
		proc, err := os.FindProcess(sess.PID)
		if err == nil {
			proc.Kill()
		}
	}

	// Remove the session file
	removeEphemeralSession(dir, name)

	return outputResult(cfg, map[string]interface{}{
		"stopped": name,
		"pid":     sess.PID,
	})
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

