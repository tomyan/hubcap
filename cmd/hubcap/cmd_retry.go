package main

import (
	"flag"
	"fmt"
	"time"
)

func cmdRetry(cfg *Config, args []string) int {
	fs := flag.NewFlagSet("retry", flag.ContinueOnError)
	fs.SetOutput(cfg.Stderr)

	attempts := fs.Int("attempts", 3, "Maximum number of attempts")
	interval := fs.Duration("interval", 1*time.Second, "Interval between attempts")

	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return ExitSuccess
		}
		return ExitError
	}

	remaining := fs.Args()
	if len(remaining) < 1 {
		fmt.Fprintln(cfg.Stderr, "usage: hubcap retry [--attempts N] [--interval duration] <command> [args...]")
		return ExitError
	}

	cmdName := remaining[0]
	cmdArgs := remaining[1:]

	info, ok := commands[cmdName]
	if !ok {
		fmt.Fprintf(cfg.Stderr, "unknown command: %s\n", cmdName)
		return ExitError
	}

	var exitCode int
	for i := 0; i < *attempts; i++ {
		if i > 0 {
			time.Sleep(*interval)
		}
		exitCode = info.Run(cfg, cmdArgs)
		if exitCode == ExitSuccess {
			return ExitSuccess
		}
	}

	return exitCode
}
