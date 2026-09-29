// Package framebudget 是服务器帧预算：统计各阶段耗时、判超预算并按优先级降级。
// 缺陷：耗时累加被截成整数，超预算也不降级。
package framebudget

import "sort"

// BudgetMs 是默认的每帧预算，毫秒。
const BudgetMs = 30.0

// Stage 是帧里的一个阶段。
type Stage struct {
	Name     string
	Ms       float64
	Priority int // 越小越重要，降级时先跳数字大的
}

// Report 是一帧的分析结果。
type Report struct {
	TotalMs    float64
	KeptMs     float64
	OverBudget bool
	Skipped    []string
}

// Analyze 分析一帧：算出总耗时，超预算时按优先级跳过阶段。
// 缺陷：每个阶段先截成整数毫秒再累加，也从不跳过任何阶段。
func Analyze(stages []Stage, budgetMs float64) Report {
	report := Report{}
	for _, stage := range stages {
		report.TotalMs += float64(int(stage.Ms))
	}
	report.KeptMs = report.TotalMs
	report.OverBudget = report.TotalMs > budgetMs
	return report
}

// SortByPriority 按降级顺序把阶段排好：优先级从小到大，同一优先级按名字升序。
// 缺陷：没人调用它。
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
