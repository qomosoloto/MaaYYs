# 架构设计

## 总体架构

~~~mermaid
flowchart LR
    MXU[MXU / ProjectInterface] --> Task[tasks/*.json]
    Task --> Pipeline[resource_pack Pipeline]
    Pipeline --> Framework[MaaFramework]
    Pipeline --> Agent[Go custom agent]
    Framework --> Device[ADB / PlayCover]
~~~

## 核心流程

任务配置提供用户入口和 Pipeline 覆盖参数；Pipeline 负责编排识别、动作与恢复；Go agent 承担需要跨截图保存状态的自定义识别和动作。

## 重大架构决策

- 上游稳定任务与 fork 实验任务并行存在，实验任务使用独立入口和节点命名空间。

