# Skyflow Client — Agent collaboration workflow

本文件是 `skyflow_client` 的项目级说明，供 Cursor、Codex 等 AI 辅助开发工具读取。它参考 `skyflow_backend/AGENTS.md`，并针对 SDK 仓库的公开 API 兼容性、离线测试和可观测性做了适配。

相关文档：

- 用户文档：[README.md](README.md)
- 代码排查与优化建议：[docs/OPTIMIZATION.md](docs/OPTIMIZATION.md)
- Plan 模板：`skyflow_backend/docs/plans/_TEMPLATE.md`
- Spec 模板：`skyflow_backend/docs/specs/_SPEC_TEMPLATE.md`

---

## Conventions

| Item | Rule |
|------|------|
| Current dev version | `v0.1` — referenced below as `{dev_version}`. |
| Base branch | `dev/{dev_version}` — sync before creating a new branch. |
| Plan file | `docs/plans/YYYY-MM-DD-<short-name>.md`. |
| Spec file | `docs/specs/YYYY-MM-DD-<short-name>.md`; `<short-name>` must match the Plan. |
| Feature branch | `feat/{dev_version}/<short-name>` — lowercase, hyphen-separated. |
| Bugfix branch | `bugfix/{dev_version}/<short-name>` — lowercase, hyphen-separated. |
| Release tag | `{dev_version}.<release-number>`, e.g. `v0.1.4`. Derive the next number from Git tags. |
| Go version | Keep `go.mod` aligned with the version required by this SDK. |
| Language | User-facing explanations may be Simplified Chinese; identifiers and file names stay ASCII. |

---

## Phase 0 — New requirement (branch first)

When a new feature or change set starts:

1. `git fetch origin`
2. `git checkout dev/v0.1`
3. `git pull origin dev/v0.1`
4. Create `feat/v0.1/<short-name>` or `bugfix/v0.1/<short-name>`.

If the branch already exists, check it out and continue. Do not recreate it without user agreement.

---

## Phase 1 — Plan only

Create `docs/plans/YYYY-MM-DD-<short-name>.md` before implementation. The plan must describe:

- Background and goals.
- Modules to change.
- Public API compatibility.
- Test plan.
- Rollout / documentation impact.

At this stage, do not modify implementation files such as `client.go`, `context.go`, `worker.go`, `struct.go`, or `demo/`.

---

## Phase 2 — Spec

Create a Spec when the Plan alone is not enough to implement safely. Typical cases:

- Changing exported SDK signatures.
- Changing Worker lifecycle or concurrency behavior.
- Changing HTTP decoding or error semantics.
- Adding new observable fields that could leak sensitive data.
- Adding new integration behavior that requires a real Skyflow Server.

The Spec's testing section is mandatory. Every newly exported symbol or behavior must have a unit test unless the Spec records an explicit, justified exception.

---

## Phase 3 — Implementation (gated)

Do not change implementation code until the user clearly authorizes implementation, for example:

- `开始实现`
- `执行计划`
- `Spec 已通过`

After the gate:

- Implement according to the Plan and Spec.
- Update Plan checkboxes, status, and the AI development log.
- Add or update tests.
- Run `gofmt`, `go test`, and `go vet` as relevant.
- Keep documentation in sync.

---

## Code-quality gates

### Public API compatibility

This repository is a published Go SDK. Avoid breaking:

- `NewClient`
- `Client` methods
- `Context` methods
- `ActivityWorker` and `WorkerNamespace` construction
- `ActivityWorker.WithLogger`
- `ActivityWorker.WithIgnoreActivityTaskNotFound`
- `ActivityFunction`

If a breaking change is unavoidable, create a new API path, deprecate the old one, and document migration.

### Client facade over raw HTTP client

Internal code must call `Client` facade methods instead of `GetClient()` when a facade method exists. This keeps logging, error handling, and future middleware consistent.

`GetClient()` is an escape hatch for generated APIs that are not yet wrapped, not the default path for task lifecycle operations.

### Structured `slog` logging

Use `slog` key-value fields. Do not interpolate dynamic values into messages:

```go
slog.ErrorContext(ctx, "send task failed", "operation", "SendTaskFailure", "error", err.Error())
```

Do not use `fmt.Println`, `fmt.Printf`, or `fmt.Sprintf` for operational logs.

Worker-level operational logs must use the logger configured on `ActivityWorker`:

- `worker.WithLogger(logger)` mutates an existing Worker instance and returns that instance.
- Passing `nil` resets the Worker to the current `slog.Default()`.
- The Worker injects its logger into task request contexts; Client facade methods and response decoding read it from that context.

Direct Client calls without a Worker context use `slog.Default()`.

Do not call package-level `slog.Info`, `slog.Error`, or `slog.Debug` from SDK operational paths, because that prevents callers from injecting their own logger configuration.

`ACTIVITY_TASK_NOT_FOUND` is a normal no-task polling result. Worker must use `PollInterval` for it. `ActivityWorker.WithIgnoreActivityTaskNotFound(true)` suppresses only that error's failure log; all other errors remain visible.

### Do not log task credentials or payloads

`TaskToken` authorizes task completion. `Input`, `Output`, and `Cause` may contain business data. Logs may include:

- SHA-256 hash prefix of the token.
- Payload lengths.
- Stable operation names, URIs, and error identifiers.

Do not log raw tokens or payloads by default.

### Typed input over magic strings

Activity input must be decoded with typed structs and `json.Unmarshal`, as used by `Context.UnmarshalInput`. Avoid manual map key lookup and loose type assertions.

### Errors are not silently dropped

Every returned error must be propagated or logged with structured fields. `_ =` is only allowed for documented best-effort side effects.

### Bounded worker resources

Activity execution must use the `ants` pool. Do not create one goroutine per unbounded task stream. Long-running work must honor cancellation and timeout requirements.

### Offline tests first

New unit tests must not require a live Skyflow Server. Use a fake implementation of `SkyflowV1ServiceHTTPClient` or an `httptest.Server` when needed. Put true integration tests behind a documented environment variable or build tag.

### Documentation updates

If behavior changes, update:

- `README.md` usage examples.
- Relevant Plan/Spec.
- `docs/OPTIMIZATION.md` when a known issue is fixed or newly discovered.

---

## Test commands

```bash
# Format changed files.
gofmt -w client.go context.go worker.go struct.go *_test.go

# Unit and compile tests.
go test ./...

# Static checks.
go vet ./...

# Race-sensitive worker changes.
go test -race ./...
```

If full `go test ./...` needs a live server, record that dependency in the Plan and provide a targeted offline command.

---

## Release tags

1. `git fetch --tags origin`.
2. List tags matching `v0.1.*`.
3. Find the largest release number.
4. Increment it by one.
5. Create the next local tag, e.g. `v0.1.4`.

---

## Summary flow

```text
dev/v0.1
→ feat/v0.1/<short-name> or bugfix/v0.1/<short-name>
→ docs/plans/YYYY-MM-DD-<short-name>.md
→ docs/specs/YYYY-MM-DD-<short-name>.md (when required)
→ user implementation gate
→ implementation + offline tests + docs
→ gofmt / go test / go vet
→ Plan completion record
```
