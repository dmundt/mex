package explorer

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// startTestServer runs an in-memory MCP server with a tool, a prompt, and a
// resource, and returns the transport to use for a client connection.
func startTestServer(t *testing.T) mcp.Transport {
	t.Helper()
	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "1.0"}, nil)

	server.AddTool(&mcp.Tool{
		Name:        "add",
		Description: "Add two numbers",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"a": map[string]any{"type": "integer"},
				"b": map[string]any{"type": "integer"},
			},
			"required": []any{"a", "b"},
		},
	}, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return &mcp.CallToolResult{
			StructuredContent: map[string]any{"sum": json.Number("3")},
		}, nil
	})

	server.AddPrompt(&mcp.Prompt{
		Name:        "greet",
		Description: "Say hello",
		Arguments:   []*mcp.PromptArgument{{Name: "name", Required: true}},
	}, func(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		return &mcp.GetPromptResult{
			Description: "hi",
			Messages:    []*mcp.PromptMessage{},
		}, nil
	})

	server.AddResource(&mcp.Resource{
		Name:        "readme",
		URI:         "docs://readme",
		Description: "Project readme",
		MIMEType:    "text/markdown",
	}, func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{{URI: "docs://readme", Text: "# hi"}},
		}, nil
	})

	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	serverSession, err := server.Connect(context.Background(), serverTransport, nil)
	if err != nil {
		t.Fatalf("server connect: %v", err)
	}
	t.Cleanup(func() { serverSession.Close() })
	return clientTransport
}

func TestConnectAndList(c *testing.T) {
	ctx := context.Background()
	clientTransport := startTestServer(c)

	client, err := Connect(ctx, clientTransport, ClientOptions{ProtocolVersion: statelessProtocolVersion})
	if err != nil {
		c.Fatalf("Connect: %v", err)
	}
	defer client.Close()

	tools, next, err := client.ListToolsPage(ctx, "")
	if err != nil {
		c.Fatalf("ListToolsPage: %v", err)
	}
	if len(tools) != 1 || tools[0].Name != "add" {
		c.Fatalf("ListToolsPage tools = %v", tools)
	}
	if next != "" {
		c.Fatalf("ListToolsPage next cursor = %q, want empty", next)
	}

	found, err := client.FindTool(ctx, "add")
	if err != nil || found == nil {
		c.Fatalf("FindTool(add) = %v, %v", found, err)
	}
	missing, err := client.FindTool(ctx, "nope")
	if err != nil || missing != nil {
		c.Fatalf("FindTool(nope) = %v, %v", missing, err)
	}

	prompts, err := listAllPrompts(ctx, client)
	if err != nil {
		c.Fatalf("listAllPrompts: %v", err)
	}
	if len(prompts) != 1 || prompts[0].Name != "greet" {
		c.Fatalf("prompts = %v", prompts)
	}

	resources, err := listAllResources(ctx, client)
	if err != nil {
		c.Fatalf("listAllResources: %v", err)
	}
	if len(resources) != 1 || resources[0].URI != "docs://readme" {
		c.Fatalf("resources = %v", resources)
	}

	result, err := client.CallTool(ctx, "add", map[string]any{"a": 1, "b": 2})
	if err != nil {
		c.Fatalf("CallTool: %v", err)
	}
	if result.StructuredContent == nil {
		c.Fatalf("CallTool result = %v", result)
	}
}

func TestInvokeTool(t *testing.T) {
	ctx := context.Background()
	clientTransport := startTestServer(t)
	client, err := Connect(ctx, clientTransport, ClientOptions{ProtocolVersion: statelessProtocolVersion})
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer client.Close()

	result, err := invokeTool(ctx, client, "add", nil, [][2]string{{"a", "1"}, {"b", "2"}})
	if err != nil {
		t.Fatalf("invokeTool: %v", err)
	}
	if result.StructuredContent == nil {
		t.Fatalf("result = %v", result)
	}

	if _, err := invokeTool(ctx, client, "nope", nil, nil); err == nil {
		t.Fatal("expected error for missing tool")
	}
}

func TestOutputFormats(t *testing.T) {
	ctx := context.Background()
	clientTransport := startTestServer(t)
	client, err := Connect(ctx, clientTransport, ClientOptions{ProtocolVersion: statelessProtocolVersion})
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer client.Close()

	result, err := client.CallTool(ctx, "add", map[string]any{"a": 1, "b": 2})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}

	text := renderToolList([]*mcp.Tool{{
		Name:        "add",
		Description: "Add two numbers",
		InputSchema: map[string]any{"type": "object"},
	}}, false)
	if text == "" {
		t.Fatal("empty renderToolList output")
	}

	info := renderCallResult(result)
	if info == "" {
		t.Fatal("empty renderCallResult output")
	}
}
