package main

import (
	"context"
	"fmt"
	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"log"
	"os"
	"time"
)

func main() {
	// 启动 MCP 服务端（工具服务）
	startMCPServer()

	// 等待服务端启动完成
	time.Sleep(1 * time.Second)

	// 创建上下文
	ctx := context.Background()

	// 从 MCP 客户端中获取工具
	mcpTools := getMCPTool(ctx)
	arkChatModel, err := newChatModel(ctx)

	messages := createMessagesFromTemplate()
	msg, err := arkChatModel.Generate(ctx, messages)
	log.Print(msg.Content)
	if err != nil {
		log.Fatal("------创建模型时出错----")
	}

	//ragent, err := react.NewAgent(ctx, &react.AgentConfig{
	//	ToolCallingModel: chatModel,
	//	ToolsConfig: compose.ToolsNodeConfig{
	//		Tools: mcpTools,
	//	},
	//})

	// 遍历所有工具，调用其 Info 和 InvokableRun 方法
	for i, mcpTool := range mcpTools {
		fmt.Println(i, ":")
		// 获取工具信息
		info, err := mcpTool.Info(ctx)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("工具名称:", info.Name)
		fmt.Println("工具描述:", info.Desc)

		// 调用工具进行计算操作（示例：1 + 1）
		result, err := mcpTool.(tool.InvokableTool).InvokableRun(ctx, `{"operation":"add", "x":979, "y":786}`)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("执行结果:", result)
		fmt.Println()
	}
}

func newChatModel(ctx context.Context) (*ark.ChatModel, error) {

	openAIBaseURL := os.Getenv("OPENAI_BASE_URL")
	openAIAPIKey := os.Getenv("OPENAI_API_KEY")
	modelName := os.Getenv("OPENAI_MODEL_NAME") // 例如 "gpt-4"

	if openAIBaseURL == "" || openAIAPIKey == "" || modelName == "" {
		log.Println("警告: OPENAI_BASE_URL, OPENAI_API_KEY, 或 OPENAI_MODEL_NAME 环境变量未设置。")
		log.Println("请设置这些变量以运行本示例。")
	}

	//openai.NewChatModel(ctx, &openai.ChatModelConfig{})
	// 2. 创建 OpenAI 聊天模型组件
	chatModel, err := ark.NewChatModel(ctx, &ark.ChatModelConfig{
		BaseURL: openAIBaseURL,
		APIKey:  openAIAPIKey,
		Model:   modelName,
	})

	if err != nil {
		return nil, fmt.Errorf("创建聊天模型失败: %w", err)
	}
	return chatModel, err
}
func createTemplate() prompt.ChatTemplate {
	// 创建模板，使用 FString 格式
	return prompt.FromMessages(schema.FString,
		// 系统消息模板
		schema.SystemMessage("你是一个{role}。你需要用{style}的语气回答问题。你的目标是帮助程序员保持积极乐观的心态，提供技术建议的同时也要关注他们的心理健康。"),

		// 插入需要的对话历史（新对话的话这里不填）
		schema.MessagesPlaceholder("chat_history", true),

		// 用户消息模板
		schema.UserMessage("问题: {question}"),
	)
}

func createMessagesFromTemplate() []*schema.Message {
	template := createTemplate()

	// 使用模板生成消息
	messages, err := template.Format(context.Background(), map[string]any{
		"role":     "程序员鼓励师",
		"style":    "积极、温暖且专业",
		"question": "我的代码一直报错，感觉好沮丧，该怎么办？",
		// 对话历史（这个例子里模拟两轮对话历史）
		"chat_history": []*schema.Message{
			schema.UserMessage("你好"),
			schema.AssistantMessage("嘿！我是你的程序员鼓励师！记住，每个优秀的程序员都是从 Debug 中成长起来的。有什么我可以帮你的吗？", nil),
			schema.UserMessage("我觉得自己写的代码太烂了"),
			schema.AssistantMessage("每个程序员都经历过这个阶段！重要的是你在不断学习和进步。让我们一起看看代码，我相信通过重构和优化，它会变得更好。记住，Rome wasn't built in a day，代码质量是通过持续改进来提升的。", nil),
		},
	})
	if err != nil {
		log.Fatalf("format template failed: %v\n", err)
	}
	return messages
}

// 输出结果
//func main() {
//	messages := createMessagesFromTemplate()
//	fmt.Printf("formatted message: %v", messages)
//}

// formatted message: [system: 你是一个程序员鼓励师。你需要用积极、温暖且专业的语气回答问题。你的目标是帮助程序员保持积极乐观的心态，提供技术建议的同时也要关注他们的心理健康。 user: 你好 assistant: 嘿！我是你的程序员鼓励师！记住，每个优秀的程序员都是从 Debug 中成长起来的。有什么我可以帮你的吗？ user: 我觉得自己写的代码太烂了 assistant: 每个程序员都经历过这个阶段！重要的是你在不断学习和进步。让我们一起看看代码，我相信通过重构和优化，它会变得更好。记住，Rome wasn't built in a day，代码质量是通过持续改进来提升的。 user: 问题: 我的代码一直报错，感觉好沮丧，该怎么办？]
