# Hubcap

Go CLI for Chrome DevTools Protocol.

## Release process

1. Tag the release: `git tag v<version> && git push origin v<version>`
2. Wait for the workflow to complete: `gh run watch`
3. Done — the workflow builds binaries, publishes the GitHub release, and updates the Homebrew tap automatically

The workflow requires `HOMEBREW_APP_ID` and `HOMEBREW_APP_PRIVATE_KEY` secrets (from the `homebrew-tap-publisher` GitHub App). If not set, the Homebrew tap step is skipped.
