# 任务清单：寮突破 V2

目录：helloagents/history/2026-08/202608302247_guild_barrier_v2/

## 1. Agent 参数化

- [x] 1.1 扩展 GuildBarrierTargetRecognition，支持可选目标布局和日志前缀，保持原版默认值。
- [x] 1.2 更新自定义识别 Schema 和单元测试。

## 2. V2 Pipeline 与任务入口

- [x] 2.1 创建独立 寮突V2.json，映射全部内部节点并实现结算后状态驱动分流。
- [x] 2.2 创建 自动寮突破V2.json 并加入 interface.json 导入列表。

## 3. 测试与安全检查

- [x] 3.1 增加 V2 独立性、状态驱动参数、任务导入和原版兼容测试。
- [x] 3.2 执行 JSON、Go 测试、构建、安全检查和原版哈希复核。

## 4. 文档与交付

- [x] 4.1 更新知识库与 Changelog，迁移方案包到历史记录。
- [x] 4.2 生成本地可运行目录并记录构建信息。
