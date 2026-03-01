package tools

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/cloudwego/eino-ext/components/tool/mcp/officialmcp"
	"github.com/cloudwego/eino/components/tool"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"starxo/internal/config"
)

// ConnectMCPServer establishes a connection to an MCP server based on the
// provided configuration. It supports two transport types:
//   - "stdio": starts a subprocess and communicates over stdin/stdout
//   - "sse": connects to an HTTP SSE endpoint
func ConnectMCPServer(ctx context.Context, cfg config.MCPServerConfig) (*mcp.ClientSession, error) {
	client := mcp.NewClient(&mcp.Implementation{
		Name:    cfg.Name,
		Version: "1.0.0",
	}, nil)

	var transport mcp.Transport

	switch cfg.Transport {
	case "stdio":
		if cfg.Command == "" {
			return nil, fmt.Errorf("MCP server %q: command is required for stdio transport", cfg.Name)
		}
		cmd := exec.CommandContext(ctx, cfg.Command, cfg.Args...)
		for k, v := range cfg.Env {
			cmd.Env = append(cmd.Env, k+"="+v)
		}
		transport = &mcp.CommandTransport{
			Command: cmd,
		}

	case "sse":
		if cfg.URL == "" {
			return nil, fmt.Errorf("MCP server %q: URL is required for SSE transport", cfg.Name)
		}
		transport = &mcp.SSEClientTransport{
			Endpoint: cfg.URL,
		}

	default:
		return nil, fmt.Errorf("MCP server %q: unsupported transport type %q (supported: stdio, sse)", cfg.Name, cfg.Transport)
	}

	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MCP server %q: %w", cfg.Name, err)
	}

	return session, nil
}

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
