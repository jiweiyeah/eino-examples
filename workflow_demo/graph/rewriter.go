package graph

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/joho/godotenv"
)

// RewriteState 定义了我们工作流图的状态。
type RewriteState struct {
	OriginalQuery string
	Decision      string // "valid" (有效) or "invalid" (无效)
}

// NewConditionalRewriterGraph 创建并编译一个条件重写工作流图。
func NewConditionalRewriterGraph(ctx context.Context) (compose.Runnable[string, string], error) {
	// 加载 .env 文件
	err := godotenv.Load("../.env")
	if err != nil {
		// 允许在没有 .env 文件的情况下运行
	}
	// 1. 从环境变量中获取 OpenAI 的凭证
	openAIBaseURL := os.Getenv("OPENAI_BASE_URL")
	openAIAPIKey := os.Getenv("OPENAI_API_KEY")
	modelName := os.Getenv("OPENAI_MODEL_NAME") // 例如 "gpt-4"

	if openAIBaseURL == "" || openAIAPIKey == "" || modelName == "" {
		log.Println("警告: OPENAI_BASE_URL, OPENAI_API_KEY, 或 OPENAI_MODEL_NAME 环境变量未设置。")
		log.Println("请设置这些变量以运行本示例。")
	}

	// 2. 创建 OpenAI 聊天模型组件
	chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		BaseURL: openAIBaseURL,
		APIKey:  openAIAPIKey,
		Model:   modelName,
	})
	if err != nil {
		return nil, fmt.Errorf("创建聊天模型失败: %w", err)
	}

	// 创建一个有状态的图
	sg := compose.NewGraph[string, string](compose.WithGenLocalState(func(ctx context.Context) *RewriteState {
		return &RewriteState{}
	}))

	// 定义图的节点

	// 节点：在状态中存储初始查询
	_ = sg.AddLambdaNode("start_node", compose.InvokableLambda(func(ctx context.Context, input string) (string, error) {
		return input, nil
	}), compose.WithStatePostHandler(func(ctx context.Context, out string, state *RewriteState) (string, error) {
		state.OriginalQuery = out
		return out, nil
	}), compose.WithNodeName("存储初始查询"))

	// 节点：为分类器提示准备输入
	_ = sg.AddLambdaNode("prepare_classifier_input", compose.InvokableLambda(func(ctx context.Context, input string) (map[string]any, error) {
		return map[string]any{"input": input}, nil
	}), compose.WithNodeName("准备分类器输入"))

	// 分类器节点
	classifierPrompt := prompt.FromMessages(
		schema.FString,
		schema.SystemMessage("你是一个查询分类器。判断用户的输入是一个有效问题还是无效的垃圾信息/辱骂。请只回答 'valid' 或 'invalid'。"),
		schema.UserMessage("用户输入: {input}"),
	)
	_ = sg.AddChatTemplateNode("classifier_prompt", classifierPrompt, compose.WithNodeName("分类器提示词"))
	_ = sg.AddChatModelNode("classifier_model", chatModel, compose.WithNodeName("分类器模型"))

	// 节点：用分类器的决定更新状态
	_ = sg.AddLambdaNode("update_state_with_decision", compose.InvokableLambda(func(ctx context.Context, msg *schema.Message) (*schema.Message, error) {
		return msg, nil
	}), compose.WithStatePostHandler(func(ctx context.Context, msg *schema.Message, state *RewriteState) (*schema.Message, error) {
		decision := strings.TrimSpace(strings.ToLower(msg.Content))
		if decision == "valid" || decision == "invalid" {
			state.Decision = decision
		} else {
			state.Decision = "invalid" // 默认值
		}
		fmt.Printf("分类器判定: %s\n", state.Decision)
		return msg, nil
	}), compose.WithNodeName("更新状态-分类决定"))

	// 节点：为重写器提示准备输入 (有效分支)
	_ = sg.AddLambdaNode("prepare_rewriter_input", compose.InvokableLambda(func(ctx context.Context, _ *schema.Message) (map[string]any, error) {
		var query string
		err := compose.ProcessState(ctx, func(ctx context.Context, state *RewriteState) error {
			query = state.OriginalQuery
			return nil
		})
		if err != nil {
			return nil, err
		}
		return map[string]any{"input": query}, nil
	}), compose.WithNodeName("准备重写器输入"))

	// 重写器节点 (有效分支)
	rewriterPrompt := prompt.FromMessages(
		schema.FString,
		schema.SystemMessage("你是一位专业的查询重写专家。请将用户的问题改写得更清晰、更适合搜索引擎。"),
		schema.UserMessage("用户问题: {input}"),
	)
	_ = sg.AddChatTemplateNode("rewriter_prompt", rewriterPrompt, compose.WithNodeName("重写器提示词"))
	_ = sg.AddChatModelNode("rewriter_model", chatModel, compose.WithNodeName("重写器模型"))
	_ = sg.AddLambdaNode("get_rewritten_output", compose.InvokableLambda(func(ctx context.Context, msg *schema.Message) (string, error) {
		return msg.Content, nil
	}), compose.WithNodeName("获取重写结果"))

	// 直通节点 (无效分支)
	_ = sg.AddLambdaNode("passthrough_node", compose.InvokableLambda(func(ctx context.Context, _ *schema.Message) (string, error) {
		var query string
		err := compose.ProcessState(ctx, func(ctx context.Context, state *RewriteState) error {
			query = state.OriginalQuery
			return nil
		})
		return query, err
	}), compose.WithNodeName("无效问题直通"))

	// 定义图的结构 (边)
	_ = sg.AddEdge(compose.START, "start_node")
	_ = sg.AddEdge("start_node", "prepare_classifier_input")
	_ = sg.AddEdge("prepare_classifier_input", "classifier_prompt")
	_ = sg.AddEdge("classifier_prompt", "classifier_model")
	_ = sg.AddEdge("classifier_model", "update_state_with_decision")

	// 条件分支
	_ = sg.AddBranch("update_state_with_decision", compose.NewGraphBranch(
		func(ctx context.Context, input *schema.Message) (string, error) {
			var decision string
			err := compose.ProcessState(ctx, func(ctx context.Context, state *RewriteState) error {
				decision = state.Decision
				return nil
			})
			if err != nil {
				return "", err
			}
			if decision == "valid" {
				return "prepare_rewriter_input", nil
			}
			return "passthrough_node", nil
		},
		map[string]bool{
			"prepare_rewriter_input": true,
			"passthrough_node":       true,
		},
	))

	// 有效分支的边
	_ = sg.AddEdge("prepare_rewriter_input", "rewriter_prompt")
	_ = sg.AddEdge("rewriter_prompt", "rewriter_model")
	_ = sg.AddEdge("rewriter_model", "get_rewritten_output")
	_ = sg.AddEdge("get_rewritten_output", compose.END)

	// 无效分支的边
	_ = sg.AddEdge("passthrough_node", compose.END)

	// 编译并返回图
	return sg.Compile(ctx)
}

// NewConditionalRewriterGraphStream 创建并编译一个支持流式输出的条件重写工作流图。
func NewConditionalRewriterGraphStream(ctx context.Context) (compose.Runnable[string, string], error) {
	// 加载 .env 文件
	err := godotenv.Load("../.env")
	if err != nil {
		// 允许在没有 .env 文件的情况下运行
	}
	// 1. 从环境变量中获取 OpenAI 的凭证
	openAIBaseURL := os.Getenv("OPENAI_BASE_URL")
	openAIAPIKey := os.Getenv("OPENAI_API_KEY")
	modelName := os.Getenv("OPENAI_MODEL_NAME") // 例如 "gpt-4"

	if openAIBaseURL == "" || openAIAPIKey == "" || modelName == "" {
		log.Println("警告: OPENAI_BASE_URL, OPENAI_API_KEY, 或 OPENAI_MODEL_NAME 环境变量未设置。")
		log.Println("请设置这些变量以运行本示例。")
	}

	// 2. 创建 OpenAI 聊天模型组件
	chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		BaseURL: openAIBaseURL,
		APIKey:  openAIAPIKey,
		Model:   modelName,
	})
	if err != nil {
		return nil, fmt.Errorf("创建聊天模型失败: %w", err)
	}

	// 创建一个有状态的图，最终输出为字符串流
	sg := compose.NewGraph[string, string](compose.WithGenLocalState(func(ctx context.Context) *RewriteState {
		return &RewriteState{}
	}))

	// 定义图的节点

	// 节点：在状态中存储初始查询
	_ = sg.AddLambdaNode("start_node", compose.StreamableLambda(func(ctx context.Context, input string) (*schema.StreamReader[string], error) {
		sr, sw := schema.Pipe[string](1)
		sw.Send(input, nil)
		sw.Close()
		return sr, nil
	}), compose.WithStatePostHandler(func(ctx context.Context, out string, state *RewriteState) (string, error) {
		state.OriginalQuery = out
		return out, nil
	}), compose.WithNodeName("存储初始查询(流式)"))

	// 节点：为分类器提示准备输入
	_ = sg.AddLambdaNode("prepare_classifier_input", compose.StreamableLambda(func(ctx context.Context, input string) (*schema.StreamReader[map[string]any], error) {
		sr, sw := schema.Pipe[map[string]any](1)
		sw.Send(map[string]any{"input": input}, nil)
		sw.Close()
		return sr, nil
	}), compose.WithNodeName("准备分类器输入(流式)"))

	// 分类器节点
	classifierPrompt := prompt.FromMessages(
		schema.FString,
		schema.SystemMessage("你是一个查询分类器。判断用户的输入是一个有效问题还是无效的垃圾信息/辱骂。请只回答 'valid' 或 'invalid'。"),
		schema.UserMessage("用户输入: {input}"),
	)
	_ = sg.AddChatTemplateNode("classifier_prompt", classifierPrompt, compose.WithNodeName("分类器提示词"))
	// 因为AI模型只支持流式，使用StreamableLambda。
	// 这个lambda调用流式端点并聚合结果为单个消息。
	_ = sg.AddLambdaNode("classifier_model_stream", compose.StreamableLambda(
		func(ctx context.Context, messages []*schema.Message) (*schema.StreamReader[*schema.Message], error) {
			stream, err := chatModel.Stream(ctx, messages)
			if err != nil {
				return nil, fmt.Errorf("streaming chat model for classifier failed: %w", err)
			}

			// 将流式响应聚合成单个字符串。
			var fullContent strings.Builder
			for {
				chunk, err := stream.Recv()
				if err != nil {
					if err == io.EOF {
						break
					}
					return nil, err // 传播其他错误。
				}
				fullContent.WriteString(chunk.Content)
			}

			// 从聚合的内容创建单个消息。
			finalMessage := &schema.Message{
				Role:    schema.Assistant,
				Content: fullContent.String(),
			}

			// 通过流管道发送单个聚合消息。
			sr, sw := schema.Pipe[*schema.Message](1)
			sw.Send(finalMessage, nil)
			sw.Close()
			return sr, nil
		},
	), compose.WithNodeName("分类器模型(流式)"))

	// 节点：用分类器的决定更新状态
	_ = sg.AddLambdaNode("update_state_with_decision", compose.StreamableLambda(func(ctx context.Context, msg *schema.Message) (*schema.StreamReader[*schema.Message], error) {
		sr, sw := schema.Pipe[*schema.Message](1)
		sw.Send(msg, nil)
		sw.Close()
		return sr, nil
	}), compose.WithStatePostHandler(func(ctx context.Context, msg *schema.Message, state *RewriteState) (*schema.Message, error) {
		decision := strings.TrimSpace(strings.ToLower(msg.Content))
		if decision == "valid" || decision == "invalid" {
			state.Decision = decision
		} else {
			state.Decision = "invalid" // 默认值
		}
		fmt.Printf("分类器判定: %s\n", state.Decision)
		return msg, nil
	}), compose.WithNodeName("更新状态-分类决定(流式)"))

	// 节点：为重写器提示准备输入 (有效分支)
	_ = sg.AddLambdaNode("prepare_rewriter_input", compose.StreamableLambda(func(ctx context.Context, _ *schema.Message) (*schema.StreamReader[map[string]any], error) {
		var query string
		err := compose.ProcessState(ctx, func(ctx context.Context, state *RewriteState) error {
			query = state.OriginalQuery
			return nil
		})
		if err != nil {
			return nil, err
		}
		sr, sw := schema.Pipe[map[string]any](1)
		sw.Send(map[string]any{"input": query}, nil)
		sw.Close()
		return sr, nil
	}), compose.WithNodeName("准备重写器输入(流式)"))

	// 重写器节点 (有效分支)
	rewriterPrompt := prompt.FromMessages(
		schema.FString,
		schema.SystemMessage("你是一位专业的查询重写专家。请将用户的问题改写得更清晰、更适合搜索引擎。"),
		schema.UserMessage("用户问题: {input}"),
	)
	_ = sg.AddChatTemplateNode("rewriter_prompt", rewriterPrompt, compose.WithNodeName("重写器提示词"))
	_ = sg.AddLambdaNode("rewriter_model_stream", compose.StreamableLambda(
		func(ctx context.Context, messages []*schema.Message) (*schema.StreamReader[string], error) {
			stream, err := chatModel.Stream(ctx, messages)
			if err != nil {
				return nil, fmt.Errorf("streaming chat model failed: %w", err)
			}

			sr, sw := schema.Pipe[string](0)

			go func() {
				defer sw.Close()
				for {
					chunk, err := stream.Recv()
					if err != nil {
						if err != io.EOF {
							// Propagate error to the stream
							sw.Send("", err)
						}
						return
					}
					sw.Send(chunk.Content, nil)
				}
			}()
			return sr, nil
		},
	), compose.WithNodeName("重写器模型(流式)"))

	// 直通节点 (无效分支)
	_ = sg.AddLambdaNode("passthrough_node", compose.StreamableLambda(func(ctx context.Context, _ *schema.Message) (*schema.StreamReader[string], error) {
		var query string
		err := compose.ProcessState(ctx, func(ctx context.Context, state *RewriteState) error {
			query = state.OriginalQuery
			return nil
		})
		if err != nil {
			return nil, err
		}
		sr, sw := schema.Pipe[string](1)
		sw.Send(query, nil)
		sw.Close()
		return sr, nil
	}), compose.WithNodeName("无效问题直通(流式)"))

	// 定义图的结构 (边)
	_ = sg.AddEdge(compose.START, "start_node")
	_ = sg.AddEdge("start_node", "prepare_classifier_input")
	_ = sg.AddEdge("prepare_classifier_input", "classifier_prompt")
	_ = sg.AddEdge("classifier_prompt", "classifier_model_stream")
	_ = sg.AddEdge("classifier_model_stream", "update_state_with_decision")

	// 条件分支
	_ = sg.AddBranch("update_state_with_decision", compose.NewGraphBranch(
		func(ctx context.Context, input *schema.Message) (string, error) {
			var decision string
			err := compose.ProcessState(ctx, func(ctx context.Context, state *RewriteState) error {
				decision = state.Decision
				return nil
			})
			if err != nil {
				return "", err
			}
			if decision == "valid" {
				return "prepare_rewriter_input", nil
			}
			return "passthrough_node", nil
		},
		map[string]bool{
			"prepare_rewriter_input": true,
			"passthrough_node":       true,
		},
	))

	// 有效分支的边
	_ = sg.AddEdge("prepare_rewriter_input", "rewriter_prompt")
	_ = sg.AddEdge("rewriter_prompt", "rewriter_model_stream")
	_ = sg.AddEdge("rewriter_model_stream", compose.END)

	// 无效分支的边
	_ = sg.AddEdge("passthrough_node", compose.END)

	// 编译并返回图
	return sg.Compile(ctx)
}