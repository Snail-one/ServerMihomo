# snailproxy

snailproxy is a small command-line installer for downloading the latest mihomo release and installing it as a system service.

## Current Scope

- Linux installation with systemd
- GitHub release discovery
- Proxy-aware API access with direct fallback
- Current platform asset filtering
- Clash subscription download/update metadata under `mihomo/profiles`
- Subscription selection and generated mihomo `config.yaml`

Windows support is present as a placeholder in the codebase, but automated builds are currently configured for Linux only.

## Build

```bash
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
