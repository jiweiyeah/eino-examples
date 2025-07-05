package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/cloudwego/eino-examples/internal/logs"
	"github.com/cloudwego/eino-examples/workflow_demo/graph"
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
	workflow, err := graph.NewConditionalRewriterGraph(ctx)
	if err != nil {
		log.Fatalf("编译图失败, err: %v", err)
	}

	// 从控制台读取输入并持续对话
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("\n请输入您的问题 (输入 'exit' 来结束): ")
		userInput, _ := reader.ReadString('\n')
		userInput = strings.TrimSpace(userInput)

		if userInput == "exit" {
			fmt.Println("再见！")
			break
		}

		if userInput == "" {
			continue
		}

		// 使用输入内容调用工作流
		res, err := workflow.Invoke(ctx, userInput)
		if err != nil {
			logs.Errorf("调用图失败, err: %v", err)
			return
		}
		logs.Infof("工作流输出: %s", res)
	}
}
