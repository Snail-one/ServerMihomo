# Local Mihomo Bundle

Put files that should be embedded into the `snailproxy` binary under `resources/mihomo/`.

Before building, run:

```bash
go generate ./resources
```

This downloads `geoip.metadb` and `geosite.dat` from the MetaCubeX `meta-rules-dat` release branch into `resources/mihomo/`.
The downloaded files are generated build inputs and are ignored by git.

When the "释放本地资源包" menu option runs, every file in `resources/mihomo/` is copied directly into the mihomo data directory:

- Linux: `/opt/mihomo`
- Windows: `%ProgramData%\mihomo` when available, otherwise the system temp mihomo directory

For example, `resources/mihomo/templates/rules.yaml` will be released as `/opt/mihomo/templates/rules.yaml` on Linux.
