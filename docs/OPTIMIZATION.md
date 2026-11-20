# Skyflow Client 代码排查与优化建议

排查日期：2026-11-19  
分支：`feat/v0.1/sdk-observability-docs`

## 结论摘要

SDK 当前职责清晰：`Client` 封装 HTTP API，`ActivityWorker` 注册并轮询 Activity，`Context` 为 Activity 提供输入解码和结果上报。主要短板集中在：

1. 可观测性：任务操作缺少统一日志，且旧响应解码器会打印完整响应体。
2. Worker 生命周期：`stop` 存在数据竞争，轮询 goroutine 未被跟踪，`Stop` 可能无限等待。
3. 超时语义：`GetActivityTaskTimeout` 字段没有被使用，Kratos HTTP client 默认超时 2 秒。
4. 测试：多个测试依赖真实 Skyflow Server，离线 CI 不稳定。
5. 公开 API 与文档：部分请求字段未生效，状态机扫描规则较脆弱。

## 本分支已处理的优化

| 问题 | 处理 |
|------|------|
| `GetActivityTask`、`SendTaskSuccess`、`SendTaskFailure` 没有客户端操作日志 | 增加开始、成功、失败三级结构化 `slog` 日志，包含操作名、URI、token hash、载荷长度和耗时。 |
| `Context.SendTaskSuccess` / `SendTaskFailure` 直接调用 `GetClient()`，绕过 Client facade | 改为调用 `Client.SendTaskSuccess` / `Client.SendTaskFailure`，确保任务结果日志统一。 |
| `CustomResponseDecoder` 使用 `slog.Info(respBodyStr)` 打印完整响应体 | 改为 Debug 级结构化日志，仅记录状态码和响应体长度，避免泄漏 TaskToken 与业务数据。 |
| `Context.SendTaskHeartbeat` 绕过 Client facade | 改为调用 `Client.SendTaskHeartbeat`。 |
| `Client.Close()` 是空实现 | 保存 Kratos transport client 并在其 `Close()` 中释放底层连接。 |
| demo 中 `select {}` 导致 `Stop()` 不可达 | 改为监听 `SIGINT` / `SIGTERM` 后调用 `Stop()`。 |
| 集成测试默认请求 `127.0.0.1:8080`，无服务时全量测试失败 | 网络用例改为仅在设置 `SKYFLOW_TEST_SERVER` 后运行，默认跳过。 |
| SDK 日志直接使用包级 `slog`，调用方无法注入 logger | logger 移至 `ActivityWorker` 层，新增 `ActivityWorker.WithLogger` 可在已有实例上设置或重置 logger；Worker 将 logger 注入请求 context，任务操作与响应解码日志统一使用。 |
| `ACTIVITY_TASK_NOT_FOUND` 无任务轮询结果会输出错误日志，且此前走错误退避间隔 | Worker 将其按正常轮询结果处理并使用 `PollInterval`；新增 `WithIgnoreActivityTaskNotFound(true)` 可关闭该错误的失败日志。 |
| 日志行为缺少测试 | 新增 fake HTTP client 和离线日志测试。 |

## 建议后续优化

### P0：修复 Worker 生命周期与数据竞争

现状：

- `ActivityWorker.stop` 是普通 `bool`，`Run` 启动的监控 goroutine 写读没有同步。
- `Stop()` 只等待 pool 中任务结束，不等待 `monitorActivity` goroutine 退出。
- `Stop()` 每 1 秒轮询一次且没有超时；pool 卡住时会阻塞调用方。
- `workerPool` 没有 `Release()`，状态也不会进入 `Close`。

建议：

1. 为 `ActivityWorker` 保存 `context.CancelFunc` 与 `sync.WaitGroup`。
2. 使用 `context.Context` 取消监控 goroutine。
3. `Stop` 接收或内置超时，使用 `WaitGroup.Wait` 等待 poller 退出。
4. 释放 ants pool 并原子更新状态。
5. 为 `Run/Stop/monitorActivity` 增加 race 测试。

### P0：统一 `GetActivityTask` 超时语义

现状：

- `ActivityWorker.GetActivityTaskTimeout` 只赋默认值，从未传给 `GetActivityTaskRequest.TimeoutSeconds`。
- Kratos HTTP client 默认请求超时为 2 秒；如果服务端长轮询 10 秒，客户端会先超时。

建议：

1. `monitorActivity` 将 worker 的 `GetActivityTaskTimeout` 转换为请求 `TimeoutSeconds`。
2. 给 `NewClient` 增加可选配置，允许设置 HTTP timeout / transport / middleware。
3. 保证 HTTP timeout 大于任务领取 timeout，并为网络异常预留余量。
4. 用配置测试覆盖 0、负数和超过 HTTP timeout 的场景。

### P1：完善错误与 poll backoff 日志

现状：

- `monitorActivity` 将 `ActivityNotFound` / `ACTIVITY_TASK_NOT_FOUND` 当作正常轮询间隔，其他错误统一睡 `PollErrorInterval`，但未记录错误码和 retry 信息。
- pool `Invoke` 失败使用 `fmt.Println`。
- Activity 函数失败后，只有 `SendTaskFailure` 失败才会记录日志；函数失败本身没有结构化日志。

建议：

1. 在 poll 层记录 `error_code`、`activity_uri`、`poll_interval`。
2. pool `Invoke` 失败改用 `slog.ErrorContext`。
3. Activity 失败、panic、上报成功/失败分别记录清晰事件。
4. 为重复错误增加指数退避和最大退避时间。

### P1：补齐 SDK facade 与请求字段

现状：

- `StartExecutionRequest.Description` 没有映射到 proto request。
- `NewWorkerNamespace(name, description, ...)` 的 `description` 参数没有赋值给 `WorkerNamespace.Description`。
- `CreateOrUpdateStateMachine` 的描述固定为 `"statemachine from file"`。

建议：

1. 映射 `StartExecutionRequest.Description`。
2. 赋值 namespace description。
3. 为状态机描述提供显式参数或从文件元数据读取，避免硬编码。
4. 补充请求构造测试，防止字段再次遗漏。

### P1：改进状态机扫描

现状：

- 只识别 `xx.json`，`task.v1.json` 这类多段文件名会被忽略。
- 目录读取顺序依赖文件系统，注册顺序不确定。
- 使用 `path.Join`，在 Windows 路径语义下不如 `filepath.Join` 稳健。

建议：

1. 使用 `filepath.Base` 与 `filepath.Ext`。
2. 用 `strings.TrimSuffix(base, ext)` 保留完整名称。
3. 对目录 entry 排序，保证注册结果确定。
4. 记录解析到的文件数量与失败文件路径。

### P1：扩展测试分层

本分支已将网络用例改为设置 `SKYFLOW_TEST_SERVER` 后才运行。后续建议：

1. 为 `CustomResponseDecoder` / `CustomErrorDecoder` 增加 `httptest.Server` 表驱动测试。
2. 为 `WorkerNamespace.ScanStateMachinePath` 增加临时目录测试。
3. 为 `ActivityWorker` 生命周期增加 race 测试。
4. CI 分别执行默认离线 job 与配置服务地址的 integration job。

### P2：响应解码健壮性

现状：

- `CustomResponseDecoder` 先按 `CustomResponse` unmarshal；只要 JSON 是对象且没有 `success` 字段，`Success` 会是 false，可能被误判为业务失败。
- 空 `data` 时直接返回 nil，调用方目标对象可能保持零值。
- `CustomErrorDecoder` 依赖 `ReturnCode == 0 || Success` 判断成功，语义分散。

建议：

1. 使用 `map[string]json.RawMessage` 先检测 `success`、`error_code`、`data` 是否同时存在，确认是自定义 envelope 后再二次解码。
2. 对空 `data` 和 `null data` 定义明确语义。
3. 为 200、4xx、5xx、非 JSON、标准 proto JSON 响应增加表驱动测试。

### P2：清理无用类型与示例

现状：

- `ResponseMessage` 未被使用，字段 tag 仍是大写 `ErrorCode` / `Data`，与当前响应格式不一致。
- `ResourceWorker` 是空类型。

建议：

1. 删除或修正 `ResponseMessage`。
2. 删除空 `ResourceWorker`，等有实际 API 时再引入。
3. 为 demo 增加真实状态机目录参数化，避免依赖仓库内固定路径。

### P2：项目工程化

建议：

1. 增加 `Makefile`：`make fmt`、`make test`、`make vet`、`make lint`。
2. 引入 `staticcheck` 或 `golangci-lint`。
3. 增加 GitHub Actions，先跑离线测试，再跑可选 integration。
4. 在 release 前执行 `go test -race ./...`。

## 优先级建议

1. 先做 Worker 生命周期与超时语义，避免生产环境任务重复领取或无法停止。
2. 再做测试分层和 decoder 表驱动测试，降低回归风险。
3. 最后处理字段映射、扫描器与工程化清理。
