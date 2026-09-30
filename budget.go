// Package framebudget 是服务器帧预算：秒级累计预算、按优先级降级、账本随时间滑出。
// 缺陷：耗时截断成整数、只看单帧预算不看秒级累计、没有降级逻辑、账本窗口不滑动。
package framebudget

import "sort"

// BudgetMs 是单帧预算参考值。
const BudgetMs = 30.0

// SecondBudgetMs 是每秒钟允许花费的总耗时。
const SecondBudgetMs = 900.0

// WindowMs 是秒级账本的窗口长度。
const WindowMs = 1000

// Stage 是帧里的一个阶段。
type Stage struct {
	Name     string
	Ms       float64
	Priority int // 越小越重要，降级时先跳数字大的
}

// Report 是一帧的分析结果。WindowMs 是最近一秒内计入的累计耗时（含本帧）。
type Report struct {
	FrameMs    float64
	WindowMs   float64
	OverBudget bool
	Skipped    []string
}

// Ledger 是跨帧的预算账本。
type Ledger struct {
	stages []Stage
	spent  float64
}

// NewLedger 建一个账本。
func NewLedger(stages []Stage) *Ledger {
	return &Ledger{stages: append([]Stage(nil), stages...)}
}

// Frame 推进一帧并给出降级结果。
// 缺陷：每阶段耗时先截成整数毫秒；只和单帧预算比较；从不跳过任何阶段；
// 账本把每帧耗时一路累加，从不随时间滑出。
func (l *Ledger) Frame(nowMs int64) Report {
	total := 0.0
	for _, stage := range l.stages {
		total += float64(int(stage.Ms))
	}
	l.spent += total
	return Report{
		FrameMs:    total,
		WindowMs:   l.spent,
		OverBudget: total > BudgetMs,
	}
}

// SortByPriority 按降级顺序排阶段：优先级从小到大，同一优先级按名字升序。
func SortByPriority(stages []Stage) []Stage {
	sorted := append([]Stage(nil), stages...)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].Priority != sorted[j].Priority {
			return sorted[i].Priority < sorted[j].Priority
		}
		return sorted[i].Name < sorted[j].Name
	})
	return sorted
}
