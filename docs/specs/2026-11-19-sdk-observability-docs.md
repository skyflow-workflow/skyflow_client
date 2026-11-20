# Spec: SDK 任务操作日志与协作文档

> 本 Spec 参考 `skyflow_backend/docs/specs/_SPEC_TEMPLATE.md`。第 5 节「测试规格」为强制章节。

---

## 1. 元信息

| 字段 | 值 |
|------|-----|
| 关联 Plan | `docs/plans/2026-11-19-sdk-observability-docs.md` |
| 功能分支 | `feat/v0.1/sdk-observability-docs` |
| 作者 | Codex |
| 状态 | 已通过 |
| 最后更新 | 2026-11-19 |

---

## 2. 范围与非目标

### 范围（必须做）

- 为 `(*Client).GetActivityTask` 增加开始、成功、失败日志。
- 为 `(*Client).SendTaskSuccess` 增加开始、成功、失败日志。
- 为 `(*Client).SendTaskFailure` 增加开始、成功、失败日志。
- Worker `Context` 的结果上报方法必须调用 Client facade，保证日志不会因为直接使用底层 HTTP client 而丢失。
- `Client.Close()` 释放 Kratos transport client 中的底层连接。
- `ActivityWorker` 支持调用方在已有实例上设置自定义 `*slog.Logger`；传入 `nil` 或未配置时使用当前 `slog.Default()`。
- `ActivityWorker` 支持调用方在已有实例上设置 `ACTIVITY_TASK_NOT_FOUND` 失败日志忽略开关；默认不忽略。
- 输出结构化 `slog` 字段，不把变量拼进 message。
- 增加可离线运行的日志单元测试。
- 依赖真实服务的既有测试通过 `SKYFLOW_TEST_SERVER` 显式开启，默认跳过。
- 参考 backend AGENTS 生成 client 的 AGENTS、README、优化建议等文档。

### 非目标（明确不做）

- 不修改 Skyflow HTTP API、proto 或服务端行为。
- 不引入第三方日志库或日志采集系统。
- 不改变 Activity Function 的公开签名。
- 不在本变更中重构 Worker 生命周期与并发模型；相关问题仅记录到优化建议。

---

## 3. 行为规格

### 对外行为

- `GetActivityTask`：
  - 请求开始：记录 `operation=GetActivityTask`、`activity_uri`、`timeout_seconds`。
  - 成功：记录 `task_found`、`task_token_hash`、`input_length`、响应中的 `timeout_seconds`、`heartbeat_seconds`、`duration_ms`。
  - 失败：记录 `activity_uri`、请求超时、错误与耗时。
- `SendTaskSuccess`：
  - 请求开始：记录 `operation=SendTaskSuccess`、`task_token_hash`、`output_length`。
  - 成功 / 失败：记录耗时；失败时记录结构化 error。
- `SendTaskFailure`：
  - 请求开始：记录 `operation=SendTaskFailure`、`task_token_hash`、`error`、`cause_length`。
  - 成功 / 失败：记录耗时；失败时记录结构化 error。
- 日志 message 固定，不使用 `fmt.Sprintf` 拼接动态值。
- Client 任务操作、Worker 上报失败日志以及响应解码日志必须使用 Worker 注入到请求 context 的 logger，而不是包级 `slog` 函数。
- `ActivityWorker.WithIgnoreActivityTaskNotFound(true)` 开启后，`GetActivityTask` 返回 `ACTIVITY_TASK_NOT_FOUND` 时不输出该错误对应的失败日志；其他错误仍输出。
- `ACTIVITY_TASK_NOT_FOUND` 属于无任务轮询的正常结果，Worker 使用 `PollInterval`，不使用 `PollErrorInterval`。

### 与现有系统交互

- `Context.SendTaskSuccess` / `Context.SendTaskFailure` 改为调用 `Client` 方法，而不是直接调用 `GetClient()` 返回的底层 HTTP client。
- `ActivityWorker.monitorActivity` 与 `runActivity` 的既有调用路径保持不变，从而自动获得日志。

---

## 4. 数据与兼容性

- 数据库 / 迁移：无。
- Proto / OpenAPI / 配置变更：无。
- 向后兼容：
  - `NewClient(address string)` 既有调用不变。
  - 新增 `(*ActivityWorker).WithLogger(logger *slog.Logger) *ActivityWorker`；方法修改当前实例并返回同一实例，便于链式调用。
  - 新增 `(*ActivityWorker).WithIgnoreActivityTaskNotFound(ignore bool) *ActivityWorker`；方法修改当前实例并返回同一实例。
  - Worker Context 字段不变。
  - 新增日志不改变请求或响应数据。
- 日志兼容：
  - Worker 未注入 logger 时使用当前默认 logger；直接调用 Client 且无 Worker context 时同样使用默认 logger。
  - 不打印原始 `TaskToken`、`Input`、`Output`；使用 token hash 与长度字段。

---

## 5. 测试规格（强制）

### 5.1 单元测试

| 包路径 | 文件 | 用例要点 |
|--------|------|----------|
| `github.com/skyflow-workflow/skyflow_client` | `client_logging_test.go` | `TestClient_GetActivityTaskOperationLogging`：成功领取任务时包含 operation、activity_uri、task_found、input_length、token hash，不包含原始 token/input。 |
| `github.com/skyflow-workflow/skyflow_client` | `client_logging_test.go` | `TestClient_SendTaskSuccessOperationLogging`：包含 output_length 与 token hash，不包含原始 token/output。 |
| `github.com/skyflow-workflow/skyflow_client` | `client_logging_test.go` | `TestClient_SendTaskFailureOperationLogging`：包含 error、cause_length 与 token hash，不包含原始 token/cause。 |
| `github.com/skyflow-workflow/skyflow_client` | `client_logging_test.go` | `TestClient_TaskOperationErrorLogging`：HTTP 调用失败时记录结构化 error。 |
| `github.com/skyflow-workflow/skyflow_client` | `client_logging_test.go` | `TestContext_SendTaskOperationLogging`：Worker Context 走 facade 后也能产生任务结果日志。 |
| `github.com/skyflow-workflow/skyflow_client` | `client_logging_test.go` | `TestActivityWorkerWithLoggerUpdatesExistingInstance`：已有 Worker 实例可切换自定义 logger；`nil` 回退 `slog.Default()`。 |
| `github.com/skyflow-workflow/skyflow_client` | `client_logging_test.go` | `TestActivityWorkerWithIgnoreActivityTaskNotFound`：开关开启后不输出 `ACTIVITY_TASK_NOT_FOUND` 失败日志。 |
| `github.com/skyflow-workflow/skyflow_client` | `client_logging_test.go` | `TestActivityWorkerLogsActivityTaskNotFoundByDefault`：默认仍输出该错误日志。 |
| `github.com/skyflow-workflow/skyflow_client` | `client_logging_test.go` | `TestActivityWorkerIgnoreActivityTaskNotFoundOnlyAppliesToNotFound`：开关不影响其他错误日志。 |
| `github.com/skyflow-workflow/skyflow_client` | `client_logging_test.go` | `TestIsActivityPollNotFound`：无任务轮询错误使用正常 poll 分类。 |
| `github.com/skyflow-workflow/skyflow_client` | `client_logging_test.go` | `TestCustomResponseDecoderUsesProvidedLogger`：响应解码日志使用注入 logger 且不输出原始响应体。 |

### 5.2 集成测试（可选）

- 命令：`go test ./...`。
- 依赖：本地或 `SKYFLOW_TEST_SERVER` 指向的 Skyflow Server。
- 数据准备：注册 unittest namespace、activity 与状态机，并创建可领取任务。

### 5.3 覆盖率与质量门禁

- [x] 新增导出行为有单元测试覆盖。
- [x] 单元测试不依赖网络。
- [x] `gofmt` 通过。
- [x] `go test ./... -run "TestClient_(GetActivityTaskOperationLogging|SendTaskSuccessOperationLogging|SendTaskFailureOperationLogging|TaskOperationErrorLogging)|TestContext_SendTaskOperationLogging|TestActivityWorker|TestCustomResponseDecoder|TestIsActivityPollNotFound"` 通过。
- [x] `go test ./demo` 编译通过。
- [x] `go test ./...` 通过（集成用例未配置 `SKYFLOW_TEST_SERVER` 时跳过）。
- [x] `go test -race ./...` 通过。
- [x] `go vet ./...` 通过。
- [x] 全量 `go test ./...` 在无服务环境下通过：集成用例默认跳过。

---

## 6. 安全与可观测性

- `TaskToken` 是完成任务的凭证，日志只输出 SHA-256 前 12 位 hash。
- `Input`、`Output`、`Cause` 可能包含业务敏感数据，日志只输出长度。
- 错误对象通过 `slog` 的 `error` 字段输出，不拼接动态 message。
- 所有日志均带 `operation` 字段，便于跨客户端和服务端链路筛选。

---

## 7. 验收 Checklist（与 Plan 对齐）

- [x] 三个任务操作均有开始与结果日志。
- [x] 自定义 logger 注入与 default logger 回退行为满足规格。
- [x] Worker Context 不绕过 facade。
- [x] 单元测试覆盖成功与失败路径。
- [x] README、AGENTS、优化建议文档已补充。
- [x] 公开 API 保持兼容。

---

## 8. 评审记录

| 日期 | 评审人 | 结论 | 备注 |
|------|--------|------|------|
| 2026-11-19 | Codex | 通过 | 按用户需求直接进入实现。 |
