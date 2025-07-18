package main

import (
	"context"
	"fmt"

	_ "github.com/cloudwego/eino-ext/components/tool/mcp"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"log"
)

// 启动 MCP 服务端并注册工具
func startMCPServer() {
	// 创建 MCP 服务端，声明支持的协议版本
	svr := server.NewMCPServer("demo", mcp.LATEST_PROTOCOL_VERSION)

	// 添加一个名为 "calculate" 的工具，支持基本运算
	svr.AddTool(
		// 定义工具结构和参数
		mcp.NewTool("calculate",
			mcp.WithDescription("执行基础运算操作"),
			mcp.WithString("operation",
				mcp.Required(),
				mcp.Description("操作类型（加法、减法、乘法、除法）"),
				mcp.Enum("add", "subtract", "multiply", "divide"),
			),
			mcp.WithNumber("x",
				mcp.Required(),
				mcp.Description("第一个数字"),
			),
			mcp.WithNumber("y",
				mcp.Required(),
				mcp.Description("第二个数字"),
			),
		),
		// 工具的实际执行逻辑
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			arg := request.Params.Arguments.(map[string]any)
			op := arg["operation"].(string)
			x := arg["x"].(float64)
			y := arg["y"].(float64)

			var result float64
			switch op {
			case "add":
				result = x + y
			case "subtract":
				result = x - y
			case "multiply":
				result = x * y
			case "divide":
				if y == 0 {
					return mcp.NewToolResultText("除数不能为零"), nil
				}
				result = x / y
			}
			log.Printf("计算结果哈哈哈哈: %.2f", result)
			return mcp.NewToolResultText(fmt.Sprintf("%.2f", result)), nil
		},
	)

	// 添加一个名为 "echo" 的工具
	svr.AddTool(
		mcp.NewTool("echo",
			mcp.WithDescription("返回输入的字符串"),
			mcp.WithString("text",
				mcp.Required(),
				mcp.Description("要返回的字符串"),
			),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			arg := request.Params.Arguments.(map[string]any)
			text := arg["text"].(string)
			log.Printf("Echo 工具收到: %s", text)
			return mcp.NewToolResultText(text), nil
		},
	)

	// 添加一个名为 "reverse_string" 的工具
	svr.AddTool(
		mcp.NewTool("reverse_string",
			mcp.WithDescription("反转输入的字符串"),
			mcp.WithString("input_string",
				mcp.Required(),
				mcp.Description("要反转的字符串"),
			),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			arg := request.Params.Arguments.(map[string]any)
			inputString := arg["input_string"].(string)

			runes := []rune(inputString)
			for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
				runes[i], runes[j] = runes[j], runes[i]
			}
			reversedString := string(runes)

			log.Printf("反转字符串工具收到: %s, 返回: %s", inputString, reversedString)
			return mcp.NewToolResultText(reversedString), nil
		},
	)

	// 异步启动 HTTP SSE 服务
	go func() {
		defer func() {
			e := recover()
			if e != nil {
				fmt.Println("服务异常:", e)
			}
		}()

		// 启动服务监听在本地 12345 端口
		err := server.NewSSEServer(svr, server.WithBaseURL("http://localhost:12345")).Start("localhost:12345")
		if err != nil {
			log.Fatal(err)
		}
	}()
}
