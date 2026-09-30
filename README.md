# frame-budget

服务器帧预算：秒级累计预算、按优先级降级、账本随时间滑出，只用 Go 标准库。

```
go test ./...
go run ./cmd/frame-budget --sample single
go run ./cmd/frame-budget --sample sustain
```

## 口径（README 为准）

- **耗时按浮点值累加**：`FrameMs` 是各阶段耗时的浮点和，不许截成整数毫秒。
- **秒级账本**：`WindowMs` 是最近 `WindowMs = 1000` 毫秒内计入的耗时之和（含本帧），
  窗口必须随时间滑出，老帧不能一直累加。
- **超预算就降级**：当"窗口里已计入的耗时 + 本帧全量耗时"超过 `SecondBudgetMs = 900` 时，
  按优先级从低到高（优先级数字大的先跳，同优先级按名字字典序升序）依次跳过阶段，
  直到窗口 + 本帧保留的耗时不超预算，或者整帧全部跳完；`Skipped` 按跳过顺序列出。
- **不变量**：`WindowMs` 不超过 `SecondBudgetMs`（整帧全跳仍超的极端除外）；
  `Skipped` 里优先级数字不降；没超预算时 `Skipped` 必须为空；同一帧序列重复执行结果相同。
- **规模**：单帧 1 万个阶段、每秒 100 帧跑 10 秒，每帧 O(n log n)，
  内存 O(阶段数 + 窗口桶数)；不许每帧重算整段历史账本。

## 输出契约（不改格式）

```
frame=30.3 window=30.3 skipped=none
frames=120 skipped_frames>=1 peak_window<=900.0
```

## 常量

```
BudgetMs       = 30.0
SecondBudgetMs = 900.0
WindowMs       = 1000
```

## 目录

```
budget.go               帧预算分析
cmd/frame-budget        命令行入口
budget_test.go          go test 用例
```
