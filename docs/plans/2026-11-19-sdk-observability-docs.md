# Feature: SDK 文档与任务操作日志

| 字段 | 值 |
|------|-----|
| 分支 | `feat/v0.1/sdk-observability-docs` |
| 创建日期 | 2026-11-19 |
| 状态 | 已完成 |

## 背景与目标

`skyflow_client` 是 Skyflow 外部 Activity Worker 的 Go SDK。仓库此前只有极简 README，缺少供 AI 协作工具读取的项目级约定；同时 `GetActivityTask`、`SendTaskSuccess`、`SendTaskFailure` 三个关键任务生命周期操作没有统一、结构化的客户端日志，排查 Worker 与 Skyflow Server 的交互链路困难。

目标：

1. 参考 `skyflow_backend/AGENTS.md`，为 client 仓库建立分支、Plan、Spec、测试与代码质量约定。
2. 补全 README、优化建议与开发文档。
3. 为任务领取与任务结果上报增加结构化 `slog` 日志，避免输出原始 TaskToken 和业务输入/输出。
4. 保证 Worker 的 `Context` 上报方法也经过 Client facade，避免绕过日志。
5. 支持调用方在已有 `ActivityWorker` 实例上注入或重置 `*slog.Logger`；未配置时回退 `slog.Default()`。
6. 支持调用方控制 `ACTIVITY_TASK_NOT_FOUND` 失败日志是否输出，并将该错误作为正常轮询结果处理。

## 设计方案

### 涉及模块

- [x] `client.go` — Client facade 与任务操作日志。
- [x] `worker.go` — Worker 持有 logger，并注入任务请求 context。
- [x] `worker.go` — 增加忽略 `ACTIVITY_TASK_NOT_FOUND` 日志开关，并按正常轮询处理该错误。
- [x] `context.go` — Worker Context 改用 Client facade 方法。
- [x] `client_test.go` — 使用 fake HTTP client 验证日志行为，不依赖真实服务。
- [x] `README.md` — SDK 快速开始、Worker 使用方式、日志说明。
- [x] `AGENTS.md` — client 仓库 AI 协作约定。
- [x] `docs/OPTIMIZATION.md` — 当前代码排查结论与优化建议。

### 核心逻辑

```text
ActivityWorker.WithLogger(logger)
  → ActivityWorker.monitorActivity / runActivity
    → 请求 context 注入 logger
  → Client.GetActivityTask
      → 记录 operation/activity_uri/timeout/request 状态
      → 记录 task_found/input_length/token_hash/耗时
  → workerPool.runActivity
      → Context.SendTaskSuccess
          → Client.SendTaskSuccess
          → 记录 output_length/token_hash/耗时
      或 Context.SendTaskFailure
          → Client.SendTaskFailure
          → 记录 error/cause_length/token_hash/耗时
```

### 数据库变更

无。

### API 变更

无。现有 Go API 签名与 HTTP 请求语义保持兼容。

## 任务拆分

- [x] 创建 `feat/v0.1/sdk-observability-docs` 分支。
- [x] 生成 client 仓库 `AGENTS.md`。
- [x] 补全 README 与优化建议文档。
- [x] 为 `GetActivityTask` 增加结构化操作日志。
- [x] 为 `SendTaskSuccess`、`SendTaskFailure` 增加结构化操作日志。
- [x] 调整 `Context` 使用 Client facade。
- [x] 让 `Client.Close()` 释放 Kratos transport client。
- [x] 修正 demo 的退出流程，使 `ActivityWorker.Stop()` 可达。
- [x] 增加不依赖真实 Skyflow Server 的日志单元测试。
- [x] 将依赖真实服务的既有测试改为 `SKYFLOW_TEST_SERVER` 显式开启。
- [x] 将 logger 移至 `ActivityWorker` 层，新增 `ActivityWorker.WithLogger` 支持修改已有实例。
- [x] Worker 将 logger 注入任务请求 context，Client 任务操作与响应解码日志统一使用该 logger。
- [x] 增加 Worker logger 修改、nil 回退 default logger、响应解码 logger 的单元测试。
- [x] 增加 `ACTIVITY_TASK_NOT_FOUND` 日志开关、默认日志行为、非目标错误不受影响和轮询分类测试。
- [x] 执行格式化、静态检查与测试。

## AI 开发记录

| 日期 | 操作 | 说明 |
|------|------|------|
| 2026-11-19 | 初始化 | 参考 backend AGENTS 创建 Plan 与 Spec。 |
| 2026-11-19 | 实现 | 增加任务操作日志、facade 调用与单元测试。 |
| 2026-11-19 | 文档 | 补全 README、AGENTS 与优化建议。 |
| 2026-11-19 | 验证 | `go test ./...`、`go test -race ./...`、`go test ./demo`、`go vet ./...` 全部通过。 |
| 2026-11-19 | 追加需求 | 支持外部传入 `*slog.Logger`，nil 时使用 `slog.Default()`。 |
| 2026-11-19 | 追加调整 | logger 移至 Worker 层，`ActivityWorker.WithLogger` 支持修改已有实例。 |
| 2026-11-19 | 追加需求 | 新增 `ActivityWorker.WithIgnoreActivityTaskNotFound`，可关闭无任务轮询错误的失败日志。 |

## 测试计划

- [x] 单元测试：`go test ./... -run "TestClient_(GetActivityTaskOperationLogging|SendTaskSuccessOperationLogging|SendTaskFailureOperationLogging|TaskOperationErrorLogging)|TestContext_SendTaskOperationLogging|TestActivityWorker|TestCustomResponseDecoder|TestIsActivityPollNotFound"`。
- [x] 编译检查：`go test ./demo`。
- [x] 集成测试：通过 `SKYFLOW_TEST_SERVER` 环境变量显式开启。

## 备注

- 日志刻意不输出原始 `TaskToken`、`Input`、`Output`，仅输出 SHA-256 前 12 位 token hash 和载荷长度，降低敏感信息泄漏风险。
- 既有集成型测试通过 `SKYFLOW_TEST_SERVER` 显式开启，未设置时自动跳过。
