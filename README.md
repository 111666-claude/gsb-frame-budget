# frame-budget

服务器帧预算：统计各阶段耗时、判超预算并按优先级降级，只用 Go 标准库。

```
go test ./...
go run ./cmd/frame-budget --sample over
go run ./cmd/frame-budget --sample degrade
```

## 口径（README 为准）

样例帧是三个阶段：`physics` 9.9ms（优先级 1）、`replication` 10.9ms（优先级 2）、
`weather` 9.5ms（优先级 3），合计 30.3ms。

- **耗时按实际值累加**：`total` 是各阶段耗时的浮点和，不许截成整数毫秒。
  `--sample over`（预算 30ms）按口径是 `total=30.3 over=true`，现在是 `total=28.0 over=false`。
- **超预算要降级**：按优先级从低到高（优先级数字大的先跳，同优先级按名字字典序升序）
  依次跳过阶段，直到剩下的耗时不超过预算为止。跳过的阶段按跳过顺序写进 `skipped`，
  `kept` 是剩下阶段的耗时之和。`--sample over` 按口径是
  `skipped=weather kept=20.8`；`--sample degrade`（预算 12ms）按口径是
  `skipped=weather,replication kept=9.9`。现在两个样例都给 `skipped=none`。
- 不变量：`kept <= total`；`skipped` 里每个阶段最多出现一次，且优先级数字不降；
  没超预算时 `skipped` 必须为空；同一帧重复分析结果相同。
- 规模：单帧 1 万个阶段在 O(n log n) 内分析完，内存 O(n)；先算总耗时再按降级顺序做减法，
  不许每试跳一个阶段都重算全表。

## 现在的行为

- `Analyze` 用 `float64(int(stage.Ms))` 累加，小数部分被丢掉，30.3ms 的帧算成 28ms。
- `OverBudget` 只做了比较，`Skipped` 从来没被写过，超预算也不降级。
- `SortByPriority` 写好了但没人调用。

## 输出

```
total=30.3 over=true skipped=weather kept=20.8
total=30.3 over=true skipped=weather,replication kept=9.9
```

## 目录

```
budget.go               帧预算分析
cmd/frame-budget        命令行入口
budget_test.go          go test 用例
```
