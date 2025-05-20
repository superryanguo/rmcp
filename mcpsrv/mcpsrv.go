package main

import (
	"context"
	"encoding/base64"
	"flag"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type ToolName string
type PromptName string

const (
	AbnormalDetect         ToolName = "AbnormalDetect"
	LONG_RUNNING_OPERATION ToolName = "longRunningOperation"
)

const (
	SIMPLE  PromptName = "simple_prompt"
	COMPLEX PromptName = "complex_prompt"
)

func NewMCPServer() *server.MCPServer {

	hooks := &server.Hooks{}

	//hooks.AddBeforeAny(func(ctx context.Context, id any, method mcp.MCPMethod, message any) {
	//fmt.Printf("beforeAny: %s, %v, %v\n", method, id, message)
	//})
	//hooks.AddOnSuccess(func(ctx context.Context, id any, method mcp.MCPMethod, message any, result any) {
	//fmt.Printf("onSuccess: %s, %v, %v, %v\n", method, id, message, result)
	//})
	//hooks.AddOnError(func(ctx context.Context, id any, method mcp.MCPMethod, message any, err error) {
	//fmt.Printf("onError: %s, %v, %v, %v\n", method, id, message, err)
	//})
	//hooks.AddBeforeInitialize(func(ctx context.Context, id any, message *mcp.InitializeRequest) {
	//fmt.Printf("beforeInitialize: %v, %v\n", id, message)
	//})
	//hooks.AddOnRequestInitialization(func(ctx context.Context, id any, message any) error {
	//fmt.Printf("AddOnRequestInitialization: %v, %v\n", id, message)
	//// authorization verification and other preprocessing tasks are performed.
	//return nil
	//})
	//hooks.AddAfterInitialize(func(ctx context.Context, id any, message *mcp.InitializeRequest, result *mcp.InitializeResult) {
	//fmt.Printf("afterInitialize: %v, %v, %v\n", id, message, result)
	//})
	hooks.AddAfterCallTool(func(ctx context.Context, id any, message *mcp.CallToolRequest, result *mcp.CallToolResult) {
		fmt.Printf("afterCallTool: %v, %v, %v\n", id, message, result)
	})
	hooks.AddBeforeCallTool(func(ctx context.Context, id any, message *mcp.CallToolRequest) {
		fmt.Printf("beforeCallTool: %v, %v\n", id, message)
	})

	mcpServer := server.NewMCPServer(
		"SmartgwLLMHandler",
		"1.0.0",
		server.WithResourceCapabilities(true, true),
		server.WithPromptCapabilities(true),
		server.WithToolCapabilities(true),
		server.WithLogging(),
		server.WithHooks(hooks),
	)

	mcpServer.AddResource(mcp.NewResource("test://static/resource",
		"Static Resource",
		mcp.WithMIMEType("text/plain"),
	), handleReadResource)

	mcpServer.AddResourceTemplate(
		mcp.NewResourceTemplate(
			"test://dynamic/resource/{id}",
			"Dynamic Resource",
		),
		handleResourceTemplate,
	)

	resources := generateResources()
	for _, resource := range resources {
		mcpServer.AddResource(resource, handleGeneratedResource)
	}

	mcpServer.AddPrompt(mcp.NewPrompt(string(SIMPLE),
		mcp.WithPromptDescription("A simple prompt"),
	), handleSimplePrompt)

	mcpServer.AddPrompt(mcp.NewPrompt(string(COMPLEX),
		mcp.WithPromptDescription("A complex prompt"),
		mcp.WithArgument("temperature",
			mcp.ArgumentDescription("The temperature parameter for generation"),
			mcp.RequiredArgument(),
		),
		mcp.WithArgument("style",
			mcp.ArgumentDescription("The style to use for the response"),
			mcp.RequiredArgument(),
		),
	), handleComplexPrompt)

	mcpServer.AddTool(mcp.NewTool(string(AbnormalDetect),
		mcp.WithDescription("AbnormalDetect of the smart gateway"),
		mcp.WithNumber("para",
			mcp.Description("First parameter"),
			mcp.Required(),
		),
		mcp.WithNumber("parb",
			mcp.Description("Second parameter"),
			mcp.Required(),
		),
	), handleAbnormalDetectTool)

	mcpServer.AddTool(mcp.NewTool(
		string(LONG_RUNNING_OPERATION),
		mcp.WithDescription(
			"Demonstrates a long running operation with progress updates",
		),
		mcp.WithNumber("duration",
			mcp.Description("Duration of the operation in seconds"),
			mcp.DefaultNumber(10),
		),
		mcp.WithNumber("steps",
			mcp.Description("Number of steps in the operation"),
			mcp.DefaultNumber(5),
		),
	), handleLongRunningOperationTool)

	mcpServer.AddNotificationHandler("notification", handleNotification)

	return mcpServer
}

func handleAbnormalDetectTool(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	arguments := request.Params.Arguments
	a, ok1 := arguments["para"].(float64)
	b, ok2 := arguments["parb"].(float64)
	if !ok1 || !ok2 {
		return nil, fmt.Errorf("invalid number arguments")
	}
	sum := a + b
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: fmt.Sprintf("The sum of %f and %f is %f.", a, b, sum),
			},
		},
	}, nil
}

func handleReadResource(
	ctx context.Context,
	request mcp.ReadResourceRequest,
) ([]mcp.ResourceContents, error) {
	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      "test://static/resource",
			MIMEType: "text/plain",
			Text:     "This is a sample resource",
		},
	}, nil
}

func handleResourceTemplate(
	ctx context.Context,
	request mcp.ReadResourceRequest,
) ([]mcp.ResourceContents, error) {
	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      request.Params.URI,
			MIMEType: "text/plain",
			Text:     "This is a sample resource",
		},
	}, nil
}

func handleGeneratedResource(
	ctx context.Context,
	request mcp.ReadResourceRequest,
) ([]mcp.ResourceContents, error) {
	uri := request.Params.URI

	var resourceNumber string
	if _, err := fmt.Sscanf(uri, "test://static/resource/%s", &resourceNumber); err != nil {
		return nil, fmt.Errorf("invalid resource URI format: %w", err)
	}

	num, err := strconv.Atoi(resourceNumber)
	if err != nil {
		return nil, fmt.Errorf("invalid resource number: %w", err)
	}

	index := num - 1

	if index%2 == 0 {
		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      uri,
				MIMEType: "text/plain",
				Text:     fmt.Sprintf("Text content for resource %d", num),
			},
		}, nil
	} else {
		return []mcp.ResourceContents{
			mcp.BlobResourceContents{
				URI:      uri,
				MIMEType: "application/octet-stream",
				Blob:     base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("Binary content for resource %d", num))),
			},
		}, nil
	}
}

func generateResources() []mcp.Resource {
	resources := make([]mcp.Resource, 100)
	for i := 0; i < 100; i++ {
		uri := fmt.Sprintf("test://static/resource/%d", i+1)
		if i%2 == 0 {
			resources[i] = mcp.Resource{
				URI:      uri,
				Name:     fmt.Sprintf("Resource %d", i+1),
				MIMEType: "text/plain",
			}
		} else {
			resources[i] = mcp.Resource{
				URI:      uri,
				Name:     fmt.Sprintf("Resource %d", i+1),
				MIMEType: "application/octet-stream",
			}
		}
	}
	return resources
}

func handleSimplePrompt(
	ctx context.Context,
	request mcp.GetPromptRequest,
) (*mcp.GetPromptResult, error) {
	return &mcp.GetPromptResult{
		Description: "A simple prompt without arguments",
		Messages: []mcp.PromptMessage{
			{
				Role: mcp.RoleUser,
				Content: mcp.TextContent{
					Type: "text",
					Text: "This is a simple prompt without arguments.",
				},
			},
		},
	}, nil
}

func handleComplexPrompt(
	ctx context.Context,
	request mcp.GetPromptRequest,
) (*mcp.GetPromptResult, error) {
	arguments := request.Params.Arguments
	return &mcp.GetPromptResult{
		Description: "A complex prompt with arguments",
		Messages: []mcp.PromptMessage{
			{
				Role: mcp.RoleUser,
				Content: mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf(
						"This is a complex prompt with arguments: temperature=%s, style=%s",
						arguments["temperature"],
						arguments["style"],
					),
				},
			},
			{
				Role: mcp.RoleAssistant,
				Content: mcp.TextContent{
					Type: "text",
					Text: "I understand. You've provided a complex prompt with temperature and style arguments. How would you like me to proceed?",
				},
			},
		},
	}, nil
}

func handleLongRunningOperationTool(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	//arguments := request.GetArguments()
	arguments := request.Params.Arguments
	progressToken := request.Params.Meta.ProgressToken
	duration, _ := arguments["duration"].(float64)
	steps, _ := arguments["steps"].(float64)
	stepDuration := duration / steps
	server := server.ServerFromContext(ctx)

	for i := 1; i < int(steps)+1; i++ {
		time.Sleep(time.Duration(stepDuration * float64(time.Second)))
		if progressToken != nil {
			err := server.SendNotificationToClient(
				ctx,
				"notifications/progress",
				map[string]any{
					"progress":      i,
					"total":         int(steps),
					"progressToken": progressToken,
					"message":       fmt.Sprintf("Server progress %v%%", int(float64(i)*100/steps)),
				},
			)
			if err != nil {
				return nil, fmt.Errorf("failed to send notification: %w", err)
			}
		}
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: fmt.Sprintf(
					"Long running operation completed. Duration: %f seconds, Steps: %d.",
					duration,
					int(steps),
				),
			},
		},
	}, nil
}

func handleNotification(
	ctx context.Context,
	notification mcp.JSONRPCNotification,
) {
	log.Printf("Received notification: %s", notification.Method)
}

func main() {
	var transport string
	flag.StringVar(&transport, "t", "stdio", "Transport type (stdio or sse)")
	flag.StringVar(&transport, "transport", "stdio", "Transport type (stdio or sse)")
	flag.Parse()

	mcpServer := NewMCPServer()

	// Only check for "sse" since stdio is the default
	if transport == "sse" {
		sseServer := server.NewSSEServer(mcpServer, server.WithBaseURL("http://localhost:8080"))
		log.Printf("SSE server listening on :8080")
		if err := sseServer.Start(":8080"); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	} else {
		if err := server.ServeStdio(mcpServer); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}
}
