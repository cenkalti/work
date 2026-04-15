package harness

import (
	"encoding/json"
	"fmt"
	"io"
	"text/tabwriter"
)

func PrintScores(w io.Writer, scores []Score, format string) error {
	if format == "json" {
		return json.NewEncoder(w).Encode(scores)
	}
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "PRINCIPLE\tSCORE\tREASONING")
	fmt.Fprintln(tw, "---------\t-----\t---------")
	for _, s := range scores {
		fmt.Fprintf(tw, "%s\t%d/10\t%s\n", s.Principle, s.Value, s.Reasoning)
	}
	return tw.Flush()
}

type BenchResult struct {
	PrincipleID      string
	PrincipleName    string
	AvgBaseline      float64
	AvgWithGuidelines float64
	Delta            float64
	Wins             int
	Ties             int
	Losses           int
}

func PrintBenchResults(w io.Writer, results []BenchResult, format string) error {
	if format == "json" {
		return json.NewEncoder(w).Encode(results)
	}
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "PRINCIPLE\tBASELINE\tWITH GUIDELINES\tDELTA\tW/T/L")
	fmt.Fprintln(tw, "---------\t--------\t---------------\t-----\t-----")
	for _, r := range results {
		delta := fmt.Sprintf("%+.1f", r.Delta)
		fmt.Fprintf(tw, "%s\t%.1f\t%.1f\t%s\t%d/%d/%d\n",
			r.PrincipleName, r.AvgBaseline, r.AvgWithGuidelines, delta,
			r.Wins, r.Ties, r.Losses)
	}
	return tw.Flush()
}

type CompareResult struct {
	Principle        string
	BaselineScore    int
	GuidelinesScore  int
	Delta            int
	BaselineReason   string
	GuidelinesReason string
}

func PrintCompare(w io.Writer, results []CompareResult, format string) error {
	if format == "json" {
		return json.NewEncoder(w).Encode(results)
	}
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "PRINCIPLE\tBASELINE\tWITH GUIDELINES\tDELTA")
	fmt.Fprintln(tw, "---------\t--------\t---------------\t-----")
	for _, r := range results {
		delta := fmt.Sprintf("%+d", r.Delta)
		fmt.Fprintf(tw, "%s\t%d/10\t%d/10\t%s\n",
			r.Principle, r.BaselineScore, r.GuidelinesScore, delta)
	}
	return tw.Flush()
}
