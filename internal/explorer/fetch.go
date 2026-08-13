package explorer

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// paginate is a helper for following server pagination until it is exhausted. It
// fails when the server repeats a non-empty cursor, which would otherwise loop
// forever.
func paginate[T any](nextPage func(cursor string) ([]T, string, error)) ([]T, error) {
	var out []T
	cursor := ""
	seenCursors := map[string]bool{}
	for {
		items, next, err := nextPage(cursor)
		if err != nil {
			return nil, err
		}
		out = append(out, items...)
		cursor = next
		if cursor == "" {
			return out, nil
		}
		if seenCursors[cursor] {
			return nil, fmt.Errorf("server repeated pagination cursor %q", cursor)
		}
		seenCursors[cursor] = true
	}
}

// listAllTools follows pagination until all tools have been collected.
func listAllTools(ctx context.Context, c MCPClient) ([]*mcp.Tool, error) {
	return paginate(func(cursor string) ([]*mcp.Tool, string, error) {
		return c.ListToolsPage(ctx, cursor)
	})
}

// listAllPrompts follows pagination until all prompts have been collected.
func listAllPrompts(ctx context.Context, c MCPClient) ([]*mcp.Prompt, error) {
	return paginate(func(cursor string) ([]*mcp.Prompt, string, error) {
		return c.ListPromptsPage(ctx, cursor)
	})
}

// listAllResources follows pagination until all resources have been collected.
func listAllResources(ctx context.Context, c MCPClient) ([]*mcp.Resource, error) {
	return paginate(func(cursor string) ([]*mcp.Resource, string, error) {
		return c.ListResourcesPage(ctx, cursor)
	})
}
