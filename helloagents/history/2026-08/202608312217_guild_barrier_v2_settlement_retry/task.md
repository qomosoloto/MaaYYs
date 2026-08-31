# 任务清单：寮突破 V2 结算补点重试修复

目录：`helloagents/plan/202608312217_guild_barrier_v2_settlement_retry/`

---

## 1. Pipeline 修复
- [√] 1.1 移除 `寮突V2-继续点击战斗结算` 的有限命中上限，避免三次补点未生效后永久失去恢复能力。
- [√] 1.2 保留 OCR 状态识别与 Android display 定向输入，并输出 display、viewport 和点击坐标诊断信息。

## 2. 测试与安全检查
- [√] 2.1 更新寮突破 Pipeline 测试，覆盖胜负共用恢复链路、补点不可耗尽和诊断输出。
- [√] 2.2 运行 Go 测试、vet、JSON 解析、原版哈希与差异检查。
- [√] 2.3 检查命令注入、敏感信息和非 V2 改动风险。

## 3. 文档与交付
- [√] 3.1 更新寮突破模块知识库与 Changelog，并迁移本方案包到历史目录。
- [√] 3.2 构建新的独立本地运行包，不覆盖现有 fix4 包。

## 执行结果

- `go test ./...`、`go vet ./...`、109 个源码 JSON 解析和 `git diff --check` 均通过。
- 两个 MuMu 目标分别解析为 `display 0` 与 `display 2`，逻辑 viewport 均为 `1920×1080`，补点坐标均为 `(960,756)`。
- 原版 `寮突.json` SHA-256 保持 `40C53876ACC9EF34120CD61358E785C7A7B8A3A3A15DA2B674A493F075BE4985`。
- 本地运行包：`D:/coding/YYS/MaaYYs-local/runtime-334b365-guildv2fix5-20260831`；运行包内 105 个 JSON 解析通过。
- Windows agent SHA-256：`D221F9293BAF9E1BDAE82568D5BC6A8F4DD238C8DE22E95884A82F414CAB0591`。
