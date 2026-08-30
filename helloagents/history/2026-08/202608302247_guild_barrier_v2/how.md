# 技术设计：寮突破 V2

## 技术方案

- 从当前原版 Pipeline 机械复制节点，所有内部节点映射到 寮突V2 命名空间，图片路径保持复用。
- 成功或失败结算节点不再无目标地 JumpBack，而是进入 寮突V2-结算后列表就绪。
- 列表就绪节点识别 击败次数 后，按全破、已有进度、零进度状态分流。
- 正常路径中的纯识别节点显式降低前后延迟；点击节点保留短保护时间。
- GuildBarrierTargetRecognition 新增可选布局节点和日志前缀参数；缺省参数保持原版行为。

## 架构决策 ADR

### ADR-001：保留原版并新增独立实验 Pipeline

**上下文：** 原流程已包含多轮上游和 fork 修复，直接继续调参会扩大回归面。  
**决策：** 原版不动，V2 独立入口与命名空间，复用图片和参数化 agent。  
**替代方案：** 直接修改原版，被拒绝，因为无法快速回退且上游同步冲突更大。  
**影响：** 增加一份 Pipeline 维护成本，但可独立验证并随时切回原版。

## 安全与性能

- 不增加外部网络访问、凭据或持久化数据。
- 不用随机延迟作为性能策略；以实际界面状态决定推进时机。
- 状态轮询采用 500ms 限频，避免无约束高频识别。
- 原版文件哈希在实施前后复核。

## 测试与部署

- 单元测试覆盖默认参数兼容、V2 自定义布局、独立节点引用和任务导入。
- 运行 go test ./...、go build、JSON 解析与节点引用审计。
- 生成新的本地运行目录，由用户在模拟器中选择实验入口验证。

## 实施结果

- 新增 `resource_pack/base/pipeline/战斗/寮突V2.json`，58 个节点全部使用 `寮突V2` 命名空间。
- 新增 `tasks/自动寮突破V2.json` 并由 `interface.json` 导入；原版任务继续使用 `寮突` 入口。
- 成功与失败结算均进入 `寮突V2-结算后列表就绪`，以 500ms 限频等待真实列表状态，最长 30 秒后回退。
- `GuildBarrierTargetRecognition` 新增可选 `target_layouts` 和 `log_prefix`；原版缺省行为不变。
- V2 正常点击使用短保护时间，重复目标保护继续要求同一玩家持续至少 6 秒且至少 3 次命中。

## 验证结果

- `go test ./...`：通过。
- `go vet ./...`：通过。
- `go build`：通过，Windows agent SHA-256 为 `CF9960D640598DF57FD4B21930043AA386EF8974555148BE96EC311BD4CC97DC`。
- 全仓 110 个 JSON 文件解析：通过。
- `interface.json`、`tasks/自动寮突破V2.json`、自定义识别参数 Schema：通过。
- V2 节点、`next`、`on_error`、`recognition_node`、`node_name`、`target_layouts` 引用审计：通过。
- 原版 `寮突.json` 无 Git 差异，SHA-256 保持 `40C53876ACC9EF34120CD61358E785C7A7B8A3A3A15DA2B674A493F075BE4985`。
- 本地运行目录 `D:/coding/YYS/MaaYYs-local/runtime-334b365-guildv2-20260830` 已启动；MXU 成功导入 V2 任务并连接 MuMu 控制器。

> Pipeline 主 Schema 缺少两个相对外部 Schema 的可解析基准 ID，AJV CLI 无法直接完成组合校验。该层由 JSON 解析、Go 结构与引用测试、MXU/MaaFramework 实际加载覆盖。

> 尚未代替用户启动真实寮突破战斗；场间耗时、素材命中和异常恢复需要在游戏环境中观察多轮后确认。
