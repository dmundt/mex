package explorer

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// listAllTools follows pagination until all tools have been collected.
func listAllTools(ctx context.Context, c MCPClient) ([]*mcp.Tool, error) {
	var out []*mcp.Tool
	cursor := ""
	for {
		tools, next, err := c.ListToolsPage(ctx, cursor)
		if err != nil {
			return nil, err
		}
		out = append(out, tools...)
		cursor = next
		if cursor == "" {
			return out, nil
		}
	}
}

// listAllPrompts follows pagination until all prompts have been collected.
func listAllPrompts(ctx context.Context, c MCPClient) ([]*mcp.Prompt, error) {
	var out []*mcp.Prompt
	cursor := ""
	for {
		prompts, next, err := c.ListPromptsPage(ctx, cursor)
		if err != nil {
			return nil, err
		}
		out = append(out, prompts...)
		cursor = next
		if cursor == "" {
			return out, nil
		}
	}
}

// listAllResources follows pagination until all resources have been collected.
func listAllResources(ctx context.Context, c MCPClient) ([]*mcp.Resource, error) {
	var out []*mcp.Resource
	cursor := ""
	for {
		resources, next, err := c.ListResourcesPage(ctx, cursor)
		if err != nil {
			return nil, err
		}
		out = append(out, resources...)
		cursor = next
		if cursor == "" {
			return out, nil
		}
	}
}