package main

import (
	"flag"
	"fmt"
	"strings"
)

// helpTokens are first-positional tokens that trigger a command's help output.
var helpTokens = map[string]bool{
	"help":   true,
	"--help": true,
	"-h":     true,
}

// dispatchCommand looks up a command by name, intercepts help requests, and
// invokes the command's Run with its arguments.
//
// A help request is the first positional being "help", "--help", or "-h" — in
// which case dispatchCommand routes to cmdHelp instead of running the command.
// This guarantees help works uniformly across every command, even ones that
// don't define their own flagset.
func dispatchCommand(cfg *Config, name string, args []string) int {
	info, ok := commands[name]
	if !ok {
		fmt.Fprintf(cfg.Stderr, "unknown command: %s\n", name)
		return ExitError
	}
	if name != "help" && isHelpRequest(args) {
		return cmdHelp(cfg, []string{name})
	}
	return info.Run(cfg, args)
}

// isHelpRequest returns true if the first arg is a help token.
func isHelpRequest(args []string) bool {
	return len(args) > 0 && helpTokens[args[0]]
}

// parsePositionals parses args for a command that accepts no flags, ensures at
// least minArgs positional values, and returns them. On unknown flag or too
// few positionals, it prints usage and returns a non-negative exit code.
//
// On success, the returned exit code is -1; callers should ignore it.
func parsePositionals(cfg *Config, cmdName string, args []string, minArgs int, usage string) ([]string, int) {
	fs := flag.NewFlagSet(cmdName, flag.ContinueOnError)
	fs.SetOutput(cfg.Stderr)
	fs.Usage = func() { fmt.Fprintln(cfg.Stderr, usage) }
	if err := fs.Parse(args); err != nil {
		return nil, ExitError
	}
	pos := fs.Args()
	if err := rejectFlagLikePositionals(cfg, pos, usage); err != nil {
		return nil, ExitError
	}
	if len(pos) < minArgs {
		fmt.Fprintln(cfg.Stderr, usage)
		return nil, ExitError
	}
	return pos, -1
}

// simple builds a Run closure for commands that accept no flags. It rejects
// unknown --flags and ensures at least minArgs positional values before
// forwarding to run with the parsed positionals.
func simple(name, usage string, minArgs int, run func(*Config, []string) int) func(*Config, []string) int {
	return func(cfg *Config, args []string) int {
		pos, code := parsePositionals(cfg, name, args, minArgs, usage)
		if code >= 0 {
			return code
		}
		return run(cfg, pos)
	}
}

// rejectFlagLikePositionals errors if any positional looks like a long flag
// (starts with "--"). This catches the case where a flag appears after a
// positional — Go's flag.Parse stops at the first non-flag, so a trailing
// "--foo" would otherwise be silently passed through as a positional.
//
// Users who need to pass a literal "--" prefixed string can use the standard
// "--" end-of-options separator.
func rejectFlagLikePositionals(cfg *Config, pos []string, usage string) error {
	for _, p := range pos {
		if strings.HasPrefix(p, "--") {
			fmt.Fprintf(cfg.Stderr, "flag provided but not defined: %s\n", p)
			fmt.Fprintln(cfg.Stderr, usage)
			return fmt.Errorf("unknown flag: %s", p)
		}
	}
	return nil
}

// reorderFlagsFirst rearranges args so flags (and their values) precede
// positionals, working around Go's flag.Parse stopping at the first non-flag
// token. Unknown --xxx tokens stay in flag position so flag.Parse can reject
// them. The "--" end-of-options separator is honored: anything after it is
// treated as positional regardless of leading dashes.
//
// fs must already have all flags registered so the helper can tell bool flags
// (no value) from value-taking flags.
func reorderFlagsFirst(fs *flag.FlagSet, args []string) []string {
	var flagArgs, posArgs []string
	i := 0
	for i < len(args) {
		a := args[i]
		if a == "--" {
			posArgs = append(posArgs, args[i+1:]...)
			break
		}
		if !strings.HasPrefix(a, "-") || a == "-" {
			posArgs = append(posArgs, a)
			i++
			continue
		}
		flagArgs = append(flagArgs, a)
		i++
		name := strings.TrimLeft(a, "-")
		valueAttached := strings.Contains(name, "=")
		if valueAttached {
			continue
		}
		name = strings.TrimLeft(strings.SplitN(a, "=", 2)[0], "-")
		f := fs.Lookup(name)
		// Unknown flag: leave as-is so Parse rejects it.
		// Bool flag: no value to consume.
		if f == nil || isBoolFlag(f) {
			continue
		}
		// Non-bool flag with a separately-passed value: pull the next token along.
		if i < len(args) {
			flagArgs = append(flagArgs, args[i])
			i++
		}
	}
	return append(flagArgs, posArgs...)
}

// isBoolFlag reports whether f is a boolean flag (no value to consume).
func isBoolFlag(f *flag.Flag) bool {
	type boolFlag interface{ IsBoolFlag() bool }
	if bf, ok := f.Value.(boolFlag); ok {
		return bf.IsBoolFlag()
	}
	return false
}
