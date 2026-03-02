# Hubcap

Go CLI for Chrome DevTools Protocol.

## Release process

1. Tag the release: `git tag v<version> && git push origin v<version>`
2. Wait for the workflow to complete: `gh run watch`
3. Done — the workflow builds binaries, publishes the GitHub release, and updates the Homebrew tap automatically

The workflow requires a `HOMEBREW_TAP_TOKEN` secret (fine-grained PAT with `Contents: read+write` on `tomyan/homebrew-tap`). If the secret is not set, the Homebrew tap step is skipped.
