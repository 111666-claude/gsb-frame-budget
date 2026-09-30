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

func TestFloatAccumulation(t *testing.T) {
	ledger := NewLedger([]Stage{
		{Name: "physics", Ms: 9.9, Priority: 1},
		{Name: "replication", Ms: 10.9, Priority: 2},
		{Name: "weather", Ms: 9.5, Priority: 3},
	})
	report := ledger.Frame(0)
	want := 9.9 + 10.9 + 9.5
	if report.FrameMs != want {
		t.Fatalf("耗时必须按浮点累加，不许截断：got %v want %v", report.FrameMs, want)
	}
	if report.FrameMs <= 30 {
		t.Fatalf("截断成整数会得到 28，浮点和应大于 30：%v", report.FrameMs)
	}
	if report.WindowMs != want {
		t.Fatalf("首帧窗口应等于本帧耗时：got %v want %v", report.WindowMs, want)
	}
}

func TestWindowSlides(t *testing.T) {
	ledger := NewLedger([]Stage{{Name: "net", Ms: 400, Priority: 1}})
	for _, now := range []int64{0, 500, 1000, 1500, 2000} {
		report := ledger.Frame(now)
		if report.WindowMs > 800 {
			t.Fatalf("t=%d 老帧必须滑出窗口，不该一直累加：%+v", now, report)
		}
		if len(report.Skipped) != 0 {
			t.Fatalf("t=%d 窗口 800 不超预算，不该降级：%+v", now, report)
		}
	}
	report := ledger.Frame(100001)
	if report.WindowMs != 400 {
		t.Fatalf("窗口完全滑空后只剩本帧：%+v", report)
	}
}

func TestDegradeOrder(t *testing.T) {
	ledger := NewLedger([]Stage{
		{Name: "physics", Ms: 500, Priority: 1},
		{Name: "b", Ms: 300, Priority: 3},
		{Name: "a", Ms: 300, Priority: 3},
		{Name: "replication", Ms: 300, Priority: 2},
	})
	report := ledger.Frame(0)
	// 全量 1400，依次跳 a、b（同优先级按名字升序）后剩 800，不超预算即停。
	want := []string{"a", "b"}
	if len(report.Skipped) != len(want) {
		t.Fatalf("降级顺序不对：%v", report.Skipped)
	}
	for i := range want {
		if report.Skipped[i] != want[i] {
			t.Fatalf("降级顺序不对：got %v want %v", report.Skipped, want)
		}
	}
	if report.WindowMs != 800 {
		t.Fatalf("降级后窗口应为保留耗时 800：%+v", report)
	}
	if report.FrameMs != 1400 {
		t.Fatalf("FrameMs 仍是全量浮点和：%+v", report)
	}
}

func TestDegradeWholeFrame(t *testing.T) {
	ledger := NewLedger([]Stage{
		{Name: "physics", Ms: 500, Priority: 1},
		{Name: "weather", Ms: 600, Priority: 2},
	})
	report := ledger.Frame(0)
	// 全量 1100，跳完 weather 仍 500+... 实际跳 weather 后 500<=900 停。
	if len(report.Skipped) != 1 || report.Skipped[0] != "weather" {
		t.Fatalf("只需跳一个阶段：%v", report.Skipped)
	}
	// 构造整帧全跳：窗口先占满 900，再来一帧只能全跳。
	ledger2 := NewLedger([]Stage{
		{Name: "x", Ms: 50, Priority: 1},
		{Name: "y", Ms: 100, Priority: 2},
	})
	for i := int64(0); i <= 500; i += 100 {
		ledger2.Frame(i) // 每帧 150，窗口累到 900
	}
	report = ledger2.Frame(600) // 900+150>900，跳 y 后 950 仍超，整帧全跳
	if len(report.Skipped) != 2 || report.Skipped[0] != "y" || report.Skipped[1] != "x" {
		t.Fatalf("整帧全跳顺序不对：%v", report.Skipped)
	}
	if report.WindowMs != 900 {
		t.Fatalf("整帧全跳后窗口不计入本帧：%+v", report)
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
		var reports []Report
		for i := int64(0); i < 120; i++ {
			reports = append(reports, ledger.Frame(i*33))
		}
		return reports
	}
	first, second := run(), run()
	for i := range first {
		if first[i].FrameMs != second[i].FrameMs ||
			first[i].WindowMs != second[i].WindowMs ||
			!sameStrings(first[i].Skipped, second[i].Skipped) {
			t.Fatalf("第 %d 帧重复执行结果不同：%+v vs %+v", i, first[i], second[i])
		}
	}
}

func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// referenceFrame 是「秒级窗口 + 降级」的参考实现：每帧从整段历史重算，刻意朴素。
func referenceFrame(stages []Stage, history []frameCost, now int64) (Report, []frameCost) {
	total := 0.0
	for _, stage := range stages {
		total += stage.Ms
	}
	windowMs := 0.0
	next := history[:0]
	for _, frame := range history {
		if now-frame.at < WindowMs {
			windowMs += frame.ms
			next = append(next, frame)
		}
	}
	remaining := total
	var skipped []string
	if windowMs+remaining > SecondBudgetMs {
		order := append([]Stage(nil), stages...)
		sort.SliceStable(order, func(i, j int) bool {
			if order[i].Priority != order[j].Priority {
				return order[i].Priority > order[j].Priority
			}
			return order[i].Name < order[j].Name
		})
		for _, stage := range order {
			if windowMs+remaining <= SecondBudgetMs {
				break
			}
			skipped = append(skipped, stage.Name)
			remaining -= stage.Ms
		}
	}
	next = append(next, frameCost{at: now, ms: remaining})
	windowMs += remaining
	return Report{
		FrameMs:    total,
		WindowMs:   windowMs,
		OverBudget: total > BudgetMs,
		Skipped:    skipped,
	}, next
}

func TestAgainstReference(t *testing.T) {
	rng := rand.New(rand.NewSource(20260930))
	names := []string{"physics", "ai", "net", "weather", "audio", "render", "gc", "io"}
	for trial := 0; trial < 200; trial++ {
		stageCount := 1 + rng.Intn(8)
		stages := make([]Stage, stageCount)
		for i := range stages {
			stages[i] = Stage{
				Name:     fmt.Sprintf("%s%d-%d", names[rng.Intn(len(names))], rng.Intn(3), i),
				Ms:       float64(rng.Intn(5000)) / 100,
				Priority: rng.Intn(5),
			}
		}
		frameCount := 1 + rng.Intn(150)
		times := make([]int64, frameCount)
		for i := range times {
			if i == 0 {
				times[i] = int64(rng.Intn(50))
			} else {
				times[i] = times[i-1] + 1 + int64(rng.Intn(1200))
			}
		}
		ledger := NewLedger(stages)
		var history []frameCost
		for _, now := range times {
			got := ledger.Frame(now)
			want, next := referenceFrame(stages, history, now)
			history = next
			if got.FrameMs != want.FrameMs ||
				got.WindowMs != want.WindowMs ||
				got.OverBudget != want.OverBudget ||
				!sameStrings(got.Skipped, want.Skipped) {
				t.Fatalf("trial=%d t=%d 与参考实现不一致：\n got %+v\nwant %+v\nstages=%+v",
					trial, now, got, want, stages)
			}
			if len(got.Skipped) == 0 && got.WindowMs > SecondBudgetMs {
				t.Fatalf("trial=%d t=%d 窗口超预算却没降级：%+v", trial, now, got)
			}
			priorityOf := map[string]int{}
			for _, stage := range stages {
				priorityOf[stage.Name] = stage.Priority
			}
			lastPriority := 1 << 30
			for _, name := range got.Skipped {
				if priorityOf[name] > lastPriority {
					t.Fatalf("trial=%d t=%d Skipped 里优先级必须不升（数字大的先跳）：%v", trial, now, got.Skipped)
				}
				lastPriority = priorityOf[name]
			}
		}
	}
}

func TestScale(t *testing.T) {
	stages := make([]Stage, 10000)
	for i := range stages {
		stages[i] = Stage{
			Name:     fmt.Sprintf("stage%05d", i),
			Ms:       0.003,
			Priority: i % 7,
		}
	}
	ledger := NewLedger(stages)
	for i := int64(0); i < 1000; i++ {
		report := ledger.Frame(i * 10)
		if report.WindowMs > SecondBudgetMs && len(report.Skipped) < len(stages) {
			t.Fatalf("帧 %d 窗口超预算且未整帧全跳：%+v", i, report)
		}
	}
}
