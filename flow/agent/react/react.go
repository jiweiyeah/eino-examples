/*
 * Copyright 2024 CloudWeGo Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	mcpclient "github.com/cloudwego/eino-examples/flow/agent/react/mcp"
	"github.com/cloudwego/eino-examples/internal/logs"
	"github.com/cloudwego/eino/callbacks"
)

func main() {
	// a simple mcp server for demo
	// you can run the server from mcp_demo/examples/mcp.go
	startMCPServer()
	time.Sleep(1 * time.Second)

	ctx := context.Background()
	// arkModel, err := ark.NewChatModel(ctx, &ark.ChatModelConfig{
	// 	APIKey: "f3772123-e155-498e-baf8-1eac959ae392",
	// 	Model:  "deepseek-v3-250324",
	// })
	arkModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		APIKey: "sk-rhlzvcpnvpbrlsvsggqbjwyosibwvqwxbotgfrbtzkeybfdr",
		Model:  "Qwen/Qwen3-8B",
		BaseURL: "https://api.siliconflow.cn/v1",
	})
	if err != nil {
		logs.Errorf("failed to create chat model: %v", err)
		return
	}

	// prepare tools
	allTools := []tool.BaseTool{}

	// get mcp tools
	mcpTools, err := mcpclient.GetMCPTools(ctx, mcpclient.DefaultMCPServerURL)
	if err != nil {
		logs.Infof("failed to get mcp tools: %v", err)
	} else {
		allTools = append(allTools, mcpTools...)
	}

	// prepare persona (system prompt) (optional)
	persona := `You are a helpful assistant.`

	// replace tool call checker with a custom one: check all trunks until you get a tool call
	// because some models(claude or doubao 1.5-pro 32k) do not return tool call in the first response
	// uncomment the following code to enable it
	toolCallChecker := func(ctx context.Context, sr *schema.StreamReader[*schema.Message]) (bool, error) {
		defer sr.Close()
		for {
			msg, err := sr.Recv()
			if err != nil {
				if errors.Is(err, io.EOF) {
					// finish
					break
				}

				return false, err
			}

			if len(msg.ToolCalls) > 0 {
				return true, nil
			}
		}
		return false, nil
	}

	logger, err := NewLoggerCallback("react-agent.log")
	if err != nil {
		logs.Fatalf("failed to create logger callback: %v", err)
	}
	defer logger.Close()

	ragent, err := react.NewAgent(ctx, &react.AgentConfig{
		Model: arkModel,
		ToolsConfig: compose.ToolsNodeConfig{
			Tools: allTools,
		},
		StreamToolCallChecker: toolCallChecker, // uncomment it to replace the default tool call checker with custom one
	})
	if err != nil {
		logs.Fatalf("failed to create agent, err: %v", err)
	}

	// if you want ping/pong, use Generate
	// msg, err := agent.Generate(ctx, []*schema.Message{
	// 	{
	// 		Role:    schema.User,
	// 		Content: "我在北京，给我推荐一些菜，需要有口味辣一点的菜，至少推荐有 2 家餐厅",
	// 	},
	// }, react.WithCallbacks(&myCallback{}))
	// if err != nil {
	// 	log.Printf("failed to generate: %v\n", err)
	// 	return
	// }
	// fmt.Println(msg.String())

	// run agent
	sr, err := ragent.Stream(ctx, []*schema.Message{
		{
			Role:    schema.System,
			Content: persona,
		},
		{
			Role:    schema.User,
			Content: "what is 123 + 456?",
		},
	}, agent.WithComposeOptions(compose.WithCallbacks(logger)))
	if err != nil {
		logs.Errorf("failed to stream: %v", err)
		return
	}

	defer sr.Close() // remember to close the stream

	logs.Infof("\n\n===== start streaming =====\n\n")

	for {
		msg, err := sr.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				// finish
				break
			}
			// error
			logs.Infof("failed to recv: %v", err)
			return
		}

		// 打字机打印
		fmt.Print(msg.Content)
	}

	logs.Infof("\n\n===== finished =====\n")

}

func startMCPServer() {
	svr := server.NewMCPServer("demo", mcp.LATEST_PROTOCOL_VERSION)
	svr.AddTool(mcp.NewTool("calculate",
		mcp.WithDescription("Perform basic arithmetic operations"),
		mcp.WithString("operation",
			mcp.Required(),
			mcp.Description("The operation to perform (add, subtract, multiply, divide)"),
			mcp.Enum("add", "subtract", "multiply", "divide"),
		),
		mcp.WithNumber("x",
			mcp.Required(),
			mcp.Description("First number"),
		),
		mcp.WithNumber("y",
			mcp.Required(),
			mcp.Description("Second number"),
		),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
				return mcp.NewToolResultText("Cannot divide by zero"), nil
			}
			result = x / y
		}
		log.Printf("Calculated result: %.2f", result)
		return mcp.NewToolResultText(fmt.Sprintf("%.2f", result)), nil
	})
	go func() {
		defer func() {
			e := recover()
			if e != nil {
				fmt.Println(e)
			}
		}()
		log.Println("--- Server Side ---")
		log.Println("Starting MCP SSE server at localhost:12345")
		err := server.NewSSEServer(svr, server.WithBaseURL("http://localhost:12345")).Start("localhost:12345")

		if err != nil {
			log.Fatal(err)
		}
	}()
}

type LoggerCallback struct {
	callbacks.HandlerBuilder // 可以用 callbacks.HandlerBuilder 来辅助实现 callback
	file                     *os.File
}

func NewLoggerCallback(filename string) (*LoggerCallback, error) {
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		return nil, err
	}
	return &LoggerCallback{file: f}, nil
}

func (cb *LoggerCallback) Close() {
	cb.file.Close()
}

func (cb *LoggerCallback) OnStart(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
	fmt.Fprintf(cb.file, "==================\n")
	inputStr, _ := json.MarshalIndent(input, "", "  ") // nolint: byted_s_returned_err_check
	fmt.Fprintf(cb.file, "[OnStart] %s\n", string(inputStr))
	return ctx
}

func (cb *LoggerCallback) OnEnd(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
	fmt.Fprintf(cb.file, "=========[OnEnd]=========\n")
	outputStr, _ := json.MarshalIndent(output, "", "  ") // nolint: byted_s_returned_err_check
	fmt.Fprintf(cb.file, "%s\n", string(outputStr))
	return ctx
}

func (cb *LoggerCallback) OnError(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
	fmt.Fprintf(cb.file, "=========[OnError]=========\n")
	fmt.Fprintf(cb.file, "%v\n", err)
	return ctx
}

func (cb *LoggerCallback) OnToolStart(ctx context.Context, input string) {
	fmt.Fprintf(cb.file, "=========[OnToolStart]=========\n")
	fmt.Fprintf(cb.file, "Input: %s\n", input)
}

func (cb *LoggerCallback) OnToolEnd(ctx context.Context, output string) {
	fmt.Fprintf(cb.file, "=========[OnToolEnd]=========\n")
	fmt.Fprintf(cb.file, "Output: %s\n", output)
}

func (cb *LoggerCallback) OnToolError(ctx context.Context, err error) {
	fmt.Fprintf(cb.file, "=========[OnToolError]=========\n")
	fmt.Fprintf(cb.file, "Error: %v\n", err)
}

func (cb *LoggerCallback) OnEndWithStreamOutput(ctx context.Context, info *callbacks.RunInfo,
	output *schema.StreamReader[callbacks.CallbackOutput]) context.Context {

	var graphInfoName = react.GraphName

	go func() {
		defer func() {
			if err := recover(); err != nil {
				fmt.Println("[OnEndStream] panic err:", err)
			}
		}()

		defer output.Close() // remember to close the stream in defer

		fmt.Fprintf(cb.file, "=========[OnEndStream]=========\n")
		for {
			frame, err := output.Recv()
			if errors.Is(err, io.EOF) {
				// finish
				break
			}
			if err != nil {
				fmt.Fprintf(cb.file, "internal error: %s\n", err)
				return
			}

			s, err := json.Marshal(frame)
			if err != nil {
				fmt.Fprintf(cb.file, "internal error: %s\n", err)
				return
			}

			if info.Name == graphInfoName { // 仅打印 graph 的输出, 否则每个 stream 节点的输出都会打印一遍
				fmt.Fprintf(cb.file, "%s: %s\n", info.Name, string(s))
			}
		}

	}()
	return ctx
}

func (cb *LoggerCallback) OnStartWithStreamInput(ctx context.Context, info *callbacks.RunInfo,
	input *schema.StreamReader[callbacks.CallbackInput]) context.Context {
	defer input.Close()
	return ctx
}

