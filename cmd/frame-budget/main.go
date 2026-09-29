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

// Run 执行一次命令行调用，返回退出码。
func Run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("frame-budget", flag.ContinueOnError)
	flags.SetOutput(stderr)
	sample := flags.String("sample", "over", "over / degrade")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	budgetMs := framebudget.BudgetMs
	switch *sample {
	case "over":
		budgetMs = 30
	case "degrade":
		budgetMs = 12
	default:
		fmt.Fprintln(stderr, "需要 --sample over|degrade")
		return 2
	}
	report := framebudget.Analyze(frame(), budgetMs)
	skipped := "none"
	if len(report.Skipped) > 0 {
		skipped = strings.Join(report.Skipped, ",")
	}
	fmt.Fprintf(stdout, "total=%.1f over=%v skipped=%s kept=%.1f\n",
		report.TotalMs, report.OverBudget, skipped, report.KeptMs)
	return 0
}

func main() {
	os.Exit(Run(os.Args[1:], os.Stdout, os.Stderr))
}
