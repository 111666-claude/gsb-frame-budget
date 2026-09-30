package framebudget

import "testing"

func TestSingleFrameUnderBudget(t *testing.T) {
	ledger := NewLedger([]Stage{
		{Name: "physics", Ms: 5, Priority: 1},
		{Name: "ai", Ms: 6, Priority: 2},
	})
	report := ledger.Frame(0)
	if report.FrameMs != 11 || report.OverBudget {
		t.Fatalf("11ms 的单帧不该超预算：%+v", report)
	}
	if len(report.Skipped) != 0 {
		t.Fatalf("没超预算不该跳过阶段：%v", report.Skipped)
	}
}

func TestWindowCountsFrame(t *testing.T) {
	ledger := NewLedger([]Stage{{Name: "net", Ms: 4, Priority: 1}})
	report := ledger.Frame(0)
	if report.WindowMs < report.FrameMs {
		t.Fatalf("这一帧的耗时应该计入窗口：%+v", report)
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

func TestConstants(t *testing.T) {
	if BudgetMs != 30 || SecondBudgetMs != 900 || WindowMs != 1000 {
		t.Fatal("帧预算常量被改了")
	}
}
