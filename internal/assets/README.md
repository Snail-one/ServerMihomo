# Linux Mihomo Install Bundle

Put files that should be embedded into the `snailproxy` binary under `internal/assets/mihomo/`.

When the offline install resources need to be refreshed, run:

```bash
go generate ./internal/assets
```

This downloads the Linux offline install inputs into `internal/assets/mihomo/`:

- `geoip.metadb`
- `geosite.dat`
- `metacubexd/`
- Latest `mihomo-linux-amd64-v3-v*.gz`
- Latest `mihomo-linux-arm64-v*.gz`

The downloader uses the GitHub API so package downloads can be verified with API `sha256` digests. If API metadata is unavailable, resource generation fails.
Set `MIHOMO_RELEASE_CHANNEL=alpha` when running `go generate ./internal/assets` to use the `Prerelease-Alpha` development release URLs instead of the stable latest release.

The downloaded files are generated build inputs and are ignored by git.

When the "本地安装" menu option runs, every file in `internal/assets/mihomo/` is copied directly into the Linux mihomo data directory:

- Linux: `/opt/mihomo`

For example, `internal/assets/mihomo/metacubexd/index.html` will be released as `/opt/mihomo/metacubexd/index.html`, and the bundled Linux package will be extracted as `/opt/mihomo/mihomo`.
