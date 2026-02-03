package report

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/klyr/klyr/internal/logging"
)

type Summary struct {
	Total        int            `json:"total"`
	Allowed      int            `json:"allowed"`
	Blocked      int            `json:"blocked"`
	Shadowed     int            `json:"shadowed"`
	RateLimited  int            `json:"rate_limited"`
	Start        time.Time      `json:"start"`
	End          time.Time      `json:"end"`
	TopRules     []CountItem    `json:"top_rules"`
	TopContracts []CountItem    `json:"top_contracts"`
	TopRateLimit []CountItem    `json:"top_rate_limits"`
	Latency      LatencySummary `json:"latency"`
}

type CountItem struct {
	Key   string `json:"key"`
	Count int    `json:"count"`
}

type LatencySummary struct {
	P50 float64 `json:"p50"`
	P95 float64 `json:"p95"`
	P99 float64 `json:"p99"`
}

type Reader struct {
	Since time.Time
}

func (r *Reader) Read(path string) ([]logging.Decision, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var decisions []logging.Decision
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var d logging.Decision
		if err := json.Unmarshal([]byte(line), &d); err != nil {
			return nil, err
		}
		if !r.Since.IsZero() && d.Timestamp.Before(r.Since) {
			continue
		}
		decisions = append(decisions, d)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return decisions, nil
}

func Summarize(decisions []logging.Decision) Summary {
	var summary Summary
	if len(decisions) == 0 {
		return summary
	}

	summary.Start = decisions[0].Timestamp
	summary.End = decisions[0].Timestamp

	ruleCounts := map[string]int{}
	contractCounts := map[string]int{}
	ratelimitCounts := map[string]int{}
	latencies := make([]int64, 0, len(decisions))

	for _, d := range decisions {
		summary.Total++
		if d.Timestamp.Before(summary.Start) {
			summary.Start = d.Timestamp
		}
		if d.Timestamp.After(summary.End) {
			summary.End = d.Timestamp
		}

		switch d.Action {
		case "allow":
			summary.Allowed++
		case "block":
			summary.Blocked++
		case "shadow":
			summary.Shadowed++
		}

		if d.RateLimited {
			summary.RateLimited++
		}

		for _, match := range d.MatchedRules {
			ruleCounts[match.ID]++
		}
		for _, v := range d.ContractViolations {
			contractCounts[v.Type]++
		}
		if d.RateLimited {
			ratelimitCounts[d.ClientIP]++
		}

		latencies = append(latencies, d.DurationMS)
	}

	summary.TopRules = topCounts(ruleCounts, 5)
	summary.TopContracts = topCounts(contractCounts, 5)
	summary.TopRateLimit = topCounts(ratelimitCounts, 5)
	summary.Latency = latencySummary(latencies)

	return summary
}

func topCounts(counts map[string]int, n int) []CountItem {
	items := make([]CountItem, 0, len(counts))
	for key, count := range counts {
		items = append(items, CountItem{Key: key, Count: count})
	}
	if len(items) == 0 {
		return nil
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].Count == items[j].Count {
			return items[i].Key < items[j].Key
		}
		return items[i].Count > items[j].Count
	})

	if len(items) > n {
		items = items[:n]
	}
	return items
}

func latencySummary(values []int64) LatencySummary {
	if len(values) == 0 {
		return LatencySummary{}
	}
	sorted := make([]int64, len(values))
	copy(sorted, values)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })

	return LatencySummary{
		P50: percentile(sorted, 0.50),
		P95: percentile(sorted, 0.95),
		P99: percentile(sorted, 0.99),
	}
}

func percentile(values []int64, p float64) float64 {
	if len(values) == 0 {
		return 0
	}
	idx := int(float64(len(values)-1) * p)
	if idx < 0 {
		idx = 0
	}
	if idx >= len(values) {
		idx = len(values) - 1
	}
	return float64(values[idx])
}

func RenderText(summary Summary) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Total: %d\n", summary.Total)
	fmt.Fprintf(&b, "Allowed: %d\n", summary.Allowed)
	fmt.Fprintf(&b, "Blocked: %d\n", summary.Blocked)
	fmt.Fprintf(&b, "Shadowed: %d\n", summary.Shadowed)
	fmt.Fprintf(&b, "Rate limited: %d\n", summary.RateLimited)
	fmt.Fprintf(&b, "Latency p50/p95/p99 (ms): %.0f/%.0f/%.0f\n", summary.Latency.P50, summary.Latency.P95, summary.Latency.P99)

	writeCounts(&b, "Top blocked rules", summary.TopRules)
	writeCounts(&b, "Top contract violations", summary.TopContracts)
	writeCounts(&b, "Top rate-limited", summary.TopRateLimit)

	return b.String()
}

func RenderMarkdown(summary Summary) string {
	var b strings.Builder
	b.WriteString("# Klyr Report\n\n")
	b.WriteString("## Totals\n\n")
	fmt.Fprintf(&b, "- Total: %d\n", summary.Total)
	fmt.Fprintf(&b, "- Allowed: %d\n", summary.Allowed)
	fmt.Fprintf(&b, "- Blocked: %d\n", summary.Blocked)
	fmt.Fprintf(&b, "- Shadowed: %d\n", summary.Shadowed)
	fmt.Fprintf(&b, "- Rate limited: %d\n", summary.RateLimited)
	fmt.Fprintf(&b, "- Latency p50/p95/p99 (ms): %.0f/%.0f/%.0f\n\n", summary.Latency.P50, summary.Latency.P95, summary.Latency.P99)

	writeCountsMarkdown(&b, "Top blocked rules", summary.TopRules)
	writeCountsMarkdown(&b, "Top contract violations", summary.TopContracts)
	writeCountsMarkdown(&b, "Top rate-limited", summary.TopRateLimit)

	return b.String()
}

func RenderJSON(summary Summary) ([]byte, error) {
	return json.MarshalIndent(summary, "", "  ")
}

func writeCounts(b *strings.Builder, title string, items []CountItem) {
	if len(items) == 0 {
		fmt.Fprintf(b, "%s: none\n", title)
		return
	}
	fmt.Fprintf(b, "%s:\n", title)
	for _, item := range items {
		fmt.Fprintf(b, "- %s: %d\n", item.Key, item.Count)
	}
}

func writeCountsMarkdown(b *strings.Builder, title string, items []CountItem) {
	b.WriteString("## ")
	b.WriteString(title)
	b.WriteString("\n\n")
	if len(items) == 0 {
		b.WriteString("- none\n\n")
		return
	}
	for _, item := range items {
		fmt.Fprintf(b, "- %s: %d\n", item.Key, item.Count)
	}
	b.WriteString("\n")
}

func WriteOutput(path string, content []byte) error {
	if path == "" {
		_, err := io.Copy(os.Stdout, bytes.NewReader(content))
		return err
	}
	return os.WriteFile(path, content, 0o600)
}
