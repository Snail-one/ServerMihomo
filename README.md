# snailproxy

snailproxy is a small command-line installer for downloading the latest mihomo release and installing it as a system service.

## Current Scope

- Linux installation with systemd
- GitHub release discovery
- Proxy-aware API access with direct fallback
- Current platform asset filtering
- Clash subscription download/update metadata under `mihomo/profiles`
- Subscription selection and generated mihomo `config.yaml`
- Embedded local mihomo offline install bundle

Windows support is present as a placeholder in the codebase, but automated builds are currently configured for Linux only.

## Local Install Bundle

Files under `resources/mihomo/` are embedded into the `snailproxy` binary at build time.

Run `go generate ./resources` before building to download the latest offline install resources:

- `geoip.metadb`
- `geosite.dat`
- `metacubexd/`
- Latest `mihomo-windows-amd64-v3-v*.zip`
- Latest `mihomo-linux-amd64-v3-v*.gz`
- Latest `mihomo-linux-arm64-v*.gz`

The mihomo package downloader uses GitHub API metadata so downloaded packages can be checked with the API `sha256` digest. If API metadata cannot be fetched, resource generation fails.
Set `MIHOMO_RELEASE_CHANNEL=alpha` during `go generate ./resources` to build the offline bundle from `Prerelease-Alpha` instead of the stable latest release.

Choose `本地安装` in the menu to install from those embedded files without network access. On Linux this releases the bundle into `/opt/mihomo` and extracts the bundled mihomo package into `/opt/mihomo/mihomo`.

Choose `验证本地 mihomo` to verify the released package with the bundled manifest sha256, compare the extracted binary with the current local mihomo file, and optionally warn if a newer release exists.

## Build

```bash
sh scripts/build.sh
```

Equivalent manual commands:

```bash
go generate ./resources
go build -o snailproxy .
```

## Run

Linux installation writes to system paths and manages a systemd service, so run it with sudo:

```bash
sudo ./snailproxy
```

To preserve proxy environment variables when using sudo:

```bash
sudo -E ./snailproxy
```
