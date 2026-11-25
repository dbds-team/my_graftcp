# 构建与发布

## 上游同步基线

2026-11-25 合入 `hmgle/graftcp` 的最新 `master`：
[`adc54a59b2bf2d4be6524f2dccbd2953aae5a593`](https://github.com/hmgle/graftcp/commit/adc54a59b2bf2d4be6524f2dccbd2953aae5a593)。
最近的官方稳定版本为 v0.9.0；本仓库此前的 v1.0.x 是 fork 的版本号，不代表比官方实现更新。

官方已将旧的 C `graftcp`、Go `graftcp-local` 和 `mgraftcp` 合并：
现在 `local/graftcp` 是包含 C tracer 的 Go/cgo 单文件程序，`local/mgraftcp`
是兼容符号链接，`graftcp-local` 已移除，不再单独构建或打包。

## 自动验证

`.github/workflows/release.yml` 在 PR 和手动触发时执行构建与测试；master push
及发布由打包 workflow 调用它，验证**同一个待打包 commit/tag**：

- 按 `local/go.mod` 选择 Go（目前最低为 1.25.0），不再固定旧的 Go 1.21。
- `./build.sh` 构建统一入口；`make test` 执行 C/Go 回归测试。
- DNS 端到端测试使用本地 SOCKS5、TCP DNS 和 HTTP fixture，实际运行 curl，
  验证 UDP/53 拦截、指定 DNS 上游、DNS-over-TCP 和 SOCKS5 转发，不依赖公网。
- `scripts/test-build-config.sh` 检查增量构建和 cgo 缓存失效。

本地运行：

```sh
./build.sh
make test GO="$(cd local && go env GOROOT)/bin/go"
scripts/test-build-config.sh
actionlint  # 可选，需预先安装
```

仅支持 Linux；Windows 没有所需的 Linux ptrace/seccomp API。

## 发布入口

统一由官方引入的 `.github/workflows/release-packages.yml` 发布，避免两个 workflow
同时创建 Release。触发方式：

- master push：以 `v0.0.0-dev` 构建四架构 artifacts，**不创建 tag、不发布 Release**；
- push 一个新的 `v*` tag；
- 发布 GitHub Release（published）；
- 手动运行 Release packages，指定已有 tag；`upload_to_release=false` 只构建 artifacts。

例如（下一版本号按需要选择，不能重复使用已有 tag）：

```sh
git tag v1.1.0
git push origin v1.1.0
```

发布需要该 tag 对应的源码已包含新 workflow。手动构建旧 tag 不会自动升级旧源码。
这里只配置和验证发布流程；修改工作区不会自动推送 tag 或发布 Release。

## 产物与兼容性

架构：`linux-amd64`、`linux-arm64`、`linux-armv7`、`linux-386`。
每个架构生成：

- `graftcp-v1.1.0-linux-amd64.tar.gz`：保留本 fork 原来的文件名与平铺布局。
  含 `graftcp`、`mgraftcp -> graftcp`、文档和示例配置，不再包含 `graftcp-local`。
- `graftcp_v1.1.0_linux-amd64.tar.gz`：官方的 `usr/bin` / `usr/share/doc` 布局。
- `graftcp_v1.1.0_linux-amd64.deb` 和 `.rpm`：官方安装包。
- Release 统一提供 `SHA256SUMS`，覆盖全部安装包及两类 tar.gz。

保留本 fork 的静态链接要求：通过 cgo external linker 和 `-static` 构建，
打包前验证 ELF 架构及不存在解释器/共享库 NEEDED 项。静态不意味着不需要 Linux
内核、ptrace 权限、系统 resolver 配置或 HTTPS CA 证书。

打包脚本依赖对应架构的 GCC/libc 开发包、Go、binutils、nfpm v2.46.3：

```sh
PATH="$(cd local && go env GOROOT)/bin:$PATH" \
  scripts/build-release-packages.sh v1.1.0 linux-amd64 amd64
# ARM64: ... v1.1.0 linux-arm64 arm64 aarch64-linux-gnu-
```

仅 publish job 获得 `contents: write`，使用 `GITHUB_TOKEN` 创建 Release 和上传资产，
不需要 SSH key 或额外 PAT。已有 Release 支持 `--clobber` 重传。

## DNS 使用

官方已支持用户要求的参数（DNS 默认关闭）：

```sh
./local/graftcp --enable-dns --dns-server 1.1.1.1:53 curl https://baidu.com
# 默认代理为 127.0.0.1:1080；建议显式指定代理并禁止 direct fallback：
./local/graftcp --select_proxy_mode only_socks5 --socks5 127.0.0.1:1080 \
  --enable-dns --dns-server 1.1.1.1:53 curl --noproxy '*' https://baidu.com
# 排查：stderr 与文件同时输出日志
./local/graftcp -d --debug-log-file graftcp-debug.log \
  --enable-dns --dns-server 1.1.1.1:53 curl --noproxy '*' https://baidu.com
```

目标程序的 UDP/53 查询被重定向到内嵌 DNS listener，再以 **DNS-over-TCP**
通过当前 SOCKS5/HTTP/direct 策略连接指定解析器；代理必须能访问解析器的 **TCP/53**。
不是 DoH/DoT，也不会强制改写程序自己实现的 HTTPS DNS。`--dns-server` 单独使用不会启用 DNS。
`auto` 模式可能回退 direct；curl 的代理环境变量可能改变连接路径，故验证时使用
`--noproxy '*'`（这只禁用 curl 自己的代理，不影响 graftcp）。

## 验证边界

当前本地运行环境的 seccomp 禁止 `PTRACE_TRACEME`（errno 1），因此完整 C tracer
及 curl 端到端测试必须在允许 ptrace 的 GitHub runner 上执行。CI 不会跳过这些测试，
也不会把权限错误当成功；本地可单独执行无 ptrace 的代理/DNS组件测试与静态打包。

## 已执行的验证（2026-11-25）

合并提交 `cf6a50a` 已推送到 master，[首轮 GitHub CI](https://github.com/dbds-team/my_graftcp/actions/runs/45572784038)
的完整 C/Go/DNS 测试、增量构建回归与四架构打包/上传全部成功；publish 正确跳过。
本地也完成四架构静态打包、ELF 检查、Debian 架构标签和 16 个包的 SHA256 校验。
ARM64/ARMv7 在 QEMU 下通过版本启动检查，386 在宿主 Linux 内核原生执行通过。

随后将 checkout/setup-go/upload-artifact/download-artifact 升级到已核验的
Node.js 24 稳定版本，避免旧 Actions 的 Node.js 20 弃用警告；CI 单独打印 curl DNS
端到端测试结果，便于直接复核。正式发布未执行，需另行推送新 tag。
