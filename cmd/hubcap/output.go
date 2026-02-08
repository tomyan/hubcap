package main

import (
	"encoding/json"
	"fmt"
)

// TextValuer is implemented by result types that have an obvious plain-text representation.
type TextValuer interface {
	TextValue() string
}

// Implement TextValuer for scalar-ish result types.

func (r TitleResult) TextValue() string          { return r.Title }
func (r URLResult) TextValue() string             { return r.URL }
func (r TextResult) TextValue() string            { return r.Text }
func (r ValueResult) TextValue() string           { return r.Value }
func (r CountResult) TextValue() string           { return fmt.Sprintf("%d", r.Count) }
func (r ExistsResult) TextValue() string          { return fmt.Sprintf("%t", r.Exists) }
func (r VisibleResult) TextValue() string         { return fmt.Sprintf("%t", r.Visible) }
func (r SourceResult) TextValue() string          { return r.HTML }
func (r HTMLResult) TextValue() string             { return r.HTML }
func (r AttrResult) TextValue() string            { return r.Value }
func (r ClipboardReadResult) TextValue() string   { return r.Text }

func outputResult(cfg *Config, v interface{}) int {
	switch cfg.Output {
	case "json":
		enc := json.NewEncoder(cfg.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(v); err != nil {
			fmt.Fprintf(cfg.Stderr, "error: %v\n", err)
			return ExitError
		}
	case "ndjson":
		enc := json.NewEncoder(cfg.Stdout)
		if err := enc.Encode(v); err != nil {
			fmt.Fprintf(cfg.Stderr, "error: %v\n", err)
			return ExitError
		}
	case "text":
		if tv, ok := v.(TextValuer); ok {
			fmt.Fprintln(cfg.Stdout, tv.TextValue())
		} else {
			// Fall back to JSON for complex types
			enc := json.NewEncoder(cfg.Stdout)
			enc.SetIndent("", "  ")
			if err := enc.Encode(v); err != nil {
				fmt.Fprintf(cfg.Stderr, "error: %v\n", err)
				return ExitError
			}
		}
	default:
		fmt.Fprintf(cfg.Stderr, "error: unknown output format: %s\n", cfg.Output)
		return ExitError
	}
	return ExitSuccess
}
