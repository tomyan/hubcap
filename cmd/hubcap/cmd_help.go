package main

import (
	"fmt"

	"github.com/tomyan/hubcap/docs"
)

func cmdHelp(cfg *Config, args []string) int {
	if len(args) == 0 {
		// Print category-grouped command list
		fmt.Fprintln(cfg.Stdout, "hubcap - Chrome DevTools Protocol CLI")
		fmt.Fprintln(cfg.Stdout)

		for _, group := range commandsByCategory() {
			fmt.Fprintf(cfg.Stdout, "%s:\n", group.Category)
			for _, cmd := range group.Commands {
				fmt.Fprintf(cfg.Stdout, "  %-14s %s\n", cmd.Name, cmd.Desc)
			}
			fmt.Fprintln(cfg.Stdout)
		}

		fmt.Fprintln(cfg.Stdout, "Run 'hubcap help <command>' for detailed help on a command.")
		return ExitSuccess
	}

	name := args[0]
	if _, ok := commands[name]; !ok {
		fmt.Fprintf(cfg.Stderr, "unknown command: %s\n", name)
		return ExitError
	}

	data, err := docs.Commands.ReadFile("commands/" + name + ".md")
	if err != nil {
		fmt.Fprintf(cfg.Stderr, "no documentation found for %q\n", name)
		return ExitError
	}

	fmt.Fprint(cfg.Stdout, string(data))
	return ExitSuccess
}
