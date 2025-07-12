package main

import (
	"context"
	mcpp "github.com/cloudwego/eino-ext/components/tool/mcp"
	"github.com/cloudwego/eino/components/tool"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"log"
)

// 创建 MCP 客户端并初始化，获取可用的工具列表
func getMCPTool(ctx context.Context) []tool.BaseTool {
	// 使用 SSE 协议连接到 MCP 服务端
	cli, err := client.NewSSEMCPClient("http://localhost:12345/sse")
	if err != nil {
		log.Fatal(err)
	}

	// 启动客户端
	err = cli.Start(ctx)
	if err != nil {
		log.Fatal(err)
	}

	// 初始化客户端信息
	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "example-client",
		Version: "1.0.0",
	}

	// 向 MCP 服务端发送初始化请求
	_, err = cli.Initialize(ctx, initRequest)
	if err != nil {
		log.Fatal(err)
	}

	// 从客户端获取工具列表并转为 Eino 可识别格式
	tools, err := mcpp.GetTools(ctx, &mcpp.Config{Cli: cli})
	if err != nil {
		log.Fatal(err)
	}

	return tools
}
