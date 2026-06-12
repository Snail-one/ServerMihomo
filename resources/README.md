# Local Mihomo Install Bundle

Put files that should be embedded into the `snailproxy` binary under `resources/mihomo/`.

Before building, run:

```bash
go generate ./resources
```

This downloads the offline install inputs into `resources/mihomo/`:

- `geoip.metadb`
- `geosite.dat`
- `metacubexd/`
- Latest `mihomo-windows-amd64-v3-v*.zip`
- Latest `mihomo-linux-amd64-v3-v*.gz`
- Latest `mihomo-linux-arm64-v*.gz`

The downloader uses the GitHub API so package downloads can be verified with API `sha256` digests. If API metadata is unavailable, resource generation fails.
Set `MIHOMO_RELEASE_CHANNEL=alpha` when running `go generate ./resources` to use the `Prerelease-Alpha` development release URLs instead of the stable latest release.
The resolved package names and sha256 values are written to `resources/mihomo/packages/manifest.json` and embedded with the offline bundle.

The downloaded files are generated build inputs and are ignored by git.

When the "本地安装" menu option runs, every file in `resources/mihomo/` is copied directly into the mihomo data directory:

- Linux: `/opt/mihomo`
- Windows: `%ProgramData%\mihomo` when available, otherwise the system temp mihomo directory

For example, `resources/mihomo/metacubexd/index.html` will be released as `/opt/mihomo/metacubexd/index.html` on Linux, and the bundled Linux package will be extracted as `/opt/mihomo/mihomo`.
