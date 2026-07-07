---

# snailproxy

snailproxy 是一个轻量级命令行安装工具，用于下载最新的 mihomo release 并将其安装为系统服务。

---

# 当前功能范围

* 仅支持 Linux + systemd 安装
* GitHub Release 版本发现
* 支持代理访问 API（失败自动切换直连）
* Linux 架构安装包自动过滤
* Clash 订阅下载 / 更新 / 修改 / 删除元数据（存储在 `mihomo/profiles`）
* 订阅 YAML 校验与 `proxies` 检查
* 选择订阅原样应用为 `mihomo config.yaml` 并重启 mihomo 服务
* 内嵌本地离线安装包

---

# 下载最新 Release

一键下载当前 Linux 架构对应的最新 `snailproxy`：

```bash
arch="$(case "$(uname -m)" in x86_64|amd64) echo amd64 ;; aarch64|arm64) echo arm64 ;; *) echo "不支持的架构: $(uname -m)" >&2; exit 1 ;; esac)" && curl -fL "https://github.com/Snail-one/ServerMihomo/releases/latest/download/snailproxy-linux-${arch}" -o snailproxy && chmod +x snailproxy
```

下载后运行：

```bash
sudo ./snailproxy
```

---

# 本地安装包（Local Install Bundle）

`internal/assets/mihomo/` 目录下的文件会在构建时被嵌入到 `snailproxy` 二进制文件中。

如果需要刷新离线安装资源，在构建前运行：

```bash
go generate ./internal/assets
```

用于下载最新的离线安装资源，包括：

* `geoip.metadb`
* `geosite.dat`
* `metacubexd/`
* 最新 `mihomo-linux-amd64-v3-v*.gz`
* 最新 `mihomo-linux-arm64-v*.gz`

构建 Linux 二进制并重新下载内嵌离线安装资源：

```bash
sh build.sh
```

如果只需要构建 Linux 二进制、跳过资源下载，运行：

```bash
sh build-only.sh
```

---

# mihomo 包下载机制

安装包下载器使用 GitHub API 元数据，因此下载的文件可以通过 API 提供的 `sha256` 进行校验。

如果无法获取 GitHub API 元数据，则资源生成会失败。

---

# Alpha 版本支持

在执行 `go generate ./internal/assets` 时，可以设置：

```bash
MIHOMO_RELEASE_CHANNEL=alpha
```

这样会从 `Prerelease-Alpha` 渠道构建离线资源，而不是稳定版 latest。

---

# 主菜单

```text
1. 安装与更新
2. 订阅管理
3. mihomo 服务与代理
4. 卸载
0. 退出
```

安装菜单：

```text
1. 本地安装
2. 在线下载并安装 mihomo
3. 安装/更新 systemd 服务
0. 返回
```

---

# 代码组织

应用内核位于 `internal/app`，只负责启动、版本参数、sudo 检查和由 registry 生成的主菜单循环。功能以编译期内置插件形式放在 `internal/features/<feature>`，每个功能包持有自己的菜单、prompt 和业务流程；默认内置功能由 `internal/features.Default()` 集中注册。插件接口和注册表定义在 `internal/feature`，通用 CLI 输入、确认和菜单渲染位于 `internal/terminal`。mihomo 订阅、配置和本地存储模型位于 `internal/domain/mihomo`；GitHub、下载器、archive、progress、platform 等外部系统能力集中在 `internal/infra/*`；内嵌本地安装资源位于 `internal/assets`。

---

# 本地安装模式

在菜单中选择：

> 安装 -> 本地安装

会使用内嵌资源进行离线安装（无需网络）。

在 Linux 上会执行：

* 解压到 `/opt/mihomo`
* 将 mihomo 二进制释放到：

  ```
  /opt/mihomo/mihomo
  ```

# 构建方法

```bash
sh build.sh
```

构建脚本会固定 `GOOS=linux`，默认使用当前或环境变量指定的 `GOARCH`。

---

# 等价手动构建方式

```bash
go generate ./internal/assets
GOOS=linux go build -o snailproxy ./cmd/snailproxy
```

---

# 运行方式

Linux 安装会写入系统路径并管理 systemd 服务，因此需要 root 权限运行：

```bash
sudo ./snailproxy
```

查看 snailproxy 程序版本不需要进入菜单：

```bash
./snailproxy --version
```

---

# 保留代理环境变量运行（重要）

如果你使用代理，需要保留环境变量：

```bash
sudo -E ./snailproxy
```
