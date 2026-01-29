# Command documentation review instructions

Review each command doc one at a time against the structure below and the source code in `cmd/hubcap/main.go` and `internal/chrome/client.go`.

## How to review

1. Pick the next command alphabetically from `docs/commands/`
2. Read the doc file
3. Read the corresponding implementation in the source code (search for the command name in the `switch cmd` block in `main.go` and the relevant method in `client.go`)
4. Check each section against the criteria below
5. Fix any issues found
6. Move on to the next command

## Required sections (in order)

### 1. Title line
```
# hubcap <command> -- <one-line description>
```
or
```
# hubcap <command>

<one-line description as a separate paragraph>
```
Either format is fine. The description should be a single sentence fragment that completes "This command will...".

### 2. When to use
- Explains when and why to use this command vs alternatives
- Mentions related commands by name with backtick formatting
- Practical guidance, not just a restatement of the description

### 3. Usage
- Fenced code block with the full usage syntax
- `hubcap <command> [flags] <required-args> [optional-args]`
- Flags in brackets, required args in angle brackets
- Must match what the source code actually parses

### 4. Arguments
- Markdown table: `| Argument | Type | Required | Description |`
- One row per positional argument
- Types should be: `string`, `number`, `int`, `float`, `bool`, `duration`
- If no arguments, write "None." instead of an empty table

### 5. Flags
- Markdown table: `| Flag | Type | Default | Description |`
- One row per command-specific flag (not global flags)
- Include the `--` prefix on flag names
- If no flags, write "None." instead of an empty table

### 6. Output
- Markdown table: `| Field | Type | Description |`
- One row per JSON field in the output
- Followed by a `json` fenced code block with a realistic example
- If the command has different output modes (e.g. with/without a flag), document each separately with a subheading or note
- If the command produces no output (only side effects), say "None. Exit code 0 on success."

### 7. Errors
- Markdown table: `| Condition | Exit code | Stderr |`
- Always include these standard rows:
  - Chrome not connected (exit 2)
  - Timeout (exit 3)
- Add command-specific errors (missing args, element not found, etc.)
- The stderr messages should match what the source code actually prints

### 8. Examples
- 3-4 practical examples
- Each has a short description line followed by a fenced code block
- At least one example should show chaining with other hubcap commands using `&&`
- Examples should be realistic and useful, not just echoing the usage line

### 9. See also
- Bulleted list of related commands
- Format: `- [command](command.md) - Brief description`
- Include commands that are alternatives, complements, or commonly used together

## What to check against the source code

- **Arguments**: Does the doc match the actual argument parsing? Check `len(remaining)` checks and how args are used.
- **Flags**: Are all flags from the source listed? Check for `flag.NewFlagSet` or individual flag definitions within the command function.
- **Output fields**: Do the documented JSON fields match what the code actually marshals and prints?
- **Error messages**: Do the stderr strings in the doc match the actual `fmt.Fprintln(cfg.Stderr, ...)` calls?
- **Exit codes**: Are the correct exit codes documented for each error path?
- **Usage string**: Does the doc's usage match the usage string printed in the source on argument errors?

## Checklist per doc

- [ ] Title present with one-line description
- [ ] "When to use" section explains purpose and alternatives
- [ ] Usage syntax matches source code
- [ ] All arguments documented with correct types
- [ ] All flags documented with correct types and defaults
- [ ] Output fields match what the code produces
- [ ] JSON example is realistic
- [ ] Error conditions and exit codes are accurate
- [ ] 3-4 practical examples including at least one chain
- [ ] See also links are relevant and use correct filenames
- [ ] No references to "cdp" anywhere in the doc
