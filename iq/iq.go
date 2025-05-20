package main

import (
	"context"
	"fmt"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

func CallTool() {

	mcpClient, err := client.NewStdioMCPClient(
		"../mcpsrv/mcpsrv",
		[]string{},
	)
	if err != nil {
		panic(err)
	}
	defer mcpClient.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Println("MCP client initialing...")
	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "McpClient",
		Version: "1.0.0",
	}

	initResult, err := mcpClient.Initialize(ctx, initRequest)
	if err != nil {
		panic(err)
	}
	fmt.Printf(
		"\nmcpsrv info: %s %s\n\n",
		initResult.ServerInfo.Name,
		initResult.ServerInfo.Version,
	)

	fmt.Println("Prompt list:")
	promptsRequest := mcp.ListPromptsRequest{}
	prompts, err := mcpClient.ListPrompts(ctx, promptsRequest)
	if err != nil {
		panic(err)
	}
	for _, prompt := range prompts.Prompts {
		fmt.Printf("- %s: %s\n", prompt.Name, prompt.Description)
		fmt.Println("Params:", prompt.Arguments)
	}

	fmt.Println()
	fmt.Println("Resource list:")
	resourcesRequest := mcp.ListResourcesRequest{}
	resources, err := mcpClient.ListResources(ctx, resourcesRequest)
	if err != nil {
		panic(err)
	}
	for _, resource := range resources.Resources {
		fmt.Printf("- uri: %s, name: %s, description: %s, MIMEType: %s\n", resource.URI, resource.Name, resource.Description, resource.MIMEType)
	}

	fmt.Println()
	fmt.Println("Available tool list:")
	toolsRequest := mcp.ListToolsRequest{}
	tools, err := mcpClient.ListTools(ctx, toolsRequest)
	if err != nil {
		panic(err)
	}

	for _, tool := range tools.Tools {
		fmt.Printf("- %s: %s\n", tool.Name, tool.Description)
		fmt.Println("Params:", tool.InputSchema.Properties)
	}
	fmt.Println()

	fmt.Println("Call the tool: AbnormalDetect")
	toolRequest := mcp.CallToolRequest{
		Request: mcp.Request{
			Method: "tools/call",
		},
	}
	toolRequest.Params.Name = "AbnormalDetect"
	toolRequest.Params.Arguments = map[string]any{
		"operation": "add",
		"para":      1,
		"parb":      1,
	}
	// Call the tool
	result, err := mcpClient.CallTool(ctx, toolRequest)
	if err != nil {
		panic(err)
	}
	fmt.Println("Tool result:", result.Content[0].(mcp.TextContent).Text)
}
