package main

import (
	"bufio"
	"fmt"
	"strings"
)

func cmdShell(cfg *Config, args []string) int {
	scanner := bufio.NewScanner(cfg.Stdin)

	for {
		fmt.Fprint(cfg.Stdout, "hubcap> ")

		if !scanner.Scan() {
			break
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// REPL-specific commands
		if line == ".quit" || line == ".exit" {
			return ExitSuccess
		}
		if strings.HasPrefix(line, ".target ") {
			cfg.Target = strings.TrimSpace(strings.TrimPrefix(line, ".target"))
			fmt.Fprintf(cfg.Stdout, "target set to %q\n", cfg.Target)
			continue
		}
		if strings.HasPrefix(line, ".output ") {
			cfg.Output = strings.TrimSpace(strings.TrimPrefix(line, ".output"))
			fmt.Fprintf(cfg.Stdout, "output set to %q\n", cfg.Output)
			continue
		}

		parts := splitArgs(line)
		if len(parts) == 0 {
			continue
		}

		cmdName := parts[0]
		cmdArgs := parts[1:]

		info, ok := commands[cmdName]
		if !ok {
			fmt.Fprintf(cfg.Stderr, "unknown command: %s\n", cmdName)
			continue
		}

		info.Run(cfg, cmdArgs)
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(cfg.Stderr, "error reading input: %v\n", err)
		return ExitError
	}

	return ExitSuccess
}
