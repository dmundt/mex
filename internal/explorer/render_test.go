package explorer

import (
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestRenderToolList(t *testing.T) {
	tools := []*mcp.Tool{
		{
			Name:        "add",
			Title:       "Add",
			Description: "Add two numbers",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"a": map[string]any{"type": "integer"},
					"b": map[string]any{"type": "integer"},
				},
				"required": []any{"a"},
			},
		},
	}

	got := renderToolList(tools, false)
	want := "add(a: integer, b?: integer) - Add\n  Add two numbers\n"
	if got != want {
		t.Errorf("renderToolList(truncated) = %q, want %q", got, want)
	}

	got = renderToolList(tools, true)
	for _, part := range []string{"add - Add", "Add two numbers", "  Parameters:", "a (integer, required)", "b (integer, optional)"} {
		if !strings.Contains(got, part) {
			t.Errorf("renderToolList(no-truncate) missing %q in:\n%s", part, got)
		}
	}

	if got := renderToolList(nil, false); got != "No tools available.\n" {
		t.Errorf("renderToolList(nil) = %q, want no-tools message", got)
	}
}

func TestRenderParameters(t *testing.T) {
	none := renderParameters(map[string]any{"type": "object"})
	if !strings.Contains(none, "Parameters: none") {
		t.Errorf("renderParameters(empty) = %q", none)
	}

	withDesc := renderParameters(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"multi": map[string]any{
				"type":        "string",
				"description": "line one\nline two",
			},
		},
	})
	for _, part := range []string{"multi (string, optional)", "line one", "line two"} {
		if !strings.Contains(withDesc, part) {
			t.Errorf("renderParameters missing %q in:\n%s", part, withDesc)
		}
	}
}

func TestSplitLines(t *testing.T) {
	got := splitLines("a\r\nb\nc\n")
	want := []string{"a", "b", "c"}
	if len(got) != len(want) {
		t.Fatalf("splitLines() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("splitLines() = %v, want %v", got, want)
		}
	}
}

func TestRenderToolListEmpty(t *testing.T) {
	if got := renderToolList(nil, false); got != "No tools available.\n" {
		t.Errorf("renderToolList(nil) = %q", got)
	}
}

func TestRenderToolInspect(t *testing.T) {
	tool := &mcp.Tool{
		Name:        "add",
		Description: "Add digits",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"a": map[string]any{"type": "integer"},
			},
		},
		OutputSchema: map[string]any{"type": "object"},
	}
	got := renderToolInspect(tool)
	for _, part := range []string{"add\n  Add digits\n", "Input schema:", "Output schema:"} {
		if !strings.Contains(got, part) {
			t.Errorf("renderToolInspect() missing %q in:\n%s", part, got)
		}
	}
	if strings.Count(got, "\n") < 10 {
		t.Errorf("renderToolInspect() looks too short:\n%s", got)
	}
}

func TestRenderPromptList(t *testing.T) {
	prompts := []*mcp.Prompt{
		{
			Name:        "greet",
			Title:       "Greeting",
			Description: "Say hello",
			Arguments: []*mcp.PromptArgument{
				{Name: "name", Description: "who to greet", Required: true},
			},
		},
		{
			Name:      "noparams",
			Arguments: nil,
		},
	}
	got := renderPromptList(prompts)
	for _, part := range []string{"greet - Greeting", "  Say hello", "  Arguments:", "    name (required)", "noparams", "  Arguments: none"} {
		if !strings.Contains(got, part) {
			t.Errorf("renderPromptList() missing %q in:\n%s", part, got)
		}
	}
}

func TestRenderResourceList(t *testing.T) {
	resources := []*mcp.Resource{
		{
			Name:        "readme",
			Title:       "README",
			URI:         "file:///README.md",
			Description: "Project readme",
			MIMEType:    "text/markdown",
			Size:        123,
		},
	}
	got := renderResourceList(resources)
	for _, part := range []string{"readme - README", "  file:///README.md", "  Project readme", "  MIME type: text/markdown", "  Size: 123 bytes"} {
		if !strings.Contains(got, part) {
			t.Errorf("renderResourceList() missing %q in:\n%s", part, got)
		}
	}
	if !strings.HasPrefix(got, "readme - README\n") {
		t.Errorf("renderResourceList() must start with the heading:\n%s", got)
	}
}

func TestRenderResourceListMinimal(t *testing.T) {
	got := renderResourceList([]*mcp.Resource{{Name: "bare", URI: "docs://bare"}})
	if strings.Contains(got, "MIME type") || strings.Contains(got, "Size:") {
		t.Errorf("minimal resource should not render mime/size lines:\n%s", got)
	}
	if !strings.Contains(got, "bare") || !strings.Contains(got, "docs://bare") {
		t.Errorf("minimal resource missing heading/uri:\n%s", got)
	}
}

func TestRenderCallResult(t *testing.T) {
	tests := []struct {
		name   string
		result *mcp.CallToolResult
		want   string
	}{
		{
			name: "text",
			result: &mcp.CallToolResult{Content: []mcp.Content{
				&mcp.TextContent{Text: "hello"},
			}},
			want: "hello\n",
		},
		{
			name:   "structured",
			result: &mcp.CallToolResult{StructuredContent: map[string]any{"ok": true}},
			want:   "Structured content:\n",
		},
		{
			name: "image",
			result: &mcp.CallToolResult{Content: []mcp.Content{
				&mcp.ImageContent{MIMEType: "image/png", Data: []byte("abcd")},
			}},
			want: "[image: image/png, 4 base64 characters]\n",
		},
		{
			name: "audio",
			result: &mcp.CallToolResult{Content: []mcp.Content{
				&mcp.AudioContent{MIMEType: "audio/wav", Data: []byte("abcdef")},
			}},
			want: "[audio: audio/wav, 6 base64 characters]\n",
		},
		{
			name: "resource link",
			result: &mcp.CallToolResult{Content: []mcp.Content{
				&mcp.ResourceLink{Name: "doc", URI: "docs://a"},
			}},
			want: "[resource: doc (docs://a)]\n",
		},
		{
			name: "embedded text resource",
			result: &mcp.CallToolResult{Content: []mcp.Content{
				&mcp.EmbeddedResource{Resource: &mcp.ResourceContents{URI: "docs://a", Text: "embedded text"}},
			}},
			want: "embedded text\n",
		},
		{
			name: "embedded blob resource",
			result: &mcp.CallToolResult{Content: []mcp.Content{
				&mcp.EmbeddedResource{Resource: &mcp.ResourceContents{URI: "docs://b", Blob: []byte("abc")}},
			}},
			want: "[resource: docs://b, application/octet-stream, 3 base64 characters]\n",
		},
		{
			name: "embedded resource with mime",
			result: &mcp.CallToolResult{Content: []mcp.Content{
				&mcp.EmbeddedResource{Resource: &mcp.ResourceContents{URI: "docs://c", MIMEType: "image/png", Blob: []byte("zz")}},
			}},
			want: "[resource: docs://c, image/png, 2 base64 characters]\n",
		},
		{
			name: "embedded nil resource",
			result: &mcp.CallToolResult{Content: []mcp.Content{
				&mcp.EmbeddedResource{},
			}},
			want: "[resource: (unknown)]\n",
		},
		{
			name: "unknown content type falls to JSON",
			result: &mcp.CallToolResult{Content: []mcp.Content{
				&mcp.ToolUseContent{Name: "t", Input: map[string]any{"a": float64(1)}}, //nolint:staticcheck // deprecated in SDK; still a valid content type
			}},
			want: "Content:\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderCallResult(tt.result)
			if !strings.Contains(got, tt.want) {
				t.Errorf("renderCallResult(%s) = %q, want contains %q", tt.name, got, tt.want)
			}
		})
	}
}

func TestRenderCallResultStructuredOnly(t *testing.T) {
	got := renderCallResult(&mcp.CallToolResult{StructuredContent: map[string]any{"a": float64(1)}})
	for _, want := range []string{"Structured content:", `"a": 1`} {
		if !strings.Contains(got, want) {
			t.Errorf("renderCallResult() missing %q in:\n%s", want, got)
		}
	}
}

func TestRenderInfo(t *testing.T) {
	info := Info{
		URL:               "http://example.com",
		Mode:              "stateless",
		Negotiation:       "server/discover",
		ProtocolVersion:   "2026-07-28",
		SupportedVersions: []string{"2026-07-28", "2025-11-25"},
		ServerInfo:        &mcp.Implementation{Name: "demo", Version: "1"},
		Instructions:      "Line one\n\nLine two",
	}
	got := renderInfo(info)
	for _, part := range []string{"URL: http://example.com", "Mode: stateless", "Negotiation: server/discover", "Protocol version: 2026-07-28", "Supported versions: 2026-07-28, 2025-11-25", "Server info:", `"name": "demo"`, "Instructions:", "  Line one", "  Line two"} {
		if !strings.Contains(got, part) {
			t.Errorf("renderInfo() missing %q in:\n%s", part, got)
		}
	}
}

func TestRenderDoctor(t *testing.T) {
	report := &doctorReport{
		URL:          "http://example.com",
		SelectedMode: "legacy",
		Healthy:      true,
		Checks: []doctorCheck{
			{Mode: "legacy", Selected: true, Status: "ok", ProtocolVersion: "2025-11-25", LatencyMs: 1.5, ToolCount: 3, Pages: 1},
			{Mode: "stateless", Selected: false, Status: "ok", ProtocolVersion: "2026-07-28", LatencyMs: 2.0, ToolCount: 3, Pages: 1},
		},
	}
	got := renderDoctor(report)
	want := "URL: http://example.com\nSelected mode: legacy\n\nlegacy (selected): ok\n  Latency: 1.50 ms\n  Protocol version: 2025-11-25\n  Tools: 3 across 1 page(s)\n\nstateless: ok\n  Latency: 2.00 ms\n  Protocol version: 2026-07-28\n  Tools: 3 across 1 page(s)\n\nResult: healthy\n"
	if got != want {
		t.Errorf("renderDoctor() = %q, want %q", got, want)
	}
}

func TestMarshalJSONIndent(t *testing.T) {
	data, err := marshalJSONIndent(map[string]any{"a": 1})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"<",`) && !strings.Contains(string(data), "{\n") {
		t.Errorf("marshalJSONIndent() unexpected: %s", data)
	}
}

func TestToolHeadingTitleFallback(t *testing.T) {
	tool := &mcp.Tool{Name: "no_title"}
	if got := toolHeading(tool, false); got != "no_title" {
		t.Errorf("toolHeading(no title) = %q", got)
	}
}
