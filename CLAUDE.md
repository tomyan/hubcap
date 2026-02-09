# Hubcap

Go CLI for Chrome DevTools Protocol.

## Release process

1. Tag the release: `git tag v<version> && git push origin v<version>`
2. The GitHub Actions release workflow (`.github/workflows/release.yml`) builds binaries for darwin/linux/windows (amd64+arm64), creates archives, and publishes a GitHub release with checksums
3. Wait for the workflow to complete: `gh run watch`
4. Download the checksums: `gh release download v<version> --pattern 'checksums.txt' --output -`
5. Update the Homebrew tap at `/Users/tom/projects/homebrew-tap/Formula/hubcap.rb` — bump version, URLs, and sha256 values for all four platform/arch combos (darwin-arm64, darwin-amd64, linux-arm64, linux-amd64)
6. Commit and push the tap: `git -C /Users/tom/projects/homebrew-tap add Formula/hubcap.rb && git -C /Users/tom/projects/homebrew-tap commit -m "hubcap <version>" && git -C /Users/tom/projects/homebrew-tap push`
