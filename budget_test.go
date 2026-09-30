package framebudget

import (
	"fmt"
	"math/rand"
	"sort"
	"testing"
)

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

func TestFrameMsFloatSum(t *testing.T) {
	ledger := NewLedger([]Stage{
		{Name: "physics", Ms: 9.9, Priority: 1},
		{Name: "replication", Ms: 10.9, Priority: 2},
		{Name: "weather", Ms: 9.5, Priority: 3},
	})
	report := ledger.Frame(0)
	want := 9.9 + 10.9 + 9.5
	if report.FrameMs != want {
		t.Fatalf("耗时必须按浮点累加：got %v want %v", report.FrameMs, want)
	}
	if report.WindowMs != want {
		t.Fatalf("首帧窗口应等于本帧保留耗时：got %v want %v", report.WindowMs, want)
	}
	if len(report.Skipped) != 0 {
		t.Fatalf("30.3ms 没超秒级预算不该降级：%v", report.Skipped)
	}
}

func TestWindowSlides(t *testing.T) {
	ledger := NewLedger([]Stage{{Name: "net", Ms: 100, Priority: 1}})
	for _, now := range []int64{0, 400, 800} {
		ledger.Frame(now)
	}
	report := ledger.Frame(1200)
	if report.WindowMs != 300 {
		t.Fatalf("t=0 的老帧应滑出窗口：got %v want 300", report.WindowMs)
	}
	for now := int64(1600); now <= 100000; now += 400 {
		report = ledger.Frame(now)
	}
	if report.WindowMs != 300 {
		t.Fatalf("账本不能无限累加，稳态窗口应保持 300：got %v", report.WindowMs)
	}
}

func TestDegradeOrder(t *testing.T) {
	stages := []Stage{
		{Name: "base", Ms: 880, Priority: 0},
		{Name: "c", Ms: 20, Priority: 1},
		{Name: "b", Ms: 10, Priority: 2},
		{Name: "a", Ms: 10, Priority: 2},
		{Name: "d", Ms: 10, Priority: 3},
	}
	ledger := NewLedger(stages)
	report := ledger.Frame(0)
	if report.FrameMs != 930 {
		t.Fatalf("FrameMs 是全量浮点和：got %v want 930", report.FrameMs)
	}
	want := []string{"d", "a", "b"}
	if !equalStrings(report.Skipped, want) {
		t.Fatalf("降级顺序应为优先级数字大的先跳、同级按名字升序：got %v want %v", report.Skipped, want)
	}
	if report.WindowMs != 900 {
		t.Fatalf("跳完 d,a,b 后窗口应正好压到预算：got %v want 900", report.WindowMs)
	}
}

func TestDegradeSkipsWholeFrame(t *testing.T) {
	ledger := NewLedger([]Stage{
		{Name: "x", Ms: 950, Priority: 1},
		{Name: "y", Ms: 50, Priority: 2},
	})
	report := ledger.Frame(0)
	if !equalStrings(report.Skipped, []string{"y", "x"}) {
		t.Fatalf("整帧全跳时 Skipped 要按顺序列全：%v", report.Skipped)
	}
	if report.WindowMs != 0 {
		t.Fatalf("整帧全跳后窗口不该计入本帧：%v", report.WindowMs)
	}
}

func TestDeterministic(t *testing.T) {
	stages := []Stage{
		{Name: "physics", Ms: 9.9, Priority: 1},
		{Name: "replication", Ms: 10.9, Priority: 2},
		{Name: "weather", Ms: 9.5, Priority: 3},
	}
	run := func() []Report {
		ledger := NewLedger(stages)
		reports := make([]Report, 120)
		for index := range reports {
			reports[index] = ledger.Frame(int64(index) * 33)
		}
		return reports
	}
	first, second := run(), run()
	for index := range first {
		if first[index].FrameMs != second[index].FrameMs ||
			first[index].WindowMs != second[index].WindowMs ||
			!equalStrings(first[index].Skipped, second[index].Skipped) {
			t.Fatalf("第 %d 帧重复执行结果不同：%+v vs %+v", index, first[index], second[index])
		}
	}
}

// refEntry 是参考实现账本里一帧计入的耗时。
type refEntry struct {
	timeMs   int64
	retained float64
}

// refFrame 是「秒级窗口 + 降级」的参考实现：每帧从头过滤整段历史，
// 与 Ledger 的增量实现互为对照。
func refFrame(stages []Stage, history []refEntry, nowMs int64) (Report, refEntry) {
	window := 0.0
	for _, entry := range history {
		if entry.timeMs > nowMs-WindowMs {
			window += entry.retained
		}
	}
	total := 0.0
	for _, stage := range stages {
		total += stage.Ms
	}
	degraded := append([]Stage(nil), stages...)
	sort.SliceStable(degraded, func(i, j int) bool {
		if degraded[i].Priority != degraded[j].Priority {
			return degraded[i].Priority > degraded[j].Priority
		}
		return degraded[i].Name < degraded[j].Name
	})
	retained := total
	var skipped []string
	for _, stage := range degraded {
		if window+retained <= SecondBudgetMs {
			break
		}
		retained -= stage.Ms
		skipped = append(skipped, stage.Name)
	}
	report := Report{
		FrameMs:    total,
		WindowMs:   window + retained,
		OverBudget: total > BudgetMs,
		Skipped:    skipped,
	}
	return report, refEntry{timeMs: nowMs, retained: retained}
}

func TestAgainstReference(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	for trial := 0; trial < 200; trial++ {
		stageCount := 1 + rng.Intn(8)
		stages := make([]Stage, stageCount)
		priority := make(map[string]int, stageCount)
		for index := range stages {
			stages[index] = Stage{
				Name:     fmt.Sprintf("stage-%d", index),
				Ms:       float64(rng.Intn(30000)) / 100,
				Priority: rng.Intn(4),
			}
			priority[stages[index].Name] = stages[index].Priority
		}
		frameCount := 1 + rng.Intn(150)
		ledger := NewLedger(stages)
		var history []refEntry
		now := int64(0)
		for frame := 0; frame < frameCount; frame++ {
			now += int64(rng.Intn(50))
			got := ledger.Frame(now)
			want, entry := refFrame(stages, history, now)
			history = append(history, entry)
			if got.FrameMs != want.FrameMs || got.WindowMs != want.WindowMs ||
				!equalStrings(got.Skipped, want.Skipped) {
				t.Fatalf("trial %d 第 %d 帧与参考实现不一致：got %+v want %+v",
					trial, frame, got, want)
			}
			if len(got.Skipped) == 0 && got.WindowMs > SecondBudgetMs {
				t.Fatalf("trial %d 第 %d 帧没降级却超预算：%+v", trial, frame, got)
			}
			if len(got.Skipped) < len(stages) && got.WindowMs > SecondBudgetMs {
				t.Fatalf("trial %d 第 %d 帧没跳完就超预算：%+v", trial, frame, got)
			}
			for index := 1; index < len(got.Skipped); index++ {
				if priority[got.Skipped[index-1]] < priority[got.Skipped[index]] {
					t.Fatalf("trial %d 第 %d 帧 Skipped 优先级不是不降：%v",
						trial, frame, got.Skipped)
				}
			}
		}
	}
}

func TestScale(t *testing.T) {
	stages := make([]Stage, 10000)
	for index := range stages {
		stages[index] = Stage{
			Name:     fmt.Sprintf("stage-%05d", index),
			Ms:       0.05,
			Priority: index % 5,
		}
	}
	ledger := NewLedger(stages)
	for frame := int64(0); frame < 1000; frame++ {
		report := ledger.Frame(frame * 10)
		if len(report.Skipped) < len(stages) && report.WindowMs > SecondBudgetMs {
			t.Fatalf("第 %d 帧没跳完就超预算：%+v", frame, report)
		}
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for index := range a {
		if a[index] != b[index] {
			return false
		}
	}
	return true
}
