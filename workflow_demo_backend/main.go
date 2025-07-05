package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/cloudwego/eino-examples/internal/logs"
	"github.com/cloudwego/eino-examples/workflow_demo_backend/graph"
	"github.com/cloudwego/eino-ext/devops"
)

func main() {
	ctx := context.Background()

	// 初始化 eino devops 服务
	err := devops.Init(ctx)
	if err != nil {
		logs.Errorf("[eino dev] 初始化失败, err=%v", err)
		return
	}

	// 编译工作流图
	workflow, err := graph.NewConditionalRewriterGraphStream(ctx)
	if err != nil {
		log.Fatalf("编译图失败, err: %v", err)
	}

	// 从控制台读取输入并持续对话
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("\n请输入您的问题 (输入 'exit' 来结束对话): ")
		userInput, _ := reader.ReadString('\n')
		userInput = strings.TrimSpace(userInput)

		if userInput == "exit" {
			fmt.Println("再见！")
			break
		}

		if userInput == "" {
			continue
		}

		// 使用输入内容以流式方式调用工作流
		stream, err := workflow.Stream(ctx, userInput)
		if err != nil {
			logs.Errorf("调用图失败, err: %v", err)
			continue
		}

		// 处理流式输出
		fmt.Print("AI: ")
		for {
			chunk, err := stream.Recv()
			if err != nil {
				if err == io.EOF {
					break // 流结束
				}
				logs.Errorf("从流中接收数据时出错, err: %v", err)
				break
			}
			fmt.Print(chunk)
		}
		fmt.Println() // 在每次输出后换行
	}
}
