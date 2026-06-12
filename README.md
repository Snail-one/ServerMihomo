# snailproxy

snailproxy is a small command-line installer for downloading the latest mihomo release and installing it as a system service.

## Current Scope

- Linux installation with systemd
- GitHub release discovery
- Proxy-aware API access with direct fallback
- Current platform asset filtering
- Clash subscription download/update metadata under `mihomo/profiles`
- Subscription selection and generated mihomo `config.yaml`
- Embedded local mihomo resource bundle release

Windows support is present as a placeholder in the codebase, but automated builds are currently configured for Linux only.

## Local Resource Bundle

Files under `resources/mihomo/` are embedded into the `snailproxy` binary at build time.

Run `go generate ./resources` before building to download the latest local resource files:

- `geoip.metadb`
- `geosite.dat`

Choose `释放本地资源包` in the menu to copy those embedded files directly into the mihomo data directory. On Linux this is `/opt/mihomo`, so `resources/mihomo/geosite.dat` becomes `/opt/mihomo/geosite.dat`.

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
