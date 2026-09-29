package framebudget

import "testing"

func TestUnderBudget(t *testing.T) {
	report := Analyze([]Stage{
		{Name: "physics", Ms: 5, Priority: 1},
		{Name: "ai", Ms: 6, Priority: 2},
	}, BudgetMs)
	if report.TotalMs != 11 || report.OverBudget {
		t.Fatalf("11ms 不该超 30ms 预算：%+v", report)
	}
	if len(report.Skipped) != 0 {
		t.Fatalf("没超预算不该跳过阶段：%v", report.Skipped)
	}
}

func TestKeptMatchesTotalWhenNothingSkipped(t *testing.T) {
	report := Analyze([]Stage{
		{Name: "net", Ms: 4, Priority: 3},
		{Name: "replication", Ms: 9, Priority: 1},
	}, BudgetMs)
	if report.KeptMs != report.TotalMs {
		t.Fatalf("没跳过阶段时保留耗时应该等于总耗时：%+v", report)
	}
}

func TestSortByPriority(t *testing.T) {
	sorted := SortByPriority([]Stage{
		{Name: "b", Priority: 2},
		{Name: "a", Priority: 2},
		{Name: "c", Priority: 1},
	})
	if sorted[0].Name != "c" || sorted[1].Name != "a" || sorted[2].Name != "b" {
		t.Fatalf("降级顺序不对：%+v", sorted)
	}
}

func TestBudgetConstant(t *testing.T) {
	if BudgetMs != 30 {
		t.Fatal("帧预算被改了")
	}
}
