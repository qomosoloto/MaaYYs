# 项目技术约定

## 技术栈

- MaaFramework Pipeline JSON 与 ProjectInterface V2
- Go 1.24 自定义 agent
- MXU 图形界面

## 开发约定

- 上游原版任务优先保持不变，二开实验流程使用独立入口和节点命名空间。
- Pipeline 节点引用必须能在当前资源包或通用 Pipeline 中解析。
- 手工代码修改使用最小补丁，禁止回滚无关变更。

## 错误与日志

- 自定义识别器按 Maa TaskID 隔离运行状态。
- 用户可见日志应包含任务版本、目标和结果，便于定位重复攻击或流程停留。

## 测试与流程

- Go 变更运行 go test ./... 与 go build。
- Pipeline、任务和接口文件必须通过 JSON 解析检查。
- 本地完整验证使用包含 MXU、MaaFramework、agent 和资源包的运行目录。

