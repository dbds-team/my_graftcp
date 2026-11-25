# Features

## Purpose

跟踪本 fork 与官方 graftcp 同步、DNS 能力和静态发布的实现状态。

## Status Rules

`planned` 待实现；`in progress` 开发中；`implemented` 已实现；`deferred` 暂缓；`dropped` 放弃。

## Current Capabilities

### 官方单命令架构与 DNS / UDP

- 状态：`implemented`
- 目标/结果：同步官方 master `adc54a5`（官方 v0.9.0 之后），`local/graftcp` 同时承担 tracer 与代理，`mgraftcp` 为别名。
- 技术：ptrace/seccomp、loopback token、UDP/53 -> DNS-over-TCP；按 SOCKS5/HTTP/direct 策略转发。通用 UDP 需单独启用。
- 代码：`graftcp.c`、`ptrace-ops.c`、`local/cmd/mgraftcp`、`local/dns.go`、`local/udp.go`。
- 风险：仅 Linux；DNS 默认关闭；auto 可能回退 direct；上游解析器需支持 TCP/53；旧 graftcp-local 已移除。
- 后续：持续运行官方回归与本地 curl/SOCKS5/DNS 端到端测试，检查后续上游行为变化。

### 静态多架构发布

- 状态：`implemented`
- 目标/结果：保留旧版平铺 tar.gz 命名，增加官方 deb/rpm/FHS tar.gz，覆盖 amd64/arm64/armv7/386。
- 技术：从 go.mod 获取 Go 版本；cgo external static link；校验 ELF 架构、静态属性；发布前执行同一 tag 的测试；master push 运行四架构打包但不发布。
- 代码：`.github/workflows/{release,release-packages}.yml`、`scripts/build-release-packages.sh`、`build.sh`。
- 风险：跨平台依赖交叉 GCC/libc；GitHub CI（run 45572784038）已验证完整回归和四架构打包；正式 Release 发布尚未触发。
- 后续：选定新的 fork 版本号后推送 tag，核对远端 jobs、资产和 SHA256SUMS。

### CLI 调试与帮助

- 状态：`implemented`
- 目标/结果：`-d` / `--enable-debug-log` 同时输出 stderr 和可配置日志文件；帮助包含 DNS 示例。
- 技术/代码：复用 `local.NewLogger` 和 `io.MultiWriter`；`local/cmd/mgraftcp/{config,main,log}.go`。
- 风险：日志可能包含连接目的地址；默认日志权限 0600；重定向输出不带 ANSI 控制码。
- 后续：排查连接时使用 `--debug-log-file` 指定独立文件。

## Planned Features

无额外功能扩展；本次重点是官方同步与发布兼容。

## Technical Principles

保留官方核心实现和提交历史；fork 差异限制在发布兼容、CLI 可观测性与测试。
DNS 测试使用本地 fixture，避免以公网是否可用判断 DNS 重定向是否正确。
不伪造已删除的 graftcp-local；不宣称支持 Windows。

## Delivery Plan

本地构建/组件测试 -> 四架构静态打包校验 -> 提交推送 master 验证完整 CI -> 按用户发布安排推送 tag。
本地 seccomp 禁止 ptrace，tracer/DNS 端到端与增量构建完整测试交由严格 CI 执行。
已验证 GitHub 完整测试与四架构 artifacts；未推送新 tag，不宣称 GitHub Release 已发布。

## Change Log

- 2026-11-25：GitHub 首轮完整 CI 通过；升级发布相关 Actions 到 Node.js 24，并显式打印 DNS 端到端验证。
- 2026-11-25：同步官方 `adc54a5`；恢复静态发布兼容，增加 DNS 端到端回归、构建入口及调试日志。
