package main

import (
	"bufio"
	"fmt"
	"strings"
)

func cmdPipe(cfg *Config, args []string) int {
	scanner := bufio.NewScanner(cfg.Stdin)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
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

		exitCode := info.Run(cfg, cmdArgs)
		if exitCode != ExitSuccess {
			return exitCode
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(cfg.Stderr, "error reading stdin: %v\n", err)
		return ExitError
	}

	return ExitSuccess
}

// splitArgs splits a command line into arguments, respecting quoted strings.
func splitArgs(line string) []string {
	var args []string
	var current strings.Builder
	inQuote := false
	quoteChar := byte(0)

	for i := 0; i < len(line); i++ {
		c := line[i]
		if inQuote {
			if c == quoteChar {
				inQuote = false
			} else {
				current.WriteByte(c)
			}
		} else if c == '"' || c == '\'' {
			inQuote = true
			quoteChar = c
		} else if c == ' ' || c == '\t' {
			if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
		} else {
			current.WriteByte(c)
		}
	}
	if current.Len() > 0 {
		args = append(args, current.String())
	}

	return args
}
