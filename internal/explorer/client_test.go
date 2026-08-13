package explorer

import (
	"context"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestInfoStateless(t *testing.T) {
	ctx := context.Background()
	client, err := Connect(ctx, startTestServer(t), ClientOptions{ProtocolVersion: statelessProtocolVersion})
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer client.Close()

	info := client.Info()
	if info.Mode != "stateless" || info.Negotiation != "server/discover" {
		t.Errorf("mode/negotiation = %q/%q", info.Mode, info.Negotiation)
	}
	if info.ServerInfo == nil || info.ServerInfo.Name != "test" {
		t.Errorf("serverInfo = %+v", info.ServerInfo)
	}
	if info.Capabilities == nil {
		t.Error("capabilities are nil")
	}
}

func TestInfoLegacyModeFlags(t *testing.T) {
	ctx := context.Background()
	client, err := Connect(ctx, startTestServer(t), ClientOptions{ProtocolVersion: legacyProtocolVersion})
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer client.Close()

	info := client.Info()
	if info.Mode != "legacy" || info.Negotiation != "initialize" {
		t.Errorf("legacy mode = %q/%q, want legacy/initialize", info.Mode, info.Negotiation)
	}
	if info.ProtocolVersion == "" {
		t.Error("protocol version is empty")
	}
}

func TestNewTransport(t *testing.T) {
	httpT, err := newTransport("https://example.test/mcp")
	if err != nil {
		t.Fatalf("http transport: %v", err)
	}
	if _, ok := httpT.(*mcp.StreamableClientTransport); !ok {
		t.Errorf("http transport type = %T", httpT)
	}

	stdioT, err := newTransport(`node server.js --port 9`)
	if err != nil {
		t.Fatalf("stdio transport: %v", err)
	}
	if _, ok := stdioT.(*mcp.CommandTransport); !ok {
		t.Errorf("stdio transport type = %T", stdioT)
	}
}

func TestNewCommandTransportErrors(t *testing.T) {
	if _, err := newCommandTransport(""); err == nil {
		t.Fatal("expected error for empty command")
	}
	if _, err := newCommandTransport("   "); err == nil {
		t.Fatal("expected error for whitespace command")
	}
	if _, err := newCommandTransport("unbalanced \"quote"); err == nil {
		t.Fatal("expected error for unterminated quote")
	}
}

func TestParseURL(t *testing.T) {
	for url, wantScheme := range map[string]string{
		"http://x": "http",
		"ws://x":   "ws",
		"stdio:":   "stdio",
	} {
		got, err := parseURL(url)
		if err != nil {
			t.Errorf("parseURL(%q) unexpected error: %v", url, err)
			continue
		}
		if got != wantScheme {
			t.Errorf("parseURL(%q) = %q, want %q", url, got, wantScheme)
		}
	}
	for _, url := range []string{"", ":nope", "ba d:", "no-scheme"} {
		if _, err := parseURL(url); err == nil {
			t.Errorf("parseURL(%q) expected error", url)
		}
	}
}

func TestMCPClientMethods(t *testing.T) {
	ctx := context.Background()
	client, err := Connect(ctx, startTestServer(t), ClientOptions{ProtocolVersion: statelessProtocolVersion})
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer client.Close()

	tools, _, err := client.ListToolsPage(ctx, "")
	if err != nil || len(tools) != 1 {
		t.Fatalf("ListToolsPage = %v, %v", tools, err)
	}
	prompts, _, err := client.ListPromptsPage(ctx, "")
	if err != nil || len(prompts) != 1 {
		t.Fatalf("ListPromptsPage = %v, %v", prompts, err)
	}
	resources, _, err := client.ListResourcesPage(ctx, "")
	if err != nil || len(resources) != 1 {
		t.Fatalf("ListResourcesPage = %v, %v", resources, err)
	}
}

// repeatingCursorClient is an MCPClient whose ListToolsPage always returns the
// same non-empty cursor, which would loop forever without cycle detection.
type repeatingCursorClient struct{}

func (repeatingCursorClient) Info() Info { return Info{} }
func (repeatingCursorClient) ListToolsPage(context.Context, string) ([]*mcp.Tool, string, error) {
	return nil, "loop", nil
}
func (repeatingCursorClient) ListPromptsPage(context.Context, string) ([]*mcp.Prompt, string, error) {
	return nil, "loop", nil
}
func (repeatingCursorClient) ListResourcesPage(context.Context, string) ([]*mcp.Resource, string, error) {
	return nil, "loop", nil
}
func (repeatingCursorClient) FindTool(context.Context, string) (*mcp.Tool, error) { return nil, nil }
func (repeatingCursorClient) CallTool(context.Context, string, map[string]any) (*mcp.CallToolResult, error) {
	return nil, nil
}
func (repeatingCursorClient) Close() error { return nil }

func TestPaginationCycleDetected(t *testing.T) {
	ctx := context.Background()
	_, err := listAllTools(ctx, repeatingCursorClient{})
	if err == nil || !strings.Contains(err.Error(), "repeated pagination cursor") {
		t.Fatalf("listAllTools error = %v, want repeated cursor error", err)
	}
}

func TestModeFromVersion(t *testing.T) {
	mode, negotiation := modeFromVersion(statelessProtocolVersion)
	if mode != "stateless" || negotiation != "server/discover" {
		t.Errorf("modeFromVersion(stateless) = %q/%q", mode, negotiation)
	}
	mode, negotiation = modeFromVersion(legacyProtocolVersion)
	if mode != "legacy" || negotiation != "initialize" {
		t.Errorf("modeFromVersion(legacy) = %q/%q", mode, negotiation)
	}
}

func TestWriteJSON(t *testing.T) {
	var sb strings.Builder
	cmd := newRootCommand()
	cmd.SetOut(&sb)
	cmd.SetErr(&sb)
	if err := writeJSON(cmd, map[string]string{"a": "b"}); err != nil {
		t.Fatalf("writeJSON: %v", err)
	}
	out := sb.String()
	for _, want := range []string{`"a": "b"`, "\n"} {
		if !strings.Contains(out, want) {
			t.Errorf("writeJSON output %q missing %q", out, want)
		}
	}
}
