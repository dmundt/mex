package explorer

import (
	"bytes"
	"context"
	"fmt"
	"sync"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
)

// fakeClient is a scriptable MCPClient for command tests.
type fakeClient struct {
	mu        sync.Mutex
	tools     []*mcp.Tool
	prompts   []*mcp.Prompt
	resources []*mcp.Resource
	info      Info

	pageSize  int
	callError error
	callDone  func(args map[string]any) *mcp.CallToolResult
	listError error
}

var _ MCPClient = (*fakeClient)(nil)

// newFakeClient returns a fake client that paginates results pageSize at a time.
func newFakeClient(tools []*mcp.Tool, pageSize int) *fakeClient {
	return &fakeClient{tools: tools, pageSize: pageSize}
}

// setInfo stores the client info. It is safe for concurrent use, which the
// doctor command exercises by connecting from two goroutines at once.
func (f *fakeClient) setInfo(info Info) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.info = info
}

func (f *fakeClient) Info() Info {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.info
}

// page slices all into pages of the given size, keyed by a cursor of the form
// "N" where N is the next starting index.
func page[T any](all []T, cursor string, size int) ([]T, string) {
	start := 0
	if cursor != "" {
		if _, err := fmt.Sscanf(cursor, "%d", &start); err != nil {
			return nil, ""
		}
	}
	if start >= len(all) {
		return nil, ""
	}
	end := start + size
	if end > len(all) {
		end = len(all)
	}
	next := ""
	if end < len(all) {
		next = fmt.Sprintf("%d", end)
	}
	return all[start:end], next
}

func (f *fakeClient) ListToolsPage(ctx context.Context, cursor string) ([]*mcp.Tool, string, error) {
	if f.listError != nil {
		return nil, "", f.listError
	}
	items, next := page(f.tools, cursor, f.pageSize)
	return items, next, nil
}

func (f *fakeClient) ListPromptsPage(ctx context.Context, cursor string) ([]*mcp.Prompt, string, error) {
	if f.listError != nil {
		return nil, "", f.listError
	}
	items, next := page(f.prompts, cursor, f.pageSize)
	return items, next, nil
}

func (f *fakeClient) ListResourcesPage(ctx context.Context, cursor string) ([]*mcp.Resource, string, error) {
	if f.listError != nil {
		return nil, "", f.listError
	}
	items, next := page(f.resources, cursor, f.pageSize)
	return items, next, nil
}

func (f *fakeClient) FindTool(ctx context.Context, name string) (*mcp.Tool, error) {
	for _, tool := range f.tools {
		if tool.Name == name {
			return tool, nil
		}
	}
	return nil, nil
}

func (f *fakeClient) CallTool(ctx context.Context, name string, arguments map[string]any) (*mcp.CallToolResult, error) {
	if f.callError != nil {
		return nil, f.callError
	}
	if f.callDone != nil {
		return f.callDone(arguments), nil
	}
	return &mcp.CallToolResult{StructuredContent: map[string]any{"called": name, "args": arguments}}, nil
}

func (f *fakeClient) Close() error { return nil }

// executeCommand runs a command's silver-arg baseline and returns output. It
// requires a temp file-path-free environment, so tests must override the
// package-level NewClient and provide a URL argument.
func executeCommand(cmd *cobra.Command, args ...string) (string, error) {
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return buf.String(), err
}
