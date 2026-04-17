package mcp

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
)

// MCPClient is a wrapper for MCP client
// It provides an object-oriented interface to interact with MCP server
type MCPClient struct {
	c *client.Client
}

// NewMCPClient creates a new MCP client instance
// httpURL: HTTP transport URL
func NewMCPClient(httpURL string) (*MCPClient, error) {
	fmt.Println("Initializing HTTP client...")
	// Create HTTP transport
	httpTransport, err := transport.NewStreamableHTTP(httpURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP transport: %w", err)
	}

	// Create client using transport
	c := client.NewClient(httpTransport)

	return &MCPClient{c: c}, nil
}

// Initialize initializes client
func (m *MCPClient) Initialize(ctx context.Context) (*mcp.InitializeResult, error) {
	// Set notification handler
	m.c.OnNotification(func(notification mcp.JSONRPCNotification) {
		fmt.Printf("Received notification: %s\n", notification.Method)
	})

	// Initialize client
	fmt.Println("Initializing client...")
	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "MCP-Go Weather Client",
		Version: "1.0.0",
	}
	initRequest.Params.Capabilities = mcp.ClientCapabilities{}

	serverInfo, err := m.c.Initialize(ctx, initRequest)
	if err != nil {
		return nil, fmt.Errorf("initialization failed: %w", err)
	}

	// Display server info
	fmt.Printf("Connected to server: %s (version %s)\n",
		serverInfo.ServerInfo.Name,
		serverInfo.ServerInfo.Version)

	return serverInfo, nil
}

// Ping executes health check
func (m *MCPClient) Ping(ctx context.Context) error {
	fmt.Println("Executing health check...")
	if err := m.c.Ping(ctx); err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	fmt.Println("Server is running and responding normally")
	return nil
}

// CallTool calls MCP tool
func (m *MCPClient) CallTool(ctx context.Context, toolName string, args map[string]any) (*mcp.CallToolResult, error) {
	callToolRequest := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      toolName,
			Arguments: args,
		},
	}

	result, err := m.c.CallTool(ctx, callToolRequest)
	if err != nil {
		return nil, fmt.Errorf("tool call failed: %w", err)
	}

	return result, nil
}

// CallWeatherTool calls get_weather tool
func (m *MCPClient) CallWeatherTool(ctx context.Context, city string) (*mcp.CallToolResult, error) {
	fmt.Printf("Querying weather for city %s...\n", city)

	callToolRequest := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "get_weather",
			Arguments: map[string]any{
				"city": city,
			},
		},
	}

	result, err := m.c.CallTool(ctx, callToolRequest)
	if err != nil {
		return nil, fmt.Errorf("tool call failed: %w", err)
	}

	return result, nil
}

// GetToolResultText gets text content from tool result
func (m *MCPClient) GetToolResultText(result *mcp.CallToolResult) string {
	var text string
	for _, content := range result.Content {
		if textContent, ok := content.(mcp.TextContent); ok {
			text += textContent.Text + "\n"
		}
	}
	return text
}

func (m *MCPClient) Close() {
	if m.c != nil {
		m.c.Close()
	}
}
