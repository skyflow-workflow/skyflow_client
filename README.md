# Skyflow Client SDK

[![License](https://img.shields.io/badge/license-Apache--2.0-blue)](LICENSE)

`skyflow_client` 是 [Skyflow](https://github.com/skyflow-workflow/skyflow_backend) 的 Go 客户端 SDK，用于：

1. 调用 Skyflow HTTP API 创建 Namespace、Activity、StateMachine 和 Execution。
2. 以外部 Activity Worker 的方式轮询并执行 `activity:*` 任务。
3. 通过 `SendTaskSuccess` / `SendTaskFailure` 上报任务结果，驱动工作流继续执行。

```text
Skyflow Server
  ↕ HTTP API
skyflow_client
  ├─ Client：API facade 与任务操作日志
  └─ ActivityWorker：注册 Activity / 轮询任务 / goroutine pool 执行
```

## 环境要求

- Go `1.24.10` 或更新版本。
- Skyflow Server 的 HTTP 地址，例如 `http://127.0.0.1:8080`。

## 安装

```bash
go get github.com/skyflow-workflow/skyflow_client
```

当前基础开发分支为 `dev/v0.1`，最近 release tag 为 `v0.1.3`。

## 快速开始

### 1. 创建 Client

默认使用进程当前的 `slog.Default()`：

```go
package main

import (
	"context"
	"log/slog"

	skyflowclient "github.com/skyflow-workflow/skyflow_client"
)

func main() {
	client, err := skyflowclient.NewClient("http://127.0.0.1:8080")
	if err != nil {
		slog.Error("create skyflow client failed", "error", err)
		return
	}
	defer client.Close()

	execution, err := client.StartExecution(context.Background(), skyflowclient.StartExecutionRequest{
		StatemachineUri: "statemachine:unittest/task_with_2_step",
		Title:           "demo execution",
		Input:           map[string]int{"x": 1, "y": 2},
	})
	if err != nil {
		slog.Error("start execution failed", "error", err)
		return
	}
	slog.Info("execution started", "execution", execution)
}
```

### 2. 自定义 logger

logger 配置在 `ActivityWorker` 层。调用方可以在已创建的 Worker 实例上修改 logger：

```go
logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
	Level: slog.LevelInfo,
}))

worker, err := skyflowclient.NewActivityWorker(client, 100, namespace)
if err != nil {
	return err
}
worker.WithLogger(logger)
```

`WithLogger` 会修改当前 Worker 实例并返回同一个实例，因此也可以按需链式调用。传入 `nil` 时，Worker 回退到当前的 `slog.Default()`。Worker 会把该 logger 注入任务请求 context，`GetActivityTask`、`SendTaskSuccess`、`SendTaskFailure` 和响应解码日志都会使用它。

### 3. 忽略无任务轮询错误日志

`ACTIVITY_TASK_NOT_FOUND` 表示当前没有可领取的任务，是 Worker 轮询中的正常结果。默认情况下，SDK 仍会输出该错误的失败日志；如果调用方希望降低日志噪音，可以在已有 Worker 实例上开启忽略开关：

```go
worker.WithIgnoreActivityTaskNotFound(true)
```

该开关只影响 `ACTIVITY_TASK_NOT_FOUND` 对应的失败日志，不影响：

- 错误本身返回给调用方。
- Worker 的轮询行为；该错误仍使用 `PollInterval` 而不是 `PollErrorInterval`。
- 其他网络或业务错误的日志输出。

传入 `false` 可恢复默认日志行为。

### 4. HTTP 超时

SDK 默认 HTTP 超时为 30 秒，避免 `GetActivityTask` 服务端重试时间超过 Kratos 默认的 2 秒。调用方可以按需覆盖：

```go
client, err := skyflowclient.NewClient(
	"http://127.0.0.1:8080",
	skyflowclient.WithHTTPTimeout(30*time.Second),
)
```

### 5. 编写 Activity

Activity 是一个普通 Go 函数，签名固定为 `func(ctx *skyflowclient.Context) error`：

```go
func addActivity(ctx *skyflowclient.Context) error {
	var input struct {
		X int `json:"x"`
		Y int `json:"y"`
	}
	if err := ctx.UnmarshalInput(&input); err != nil {
		return err
	}

	return ctx.SendTaskSuccess(input.X + input.Y)
}
```

如果任务失败，返回前调用：

```go
return ctx.SendTaskFailure("SubError", "sub result is negative")
```

### 6. 启动 Activity Worker

```go
client, err := skyflowclient.NewClient("http://127.0.0.1:8080")
if err != nil {
	return err
}
defer client.Close()

namespace := skyflowclient.NewWorkerNamespace(
	"unittest",
	"unit test namespace",
	[]string{"./statemachines"}, // 目录中的 *.json 会被注册为状态机
	skyflowclient.NewWorkerActivity("add", addActivity, "add 计算两个数的和"),
)

worker, err := skyflowclient.NewActivityWorker(client, 100, namespace)
if err != nil {
	return err
}

if err := worker.Register(); err != nil {
	return err
}
if err := worker.Run(); err != nil {
	return err
}
defer worker.Stop()
```

`NewActivityWorker` 的第三个参数是 goroutine pool 大小，应按 Activity 的最大并发处理能力设置，避免无限制占用资源。

完整示例见 [demo/demo1.go](demo/demo1.go)。

## SDK 常用 API

SDK 常用方法如下：

| 方法 | 用途 |
|------|------|
| `NewClient` | 创建 Client |
| `GetActivityTask` | 领取一个待执行的 Activity task |
| `SendTaskSuccess` | 上报任务成功结果 |
| `SendTaskFailure` | 上报任务失败信息 |
| `SendTaskHeartbeat` | 上报任务心跳 |
| `CreateOrUpdateNamespace` | 创建或更新 Namespace |
| `CreateOrUpdateActivity` | 创建或更新 Activity |
| `CreateOrUpdateStateMachine` | 创建或更新状态机 |
| `StartExecution` | 启动一个 Execution |
| `GetClient` | 返回 Kratos 生成的完整 HTTP client，用于 SDK 未封装的接口 |
| `ActivityWorker.WithLogger` | 在已有 Worker 实例上设置或重置 `*slog.Logger` |
| `ActivityWorker.WithIgnoreActivityTaskNotFound` | 控制 `ACTIVITY_TASK_NOT_FOUND` 失败日志是否输出 |

优先使用 `Client` facade。只有在 facade 尚未封装时才使用 `GetClient()`，这样日志、错误处理和后续兼容性策略才能统一生效。

## 任务操作日志

SDK 使用标准库 `log/slog` 输出结构化日志。Worker 未设置 logger 时使用 `slog.Default()`；调用 `ActivityWorker.WithLogger(logger)` 后使用调用方实例。生产环境可以显式配置 JSON handler：

```go
logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
worker.WithLogger(logger)
```

本 SDK 为以下关键操作输出开始、成功、失败日志：

- `GetActivityTask`
- `SendTaskSuccess`
- `SendTaskFailure`

当 `worker.WithIgnoreActivityTaskNotFound(true)` 开启时，`GetActivityTask` 返回 `ACTIVITY_TASK_NOT_FOUND` 时不再输出对应的失败日志。请求开始日志仍保持现有行为。

常用字段：

| 字段 | 说明 |
|------|------|
| `operation` | 操作名 |
| `activity_uri` | Activity URI |
| `task_found` | 是否领取到任务 |
| `task_token_hash` | TaskToken 的 SHA-256 前 12 位，用于关联日志且不泄漏原始 token |
| `input_length` / `output_length` / `cause_length` | 载荷长度，不打印业务内容 |
| `timeout_seconds` / `heartbeat_seconds` | 任务或请求超时信息 |
| `duration_ms` | 客户端操作耗时 |
| `error` | 结构化错误信息 |

示例：

```json
{
  "time": "2026-11-19T10:00:00+08:00",
  "level": "INFO",
  "msg": "skyflow client operation succeeded",
  "operation": "GetActivityTask",
  "activity_uri": "activity:unittest/add",
  "task_found": true,
  "task_token_hash": "6f4a1d5f6a25",
  "input_length": 13,
  "timeout_seconds": 10,
  "heartbeat_seconds": 5,
  "duration_ms": 12
}
```

> 安全约定：SDK 不在日志中输出原始 `TaskToken`、任务 `Input`、任务 `Output` 或失败 `Cause`。这些值可能包含任务凭证或业务敏感数据。

## 测试

离线单元测试：

```bash
go test ./... -run "TestClient_(GetActivityTaskOperationLogging|SendTaskSuccessOperationLogging|SendTaskFailureOperationLogging|TaskOperationErrorLogging)|TestContext_SendTaskOperationLogging|TestActivityWorker|TestCustomResponseDecoder|TestIsActivityPollNotFound"
```

编译 demo：

```bash
go test ./demo
```

集成测试默认跳过。配置真实服务地址后可执行：

```bash
SKYFLOW_TEST_SERVER=http://127.0.0.1:8080 go test ./...
```

未配置环境变量时，完整测试只运行离线用例：

```bash
go test ./...
```

## 文档

- [AGENTS.md](AGENTS.md)：AI 协作与代码质量约定。
- [docs/OPTIMIZATION.md](docs/OPTIMIZATION.md)：当前代码排查与优化建议。
- [docs/plans/2026-11-19-sdk-observability-docs.md](docs/plans/2026-11-19-sdk-observability-docs.md)：本分支开发计划。
- [docs/specs/2026-11-19-sdk-observability-docs.md](docs/specs/2026-11-19-sdk-observability-docs.md)：本变更行为与测试规格。

## License

[Apache-2.0](LICENSE)
