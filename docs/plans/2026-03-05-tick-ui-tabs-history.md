# Tabs + History Refresh Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 为 TUI 增加 Tab 循环切换，并重做 History 的导航/渲染：取消 left/right；up/down 触顶/触底自动滚动 1 天且不超过今天；History 左右布局右侧只显示选中日期任务，延误按统一规则标红；整体在工作区内上下左右居中；日期显示 MM-DD。

**Architecture:** 在 UI 层引入一个 History 右侧渲染器与“居中到盒子”的通用 helper；在 app/store 层新增一个按创建日查询 active 任务的方法，用于把“仍未完成且延误”的任务显示在其创建日。

**Tech Stack:** Go, Bubble Tea, Bubbles key/list, Lipgloss, termenv, SQLite (modernc.org/sqlite)

---

### Task 1: 增加 store/app 查询 - 按创建日列出 active 任务

**Files:**
- Modify: `internal/store/store.go`
- Modify: `internal/store/sqlite/sqlite.go`
- Modify: `internal/app/app.go`
- Test: `internal/store/sqlite/sqlite_test.go`

**Step 1: 写一个失败的 store 测试**

在 `internal/store/sqlite/sqlite_test.go` 新增测试：
- 创建 2 个 active task，created_day 不同；
- 调用 `ListActiveByCreatedDay(day)` 只返回该日创建的 active。

**Step 2: 运行测试确认失败**

Run: `go test ./internal/store/sqlite -count=1`
Expected: FAIL（缺少接口/实现）

**Step 3: 最小实现接口与 sqlite 查询**

- 在 `internal/store/store.go` 增加方法签名：
  - `ListActiveByCreatedDay(ctx context.Context, day domain.Day) ([]domain.Task, error)`
- 在 `internal/store/sqlite/sqlite.go` 实现：
  - SQL：`WHERE status = 'active' AND created_day = ? ORDER BY id ASC`
- 在 `internal/app/app.go` 增加透传方法（例如 `HistoryActiveCreatedByDay`），仅包装 store 错误上下文。

**Step 4: 运行测试确认通过**

Run: `go test ./internal/store/sqlite -count=1`
Expected: PASS

**Step 5: 提交（可选，需人工确认）**

```bash
git add internal/store/store.go internal/store/sqlite/sqlite.go internal/app/app.go internal/store/sqlite/sqlite_test.go
git commit -m "store: list active tasks by created day"
```

---

### Task 2: UI keymap - 移除 History left/right，并新增 Tab 切换

**Files:**
- Modify: `internal/ui/keys.go`
- Modify: `internal/ui/styles.go`
- Modify: `internal/ui/model.go`
- Test: `internal/ui/model_history_test.go`

**Step 1: 写失败测试（Tab 循环切换）**

在 `internal/ui/model_history_test.go`（或新文件）增加：
- 初始 Today，按 `Tab` -> Upcoming，按 `Tab` -> History，按 `Tab` -> Today。

Run: `go test ./internal/ui -run TestModel_TabCyclesViews -count=1`
Expected: FAIL（尚无 Tab 行为）

**Step 2: 写失败测试（History 取消 left/right）**

- 验证按 `left/right` 或 `h/l` 不再移动窗口（historyFrom/historyTo 不变）。

**Step 3: 写失败测试（History up/down 触顶/触底 auto-roll 一天 + clamp today）**

覆盖：
- `historyIndex==0` 再按 up：窗口整体 -1 天，并触发 stats 刷新；
- `historyIndex==6` 且 `historyTo<today` 再按 down：窗口整体 +1 天并触发 stats 刷新；
- `historyIndex==6` 且 `historyTo==today` 再按 down：无变化。

**Step 4: 实现 keymap 与 Update 逻辑**

- `internal/ui/keys.go`
  - 删除 `HistoryLeft/HistoryRight` binding。
  - 新增 `Tab` binding（keys: `tab`）。
- `internal/ui/styles.go`
  - 更新 help 文案（History 不再展示 left/right/h/l）。
- `internal/ui/model.go`
  - 在 `tea.KeyMsg` 分支中处理 `Tab`：循环切换 view，并触发对应刷新（Today/Upcoming -> `cmdRefreshActive()`；History -> 初始化窗口并 `cmdRefreshHistoryWithStats()`）。
  - 调整 History up/down 行为为“触顶/触底滚动 1 天 + clamp today”。

**Step 5: 运行 UI 测试**

Run: `go test ./internal/ui -count=1`
Expected: PASS

**Step 6: 提交（可选，需人工确认）**

```bash
git add internal/ui/keys.go internal/ui/styles.go internal/ui/model.go internal/ui/model_history_test.go
git commit -m "ui: add tab cycling and simplify history navigation"
```

---

### Task 3: History 右侧内容 - `[✓]/[✗]/[ ]` 行格式 + 延误标红

**Files:**
- Modify: `internal/ui/model.go`
- Modify: `internal/ui/views.go`
- Modify: `internal/ui/styles.go`
- Test: `internal/ui/model_history_test.go`

**Step 1: 写失败测试（渲染行格式）**

- 构造：
  - 选中日有 1 条 done、1 条 abandoned；
  - 同时构造 1 条 active 且 due < today、created_day==选中日（未完成延误）。
- 断言 History body（或整体 View）包含：
  - `[✓] done-title`
  - `[✗] ab-title`
  - `[ ] delayed-active-title`

**Step 2: 写失败测试（日期显示 MM-DD）**

- 断言左侧日期行不包含年份（例如不匹配 `2026-`），而包含 `03-04` 这种格式。

**Step 3: 实现数据获取与拼装**

- `internal/ui/model.go`
  - 扩展 history refresh message：加入 `activeCreated []domain.Task`（或单独字段）。
  - `cmdRefreshHistorySelectedDay` 与 `cmdRefreshHistoryWithStats`：
    - 并行/顺序调用：done/abandoned + 新增 app 方法（按 created_day 查 active）；
    - UI 侧过滤出 `StatusActive && DueDay < currentDay()`。
  - 保持 stats 刷新只在“进入 History”与“窗口滚动”时触发。

**Step 4: 实现渲染（仍是左右布局）**

- `internal/ui/views.go`
  - 左侧输出 7 行 `MM-DD`。
  - 右侧输出任务行列表（完成 -> 放弃 -> 未完成延误）。
  - 延误标红：
    - done: `DueDay < DoneDay`
    - abandoned: `DueDay < AbandonedDay`
    - active-delayed: 已过滤即为延误
  - 任务行前缀：
    - done: `[✓]`
    - abandoned: `[✗]`
    - active-delayed: `[ ]`

**Step 5: 运行测试确认通过**

Run: `go test ./internal/ui -count=1`
Expected: PASS

**Step 6: 提交（可选，需人工确认）**

```bash
git add internal/ui/model.go internal/ui/views.go internal/ui/styles.go internal/ui/model_history_test.go
git commit -m "ui: redesign history body with status brackets and delayed highlighting"
```

---

### Task 4: History 内容块在工作区内上下左右居中 + 中间竖分割线

**Files:**
- Modify: `internal/ui/styles.go`
- Modify: `internal/ui/views.go`
- Test: `internal/ui/model_test.go`

**Step 1: 写失败测试（居中）**

- 固定窗口大小（例如 80x24），让 History 右侧内容较短；
- 断言渲染结果在 workspace 内出现左右 padding，并且上下也有空行（可通过定位分割线/左侧日期块的起始行来验证相对位置）。

**Step 2: 实现一个“居中到盒子”的 helper**

- 在 `internal/ui/styles.go` 新增 helper（伪码）：
  - `centerInBox(s string, boxW, boxH int) string`
  - 使用 ANSI-aware 宽度计算（`ansi.StringWidth`），按行补空格实现水平居中；按行数补空行实现垂直居中。

**Step 3: 在 History 渲染时使用 helper**

- `internal/ui/views.go`
  - 先生成 contentBlock（左列 + ` | ` + 右列）
  - 将其 `centerInBox(contentBlock, workspaceWidth, innerHeight)`
  - 注意：innerHeight/workspaceWidth 从 Model 的窗口尺寸与 sheet margin 计算，保持与 `Model.View()` 的 sizing 一致。

**Step 4: 运行测试确认通过**

Run: `go test ./internal/ui -count=1`
Expected: PASS

**Step 5: 提交（可选，需人工确认）**

```bash
git add internal/ui/styles.go internal/ui/views.go internal/ui/model_test.go
git commit -m "ui: center history content block and add vertical separator"
```

---

### Task 5: 全量验证

**Step 1: 跑全量测试**

Run: `go test ./... -count=1`
Expected: PASS

**Step 2: 手动验收（可选）**

Run: `go run ./cmd/tick`

检查：
- `Tab` 循环切换三视图。
- History up/down 滚动窗口与 clamp today。
- History 左侧 `MM-DD`，右侧 `[✓]/[✗]/[ ]`；延误任务标红。
- 内容块在 workspace 内上下左右居中，且左右有竖分割线。
