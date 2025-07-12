package mcp

import (
	"context"
	"log"

	mcpp "github.com/cloudwego/eino-ext/components/tool/mcp"
	"github.com/cloudwego/eino/components/tool"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

const (
	DefaultMCPServerURL = "http://localhost:12345/sse"
)

func GetMCPTools(ctx context.Context, url string) ([]tool.BaseTool, error) {
	log.Println("Creating MCP SSE client to connect to", url)
	cli, err := client.NewSSEMCPClient(url)
	if err != nil {
		return nil, err
	}
	log.Println("Starting client...")
	err = cli.Start(ctx)
	if err != nil {
		return nil, err
	}

	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "eino-react-agent-client",
		Version: "1.0.0",
	}

	log.Println("Initializing client with server...")
	_, err = cli.Initialize(ctx, initRequest)
	if err != nil {
		return nil, err
	}

	log.Println("Getting tools from server...")
	tools, err := mcpp.GetTools(ctx, &mcpp.Config{Cli: cli})
	if err != nil {
		return nil, err
	}

	return tools, nil
} 