package explorer

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// doctorCheck is the result of checking one protocol mode.
type doctorCheck struct {
	Mode            string                  `json:"mode"`
	Selected        bool                    `json:"selected"`
	Status          string                  `json:"status"`
	ProtocolVersion string                  `json:"protocolVersion,omitempty"`
	LatencyMs       float64                 `json:"latencyMs"`
	ToolCount       int                     `json:"toolCount,omitempty"`
	Pages           int                     `json:"pages,omitempty"`
	ServerInfo      *mcp.Implementation     `json:"serverInfo,omitempty"`
	Capabilities    *mcp.ServerCapabilities `json:"capabilities,omitempty"`
	Error           string                  `json:"error,omitempty"`
}

// doctorReport is the collected report for both modes.
type doctorReport struct {
	URL          string        `json:"url"`
	SelectedMode string        `json:"selectedMode"`
	Healthy      bool          `json:"healthy"`
	Checks       []doctorCheck `json:"checks"`
}

// doctorServer checks both protocol modes, putting the selected mode first.
func doctorServer(ctx context.Context, url string, stateless bool) *doctorReport {
	report := &doctorReport{URL: url, SelectedMode: "legacy"}
	if stateless {
		report.SelectedMode = "stateless"
	}

	results := make(chan doctorCheck, 2)
	var wg sync.WaitGroup
	for _, mode := range []bool{stateless, !stateless} {
		mode := mode
		selected := mode == stateless
		wg.Add(1)
		go func() {
			defer wg.Done()
			results <- runDoctorCheck(ctx, url, mode, selected)
		}()
	}
	wg.Wait()
	close(results)

	for check := range results {
		report.Checks = append(report.Checks, check)
	}
	sort.SliceStable(report.Checks, func(i, j int) bool {
		return report.Checks[i].Selected && !report.Checks[j].Selected
	})
	report.Healthy = len(report.Checks) > 0 && report.Checks[0].Selected && report.Checks[0].Status == "ok"
	return report
}

// runDoctorCheck connects to the server in one mode and exercises its tools.
func runDoctorCheck(ctx context.Context, url string, stateless, selected bool) doctorCheck {
	modeName := "legacy"
	protocolVersion := legacyProtocolVersion
	if stateless {
		modeName = "stateless"
		protocolVersion = statelessProtocolVersion
	}

	check := doctorCheck{Mode: modeName, Selected: selected}
	started := time.Now()
	fail := func(err error) doctorCheck {
		check.Status = "error"
		check.LatencyMs = elapsedMs(started)
		check.Error = exceptionMessage(err)
		return check
	}

	client, err := NewClient(ClientOptions{URL: url, ProtocolVersion: protocolVersion})
	if err != nil {
		return fail(err)
	}
	defer client.Close()

	seenCursors := map[string]bool{}
	cursor := ""
	pages := 0
	toolCount := 0
	for {
		tools, next, err := client.ListToolsPage(ctx, cursor)
		if err != nil {
			return fail(err)
		}
		pages++
		toolCount += len(tools)
		cursor = next
		if cursor == "" {
			break
		}
		if seenCursors[cursor] {
			return fail(fmt.Errorf("server repeated pagination cursor %q", cursor))
		}
		seenCursors[cursor] = true
	}

	info := client.Info()
	check.Status = "ok"
	check.LatencyMs = elapsedMs(started)
	check.ProtocolVersion = info.ProtocolVersion
	check.ToolCount = toolCount
	check.Pages = pages
	check.ServerInfo = info.ServerInfo
	check.Capabilities = info.Capabilities
	return check
}

func elapsedMs(started time.Time) float64 {
	ms := float64(time.Since(started).Microseconds()) / 1000.0
	return math.Round(ms*100) / 100.0
}

// exceptionMessage returns a single-sentence error description.
func exceptionMessage(err error) string {
	if err == nil {
		return ""
	}
	return strings.TrimSpace(err.Error())
}

// capabilityNames lists the server capabilities the server supports, in a
// stable order.
func capabilityNames(capabilities *mcp.ServerCapabilities) []string {
	if capabilities == nil {
		return nil
	}
	var names []string
	if capabilities.Completions != nil {
		names = append(names, "completions")
	}
	if capabilities.Logging != nil {
		names = append(names, "logging")
	}
	if capabilities.Prompts != nil {
		names = append(names, "prompts")
	}
	if capabilities.Resources != nil {
		names = append(names, "resources")
	}
	if capabilities.Tools != nil {
		names = append(names, "tools")
	}
	return names
}

// renderDoctor renders the human-readable output of the doctor command.
func renderDoctor(report *doctorReport) string {
	var b strings.Builder
	fmt.Fprintf(&b, "URL: %s\n", report.URL)
	fmt.Fprintf(&b, "Selected mode: %s\n", report.SelectedMode)

	for _, check := range report.Checks {
		b.WriteString("\n")
		selected := ""
		if check.Selected {
			selected = " (selected)"
		}
		fmt.Fprintf(&b, "%s%s: %s\n", check.Mode, selected, check.Status)
		fmt.Fprintf(&b, "  Latency: %.2f ms\n", check.LatencyMs)
		if check.Status == "ok" {
			fmt.Fprintf(&b, "  Protocol version: %s\n", check.ProtocolVersion)
			fmt.Fprintf(&b, "  Tools: %d across %d page(s)\n", check.ToolCount, check.Pages)
			if names := capabilityNames(check.Capabilities); len(names) > 0 {
				fmt.Fprintf(&b, "  Capabilities: %s\n", strings.Join(names, ", "))
			}
		} else {
			fmt.Fprintf(&b, "  Error: %s\n", check.Error)
		}
	}

	b.WriteString("\n")
	result := "healthy"
	if !report.Healthy {
		result = "unhealthy"
	}
	fmt.Fprintf(&b, "Result: %s\n", result)
	return b.String()
}
