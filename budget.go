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

// frameEntry 是窗口里一帧实际计入（保留）的耗时。
type frameEntry struct {
	timeMs int64
	ms     float64
}

// Ledger 是跨帧的预算账本。entries 只保留仍在窗口内的帧，
// 老帧随时间滑出，内存是 O(阶段数 + 窗口桶数)。
type Ledger struct {
	stages   []Stage
	degraded []Stage // 按跳过顺序排好：优先级数字大的在前，同优先级按名字升序
	entries  []frameEntry
	head     int
}

// NewLedger 建一个账本。
func NewLedger(stages []Stage) *Ledger {
	degraded := append([]Stage(nil), stages...)
	sort.SliceStable(degraded, func(i, j int) bool {
		if degraded[i].Priority != degraded[j].Priority {
			return degraded[i].Priority > degraded[j].Priority
		}
		return degraded[i].Name < degraded[j].Name
	})
	return &Ledger{
		stages:   append([]Stage(nil), stages...),
		degraded: degraded,
	}
}

// Frame 推进一帧并给出降级结果。nowMs 需随帧号单调不减。
// 耗时按浮点累加；窗口只计入最近 WindowMs 毫秒内保留的耗时；
// 窗口加本帧全量超过 SecondBudgetMs 时按降级顺序跳阶段，
// 直到不超预算或整帧跳完。每帧 O(n log n)，不重算整段历史账本。
func (l *Ledger) Frame(nowMs int64) Report {
	l.evict(nowMs)
	window := l.windowSum()

	total := 0.0
	for _, stage := range l.stages {
		total += stage.Ms
	}

	retained := total
	var skipped []string
	for _, stage := range l.degraded {
		if window+retained <= SecondBudgetMs {
			break
		}
		retained -= stage.Ms
		skipped = append(skipped, stage.Name)
	}

	l.entries = append(l.entries, frameEntry{timeMs: nowMs, ms: retained})
	window += retained

	return Report{
		FrameMs:    total,
		WindowMs:   window,
		OverBudget: total > BudgetMs,
		Skipped:    skipped,
	}
}

// evict 把滑出窗口的老帧从账本里移除，并摊还压缩底层数组。
func (l *Ledger) evict(nowMs int64) {
	cutoff := nowMs - WindowMs
	for l.head < len(l.entries) && l.entries[l.head].timeMs <= cutoff {
		l.head++
	}
	if l.head > 0 && l.head*2 >= len(l.entries) {
		l.entries = append([]frameEntry(nil), l.entries[l.head:]...)
		l.head = 0
	}
}

// windowSum 按时间顺序累加窗口内计入的耗时，只看不超窗口的桶。
func (l *Ledger) windowSum() float64 {
	sum := 0.0
	for _, entry := range l.entries[l.head:] {
		sum += entry.ms
	}
	return sum
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
