// Package framebudget 是服务器帧预算：秒级累计预算、按优先级降级、账本随时间滑出。
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

// frameCost 是窗口里一帧实际计入的耗时。
type frameCost struct {
	at int64
	ms float64
}

// Ledger 是跨帧的预算账本。
type Ledger struct {
	stages  []Stage
	degrade []Stage // 预先排好的降级顺序：优先级数字大的先跳，同优先级按名字升序
	window  []frameCost
}

// NewLedger 建一个账本。
func NewLedger(stages []Stage) *Ledger {
	degrade := append([]Stage(nil), stages...)
	sort.SliceStable(degrade, func(i, j int) bool {
		if degrade[i].Priority != degrade[j].Priority {
			return degrade[i].Priority > degrade[j].Priority
		}
		return degrade[i].Name < degrade[j].Name
	})
	return &Ledger{
		stages:  append([]Stage(nil), stages...),
		degrade: degrade,
	}
}

// Frame 推进一帧并给出降级结果。
// 耗时按浮点累加；窗口只保留最近 WindowMs 内计入的耗时；
// 窗口加本帧全量超过 SecondBudgetMs 时按降级顺序跳阶段，直到不超预算或整帧跳完。
func (l *Ledger) Frame(nowMs int64) Report {
	kept := 0
	for kept < len(l.window) && nowMs-l.window[kept].at >= WindowMs {
		kept++
	}
	l.window = l.window[kept:]
	if len(l.window) == 0 {
		l.window = nil
	}

	windowMs := 0.0
	for _, frame := range l.window {
		windowMs += frame.ms
	}

	total := 0.0
	for _, stage := range l.stages {
		total += stage.Ms
	}

	remaining := total
	var skipped []string
	if windowMs+remaining > SecondBudgetMs {
		for _, stage := range l.degrade {
			if windowMs+remaining <= SecondBudgetMs {
				break
			}
			skipped = append(skipped, stage.Name)
			remaining -= stage.Ms
		}
	}

	l.window = append(l.window, frameCost{at: nowMs, ms: remaining})
	windowMs += remaining

	return Report{
		FrameMs:    total,
		WindowMs:   windowMs,
		OverBudget: total > BudgetMs,
		Skipped:    skipped,
	}
}

// SortByPriority 按重要性排阶段：优先级从小到大，同一优先级按名字升序。
func SortByPriority(stages []Stage) []Stage {
	sorted := append([]Stage(nil), stages...)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].Priority !=
			sorted[j].Priority {
			return sorted[i].Priority < sorted[j].Priority
		}
		return sorted[i].Name < sorted[j].Name
	})
	return sorted
}
