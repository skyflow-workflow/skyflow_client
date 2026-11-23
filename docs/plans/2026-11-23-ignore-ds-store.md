# Chore: ignore macOS DS_Store files

| 字段 | 值 |
|------|-----|
| 分支 | `feat/v0.1/ignore-ds-store` |
| 创建日期 | 2026-11-23 |
| 状态 | 已完成 |

## 背景与目标

本地开发环境会生成 macOS Finder 元数据文件 `.DS_Store`。该文件不属于 SDK 源码，此前会出现在 `git status` 的未跟踪文件中，容易误提交。

目标是在 `.gitignore` 中忽略 `.DS_Store`，保持工作区干净。

## 设计方案

### 涉及模块

- [x] `.gitignore` — 增加 macOS Finder 元数据忽略规则。

### 核心逻辑

在 `.gitignore` 末尾增加：

```gitignore
# macOS Finder metadata
.DS_Store
```

### 数据库变更

无。

### API 变更

无。

## 任务拆分

- [x] 新建 `feat/v0.1/ignore-ds-store` 分支。
- [x] 在 `.gitignore` 中增加 `.DS_Store`。
- [x] 使用 `git check-ignore -v .DS_Store` 验证规则生效。

## AI 开发记录

| 日期 | 操作 | 说明 |
|------|------|------|
| 2026-11-23 | 实现 | 增加 `.DS_Store` 忽略规则并验证生效。 |

## 测试计划

- [x] `git check-ignore -v .DS_Store`
- [x] `git status --short`

## 备注

本变更为仓库配置清理，不涉及 Go 代码行为，未创建 Spec。
