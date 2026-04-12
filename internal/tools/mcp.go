package tools

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino-ext/components/tool/mcp/officialmcp"
	"github.com/cloudwego/eino/components/tool"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// LoadMCPTools fetches tools from a connected MCP server session.
// If toolNames is empty, all available tools are returned.
func LoadMCPTools(ctx context.Context, session *mcp.ClientSession, toolNames []string) ([]tool.BaseTool, error) {
	tools, err := officialmcp.GetTools(ctx, &officialmcp.Config{
		Cli:          session,
		ToolNameList: toolNames,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to load MCP tools: %w", err)
	}
	return tools, nil
}
