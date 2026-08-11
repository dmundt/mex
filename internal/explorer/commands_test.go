package explorer

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// installNewClient swaps the package-level NewClient factory so that commands
// receive the given fake client regardless of URL.
func installNewClient(t *testing.T, fake *fakeClient) {
	t.Helper()
	orig := NewClient
	NewClient = func(opts ClientOptions) (MCPClient, error) {
		fake.info = Info{
			URL:               opts.URL,
			Mode:              "stateless",
			Negotiation:       "server/discover",
			ProtocolVersion:   statelessProtocolVersion,
			SupportedVersions: []string{statelessProtocolVersion},
			ServerInfo:        &mcp.Implementation{Name: "fake", Version: "1.0"},
			Capabilities:      &mcp.ServerCapabilities{Tools: &mcp.ToolCapabilities{}},
		}
		return fake, nil
	}
	t.Cleanup(func() { NewClient = orig })
}

// installNewClientErr makes NewClient fail with err for any URL.
func installNewClientErr(t *testing.T, err error) {
	t.Helper()
	orig := NewClient
	NewClient = func(opts ClientOptions) (MCPClient, error) { return nil, err }
	t.Cleanup(func() { NewClient = orig })
}

func sampleTool(name string) *mcp.Tool {
	return &mcp.Tool{
		Name:        name,
		Description: name + " does things",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"x": map[string]any{"type": "integer"},
			},
		},
	}
}

func TestListCommand(t *testing.T) {
	fake := newFakeClient([]*mcp.Tool{sampleTool("alpha"), sampleTool("beta")}, 1)
	installNewClient(t, fake)

	cmd := newListCommand()
	out, err := executeCommand(cmd, "http://fake/mcp")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"alpha(x?: integer)", "beta(x?: integer)"} {
		if !strings.Contains(out, want) {
			t.Errorf("list output missing %q in:\n%s", want, out)
		}
	}

	cmd = newListCommand()
	out, err = executeCommand(cmd, "--json", "http://fake/mcp")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(strings.TrimSpace(out), "[") {
		t.Errorf("json list output should be an array, got:\n%s", out)
	}

	cmd = newListCommand()
	out, err = executeCommand(cmd, "-N", "http://fake/mcp")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "  Parameters:") {
		t.Errorf("no-truncate output missing parameters in:\n%s", out)
	}

	cmd = newListCommand()
	if _, err := executeCommand(cmd); err == nil {
		t.Error("expected error for missing URL")
	}
}

func TestPromptsCommand(t *testing.T) {
	fake := newFakeClient(nil, 10)
	fake.prompts = []*mcp.Prompt{{Name: "greet", Description: "Say hi", Arguments: []*mcp.PromptArgument{{Name: "name"}}}}
	installNewClient(t, fake)

	cmd := newPromptsCommand()
	out, err := executeCommand(cmd, "http://fake/mcp")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "greet") || !strings.Contains(out, "  Say hi") {
		t.Errorf("prompts output unexpected:\n%s", out)
	}

	cmd = newPromptsCommand()
	out, err = executeCommand(cmd, "--json", "http://fake/mcp")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "greet") {
		t.Errorf("json prompts output missing greet:\n%s", out)
	}
}

func TestResourcesCommand(t *testing.T) {
	fake := newFakeClient(nil, 10)
	fake.resources = []*mcp.Resource{{Name: "cfg", URI: "docs://cfg", Description: "Config file", MIMEType: "text/plain", Size: 42}}
	installNewClient(t, fake)

	cmd := newResourcesCommand()
	out, err := executeCommand(cmd, "http://fake/mcp")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"cfg", "docs://cfg", "  Config file", "  MIME type: text/plain", "  Size: 42 bytes"} {
		if !strings.Contains(out, want) {
			t.Errorf("resources output missing %q in:\n%s", want, out)
		}
	}
}

func TestInspectCommand(t *testing.T) {
	fake := newFakeClient([]*mcp.Tool{sampleTool("alpha")}, 10)
	installNewClient(t, fake)

	cmd := newInspectCommand()
	out, err := executeCommand(cmd, "http://fake/mcp", "alpha")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "alpha") || !strings.Contains(out, "Input schema:") {
		t.Errorf("inspect output unexpected:\n%s", out)
	}

	cmd = newInspectCommand()
	_, err = executeCommand(cmd, "http://fake/mcp", "missing")
	if err == nil {
		t.Fatal("expected error for missing tool")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("missing tool error = %v", err)
	}
}

func TestCallCommand(t *testing.T) {
	fake := newFakeClient([]*mcp.Tool{sampleTool("alpha")}, 10)
	fake.callDone = func(args map[string]any) *mcp.CallToolResult {
		return &mcp.CallToolResult{Content: []mcp.Content{
			&mcp.TextContent{Text: "result-text"},
		}}
	}
	installNewClient(t, fake)

	cmd := newCallCommand()
	out, err := executeCommand(cmd, "http://fake/mcp", "alpha", `{"x": 1}`)
	if err != nil {
		t.Fatal(err)
	}
	if out != "result-text\n" {
		t.Errorf("call output = %q, want %q", out, "result-text\n")
	}

	cmd = newCallCommand()
	out, _ = executeCommand(cmd, "http://fake/mcp", "alpha", "--json", `{"x": 1}`)
	if out != "result-text\n" {
		t.Errorf("call --json output = %q", out)
	}

	// -a arguments without JSON
	cmd = newCallCommand()
	_, err = executeCommand(cmd, "http://fake/mcp", "alpha", "-a", "x", "5")
	if err != nil {
		t.Fatal(err)
	}

	// usage error: --json and --raw
	cmd = newCallCommand()
	_, err = executeCommand(cmd, "http://fake/mcp", "alpha", "--json", "--raw")
	if err == nil || !strings.Contains(err.Error(), "cannot be used together") {
		t.Errorf("expected conflict error, got %v", err)
	}

	// tool-level error exits via ExitCodeError
	fake.callDone = func(args map[string]any) *mcp.CallToolResult {
		return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: "boom"}}}
	}
	cmd = newCallCommand()
	_, err = executeCommand(cmd, "http://fake/mcp", "alpha")
	if err == nil {
		t.Fatal("expected ExitCodeError for isError result")
	}
	var exitCode *ExitCodeError
	if !errors.As(err, &exitCode) || exitCode.Code != 1 {
		t.Errorf("expected ExitCodeError{1}, got %v", err)
	}
}

func TestCallCommandStdin(t *testing.T) {
	fake := newFakeClient([]*mcp.Tool{sampleTextTool("alpha")}, 10)
	fake.callDone = func(args map[string]any) *mcp.CallToolResult {
		if args["x"] == "from-stdin" {
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "ok"}}}
		}
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "nope"}}}
	}
	installNewClient(t, fake)

	cmd := newCallCommand()
	cmd.SetIn(strings.NewReader(`{"x": "from-stdin"}`))
	out, err := executeCommand(cmd, "http://fake/mcp", "alpha", "-")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "ok") {
		t.Errorf("stdin call output = %q", out)
	}
}

// sampleTextTool returns a tool whose argument is a plain string.
func sampleTextTool(name string) *mcp.Tool {
	return &mcp.Tool{
		Name:        name,
		InputSchema: map[string]any{"type": "object", "properties": map[string]any{"x": map[string]any{"type": "string"}}},
	}
}

func TestCallCommandHelpListsFlags(t *testing.T) {
	cmd := newCallCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	out := buf.String()
	for _, want := range []string{"-a, --argument NAME VALUE", "--raw", "--json", "--legacy", "--stateless"} {
		if !strings.Contains(out, want) {
			t.Errorf("call --help missing %q in:\n%s", want, out)
		}
	}
}

func TestCallCommandRegisteredFlagsDoNotBreakParsing(t *testing.T) {
	fake := newFakeClient([]*mcp.Tool{sampleTool("alpha")}, 10)
	fake.callDone = func(args map[string]any) *mcp.CallToolResult {
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "ok"}}}
	}
	installNewClient(t, fake)

	cmd := newCallCommand()
	out, err := executeCommand(cmd, "http://fake/mcp", "alpha", `{"x": 1}`, "--json")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "ok") {
		t.Errorf("json call output = %q", out)
	}
}

func TestInfoCommand(t *testing.T) {
	fake := newFakeClient(nil, 10)
	installNewClient(t, fake)

	cmd := newInfoCommand()
	out, err := executeCommand(cmd, "http://fake/mcp")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"URL: http://fake/mcp", "Mode: stateless", "Negotiation: server/discover", "Protocol version:"} {
		if !strings.Contains(out, want) {
			t.Errorf("info output missing %q in:\n%s", want, out)
		}
	}

	cmd = newInfoCommand()
	out, err = executeCommand(cmd, "--json", "http://fake/mcp")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `"mode": "stateless"`) {
		t.Errorf("json info output unexpected:\n%s", out)
	}
}

func TestDoctorCommand(t *testing.T) {
	fake := newFakeClient([]*mcp.Tool{sampleTool("alpha")}, 10)
	installNewClient(t, fake)

	cmd := newDoctorCommand()
	out, err := executeCommand(cmd, "http://fake/mcp")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Selected mode: stateless", "stateless (selected): ok", "Result: healthy"} {
		if !strings.Contains(out, want) {
			t.Errorf("doctor output missing %q in:\n%s", want, out)
		}
	}
}

func TestDoctorCommandUnhealthy(t *testing.T) {
	fake := newFakeClient(nil, 10)
	fake.listError = context.DeadlineExceeded
	installNewClient(t, fake)

	cmd := newDoctorCommand()
	_, err := executeCommand(cmd, "http://fake/mcp")
	if err == nil {
		t.Fatal("expected ExitCodeError for unhealthy doctor")
	}
	var exitCode *ExitCodeError
	if !errors.As(err, &exitCode) || exitCode.Code != 1 {
		t.Errorf("expected ExitCodeError{1}, got %v", err)
	}
}

func TestRunCommandErrors(t *testing.T) {
	installNewClientErr(t, errors.New("connection refused"))

	cmd := newListCommand()
	_, err := executeCommand(cmd, "http://bad/mcp")
	if err == nil {
		t.Fatal("expected connection error")
	}
	if !strings.Contains(err.Error(), "connection refused") {
		t.Errorf("error = %v", err)
	}
}

func TestFetchPagination(t *testing.T) {
	fake := newFakeClient([]*mcp.Tool{sampleTool("a"), sampleTool("b"), sampleTool("c")}, 1)
	ctx := context.Background()
	tools, err := listAllTools(ctx, fake)
	if err != nil {
		t.Fatal(err)
	}
	if len(tools) != 3 {
		t.Errorf("listAllTools = %d tools, want 3", len(tools))
	}
}
