// Command frame-budget 跑帧预算样例。
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"example.com/framebudget"
)

func frame() []framebudget.Stage {
	return []framebudget.Stage{
		{Name: "physics", Ms: 9.9, Priority: 1},
		{Name: "replication", Ms: 10.9, Priority: 2},
		{Name: "weather", Ms: 9.5, Priority: 3},
	}
}

func skippedText(skipped []string) string {
	if len(skipped) == 0 {
		return "none"
	}
	return strings.Join(skipped, ",")
}

// Run 执行一次命令行调用，返回退出码。
func Run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("frame-budget", flag.ContinueOnError)
	flags.SetOutput(stderr)
	sample := flags.String("sample", "single", "single / sustain")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	switch *sample {
	case "single":
		ledger := framebudget.NewLedger(frame())
		report := ledger.Frame(0)
		fmt.Fprintf(stdout, "frame=%.1f window=%.1f skipped=%s\n",
			report.FrameMs, report.WindowMs, skippedText(report.Skipped))
	case "sustain":
		// 三十帧每秒的高负载持续四秒。
		ledger := framebudget.NewLedger(frame())
		var report framebudget.Report
		skippedFrames := 0
		peakWindow := 0.0
		for index := 0; index < 120; index++ {
			report = ledger.Frame(int64(index) * 33)
			if len(report.Skipped) > 0 {
				skippedFrames++
			}
			if report.WindowMs > peakWindow {
				peakWindow = report.WindowMs
			}
		}
		fmt.Fprintf(stdout, "frames=120 skipped_frames=%d peak_window=%.1f frame=%.1f skipped=%s\n",
			skippedFrames, peakWindow, report.FrameMs, skippedText(report.Skipped))
	default:
		fmt.Fprintln(stderr, "需要 --sample single|sustain")
		return 2
	}
	return 0
}

func main() {
	os.Exit(Run(os.Args[1:], os.Stdout, os.Stderr))
}
