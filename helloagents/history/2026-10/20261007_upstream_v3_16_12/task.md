# 任务清单: 同步上游 v3.16.12 并构建本地包

## 范围

- 合并 `upstream/main` 的 `57bd859`，保留 fork 寮突破 V2、自定义识别和 MXU 定制。
- 接纳竞猜、斗技、庭院导航和 Android 打包更新；不顺带修复重复攻击结界问题。
- 版本使用 `v3.16.12-local`，不恢复官方自动更新字段。
- 使用独立新目录打包，不覆盖旧包；复制现有配置，重新编译 agent。

## 执行

- [x] 检查工作树和远端，完成普通合并并解决版本冲突。
- [x] 验证 Go 测试、静态检查、JSON、任务导入及 fork 定制保留。
- [x] 更新同步记录，提交并推送当前 fork 分支。
- [x] 构建 Windows x64 包，核验资源、配置、依赖及 agent 版本。

## 验收边界

- 不运行游戏任务，不以静态检查替代实机回归。
- MXU 沿用本地 fork `57688c0`，不额外同步 MXU 上游。

## 验证记录

- `go test -count=1 ./...` 和 `go vet ./...` 通过。
- 114 个 JSON 严格解析通过，33 个接口导入均存在。
- 将 V2 导入测试中的旧文件名 `自动寮突破.json` 对齐到上游已改名的 `寮突破.json`，任务名和入口断言不变。
- V2 流程及目标识别实现保持不变；V2 SHA256 为 `BD460B1DEB73726A0883DB504D92D6993A870FA501A06931FE5EC3819965CB79`。
- 合并提交 `26d68d63` 已推送到 `origin/codex/fix-guild-barrier-target-tracking`，使用普通合并，没有强推。
- Windows x64 agent 编译成功，版本探针输出 `v3.16.12-local`；无参数的 `Missing agent identifier` 为预期结果。
- 包内 assets (7)、preset (2)、resource_pack (823)、tasks (39) 与源码逐文件哈希一致。
- maafw (30) 和 config (2) 与旧包逐文件哈希一致，MXU 与本地 fork release 二进制哈希一致。
- 离线调用 MaaFramework API 验证版本 `v5.13.0-beta.2`，base 与 official 资源加载状态均为 `3000` (Succeeded)。

## 打包来源

- 源码: `D:\coding\YYS\MaaYYs`，`codex/fix-guild-barrier-target-tracking`。
- MXU: `D:\coding\YYS\MXU`，`57688c0`；使用 `src-tauri\target\release\mxu.exe`。
- MaaFramework 与配置: `D:\coding\YYS\MaaYYs-local\runtime-da4aab8-v3.15.8-local-mxu57688c0-20260913`。
- 最终目录命名: `D:\coding\YYS\MaaYYs-local\runtime-<source commit>-v3.16.12-local-mxu57688c0-20261007`。
- 同步记录提交后重新编译 agent，确保 Go VCS 构建信息对应最终源码；不包含未验证的游戏运行结论。
