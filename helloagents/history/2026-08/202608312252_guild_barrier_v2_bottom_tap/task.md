# 任务清单：寮突破 V2 结算底部点击修复

目录：`helloagents/plan/202608312252_guild_barrier_v2_bottom_tap/`

---

## 1. Pipeline 修复
- [√] 1.1 将胜负结果节点的常规点击范围收窄到结算页底部安全带，避免打开加成说明弹窗。
- [√] 1.2 将指定 display 的 Shell 补点从高度 70% 调整到 OCR 提示所在的 95%。

## 2. 测试与验证
- [√] 2.1 更新 Pipeline 回归测试，锁定常规点击安全带和 Shell 底部相对坐标。
- [√] 2.2 运行 Go、JSON、原版哈希和差异验证。

## 3. 文档与交付
- [√] 3.1 更新寮突破故障复盘、Changelog 与历史索引并归档本任务。
- [√] 3.2 构建新的独立本地运行包，保留双实例配置。

## 执行结果

- 两个 1920×1080 MuMu 实例分别在 `display 0` 与 `display 2` 手工发送 `(960,1026)` 后，均立即离开结算页并继续任务。
- `go test ./...`、`go vet ./...`、源码及运行包 JSON 解析、854 个打包文件逐一哈希和 `git diff --check` 均通过。
- 原版 `寮突.json` SHA-256 保持 `40C53876ACC9EF34120CD61358E785C7A7B8A3A3A15DA2B674A493F075BE4985`。
- 本地运行包：`D:/coding/YYS/MaaYYs-local/runtime-334b365-guildv2fix6-20260831`；复用 fix5 的 MXU、MaaFramework 和双实例配置，缓存为空。
- V2 Pipeline SHA-256：`BD460B1DEB73726A0883DB504D92D6993A870FA501A06931FE5EC3819965CB79`；Windows agent SHA-256：`FE24A79542D368427FB53D6058E9DB006F23FA20378A01D78064A2F0D30A4EA2`。
