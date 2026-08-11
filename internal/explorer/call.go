package explorer

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// invokeTool finds, validates arguments for, and invokes a tool.
func invokeTool(ctx context.Context, client MCPClient, name string, argumentsJSON *string, argumentPairs [][2]string) (*mcp.CallToolResult, error) {
	tool, err := client.FindTool(ctx, name)
	if err != nil {
		return nil, err
	}
	if tool == nil {
		return nil, fmt.Errorf("Tool '%s' not found.", name)
	}
	arguments, err := buildArguments(toolInputSchema(tool), argumentsJSON, argumentPairs)
	if err != nil {
		return nil, err
	}
	return client.CallTool(ctx, name, arguments)
}
