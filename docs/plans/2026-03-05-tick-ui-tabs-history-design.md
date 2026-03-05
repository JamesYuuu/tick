## 背景与目标

本次迭代聚焦在 tick 的 TUI（Bubble Tea/Lipgloss）顶部 Tab 切换与 History 视图的交互/渲染。

目标：
- 使用 `Tab` 在 Today / Upcoming / History 之间循环切换（保留 `1/2/3` 直达）。
- History 视图取消 left/right（含方向键与 h/l）；改为 up/down 在触顶/触底时自动滚动日期窗口（每次滚动 1 天），且不能滚动到“今天”之后。
- History 视图仍采用左右布局，但右侧只显示“选中日期”的任务完成情况，避免视觉上把 Done/Abandoned 跟在日期后面。
- 左右中间增加竖向分割线。
- 左列+分割线+右列整体在工作区内水平与垂直居中。

非目标：
- 不区分主动/被动延误；只按统一规则标红。
- 不引入新的事件日志（如 postpone 历史）。

---

## 交互与按键

### 视图切换

- `Tab`：Today -> Upcoming -> History 循环切换。
- `1/2/3`：保留直达 Today/Upcoming/History。

约束：
- 当处于添加任务输入态（`adding=true`）时，`Tab` 不触发切换（避免干扰输入组件行为）。

### History 导航（窗口 7 天）

内部状态：
- `historyFrom` / `historyTo`：窗口起止（始终相差 6 天，共 7 天）。
- `historyIndex`：选中行（0..6）。

行为：
- 移除 `HistoryLeft/HistoryRight` 所有绑定与 help 文案。
- `up/k`：
  - 若 `historyIndex > 0`：`historyIndex--`，刷新“选中日期”右侧数据（done/abandoned/active-delayed）。
  - 若 `historyIndex == 0`：窗口整体回滚 1 天：`historyFrom--`、`historyTo--`（`historyIndex` 保持 0），刷新右侧数据，并刷新 stats（因为 stats 范围变动）。
- `down/j`：
  - 若 `historyIndex < 6`：`historyIndex++`，刷新“选中日期”右侧数据。
  - 若 `historyIndex == 6`：
    - 若 `historyTo < currentDay()`：窗口整体前滚 1 天：`historyFrom++`、`historyTo++`（`historyIndex` 保持 6），刷新右侧数据，并刷新 stats。
    - 若 `historyTo == currentDay()`：不做任何事（保证不超过今天）。

---

## History 渲染设计

### 左列（日期窗口）

- 共 7 行，每行格式：
  - 选中：`> MM-DD`
  - 未选中：`  MM-DD`
- 日期不显示年份（不使用 YYYY），只显示 `MM-DD`。

### 中间分割线

- 在左右两列之间放置竖线分隔：` | `（竖线左右各一个空格）。
- 竖线高度与左右内容块高度一致（通常与 innerHeight 对齐）。

### 右列（选中日期的任务列表）

右列仅展示“选中日期”的任务，不再显示 Done/Abandoned 标题分块，避免与左列行对齐造成误读。

每行任务统一前缀为 `[]`：
- 完成：`[✓] <title>`
- 放弃：`[✗] <title>`
- 延误且仍未完成：`[ ] <title>`

延误标红规则（统一，不再讨论主动/被动）：
- 已完成任务：在其完成日期（`DoneDay`）显示；若 `DueDay < DoneDay` 则整行标红。
- 已放弃任务：在其放弃日期（`AbandonedDay`）显示；若 `DueDay < AbandonedDay` 则整行标红。
- 仍未完成任务：在其创建日期（`CreatedDay`）显示；仅当它在“今天”视角下仍然延误（`StatusActive && DueDay < currentDay()`）时才会出现在 History 里；整行标红；且方括号内不使用图标（即 `[ ]`）。

排序（默认）：完成 -> 放弃 -> 未完成延误（每组内部按 store 返回顺序）。

---

## 居中规则（工作区内）

History 内容块指：左列+分割线+右列的整体拼接结果。

- 水平居中：在 `workspaceWidth` 范围内计算内容块实际宽度，左右补空格居中。
- 垂直居中：在 `innerHeight` 范围内计算内容块实际高度，上下补空行居中。
- 当内容超过可用宽/高：遵循现有的裁剪/截断策略（ANSI-aware 截断）。

---

## 数据需求与接口

现有接口：
- `HistoryDoneByDay(day)`：返回 done_day == day 的任务。
- `HistoryAbandonedByDay(day)`：返回 abandoned_day == day 的任务。

新增需求（为“仍未完成且延误的任务在创建日显示”提供数据）：
- store/app 增加：按创建日列出 active 任务（created_day == day 且 status == active）。UI 再基于 `DueDay < currentDay()` 过滤为“仍未完成且延误”。

---

## 测试策略

- UI model 测试：
  - `Tab` 循环切换视图。
  - History up/down：普通移动刷新选中日；触顶/触底时窗口滚动 1 天且 stats 刷新；向未来不超过 today。
  - History 渲染：日期显示 MM-DD；右侧行格式包含 `[✓]/[✗]/[ ]`；延误标红（至少验证 ANSI 序列或使用 termenv Ascii 配置下的可判定输出）。
- store/sqlite 测试：新增 ListActiveByCreatedDay 查询的正确性。
