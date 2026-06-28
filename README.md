---

# snailproxy

snailproxy 是一个轻量级命令行安装工具，用于下载最新的 mihomo release 并将其安装为系统服务。

---

# 当前功能范围

* Linux + systemd 安装支持
* GitHub Release 版本发现
* 支持代理访问 API（失败自动切换直连）
* 当前平台安装包自动过滤
* Clash 订阅下载 / 更新 / 修改 / 删除元数据（存储在 `mihomo/profiles`）
* 订阅 YAML 校验与 `proxies` 检查
* 选择订阅原样应用为 `mihomo config.yaml` 并重启 mihomo 服务
* 内嵌本地离线安装包

---

# 本地安装包（Local Install Bundle）

`resources/mihomo/` 目录下的文件会在构建时被嵌入到 `snailproxy` 二进制文件中。

如果需要刷新离线安装资源，在构建前运行：

```bash
go generate ./resources
```

用于下载最新的离线安装资源，包括：

* `geoip.metadb`
* `geosite.dat`
* `metacubexd/`
* 最新 `mihomo-linux-amd64-v3-v*.gz`
* 最新 `mihomo-linux-arm64-v*.gz`

普通二进制构建可以直接运行：

```bash
scripts/build.sh
```

如果需要重新下载并内嵌最新离线安装资源，运行：

```bash
scripts/build.sh --generate
```

---

# mihomo 包下载机制

安装包下载器使用 GitHub API 元数据，因此下载的文件可以通过 API 提供的 `sha256` 进行校验。

如果无法获取 GitHub API 元数据，则资源生成会失败。

---

# Alpha 版本支持

在执行 `go generate ./resources` 时，可以设置：

```bash
MIHOMO_RELEASE_CHANNEL=alpha
```

这样会从 `Prerelease-Alpha` 渠道构建离线资源，而不是稳定版 latest。

---

# 本地安装模式

在菜单中选择：

> 本地安装

会使用内嵌资源进行离线安装（无需网络）。

在 Linux 上会执行：

* 解压到 `/opt/mihomo`
* 将 mihomo 二进制释放到：

  ```
  /opt/mihomo/mihomo
  ```

---

# 本地 mihomo 校验

选择：

> 验证本地 mihomo

会执行以下操作：

* 使用内嵌 manifest 的 sha256 校验安装包
* 对比当前系统中的 mihomo 文件
* 提示是否有新版本可用

---

# 构建方法

```bash
sh scripts/build.sh
```

---

# 等价手动构建方式

```bash
go generate ./resources
go build -o snailproxy .
```

---

# 运行方式

Linux 安装会写入系统路径并管理 systemd 服务，因此需要 root 权限运行：

```bash
sudo ./snailproxy
```

---

# 保留代理环境变量运行（重要）

如果你使用代理，需要保留环境变量：

```bash
sudo -E ./snailproxy
```
