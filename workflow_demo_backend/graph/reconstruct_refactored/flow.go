package reconstruct_refactored

import (
	"context"
	"fmt"
	"log"
	"os"
	"reflect"
	"strings"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/joho/godotenv"

	"workflow_demo_backend/graph/reconstruct_refactored/prompts"
	"workflow_demo_backend/graph/reconstruct_refactored/types"
)

func NewReconstructGraphRefactored(ctx context.Context) (compose.Runnable[string, string], error) {
	// 加载 .env 文件
	// 注意: 相对路径可能需要根据实际执行位置调整。如果 .env 在项目根目录，可能需要修改为 "../../../.env"
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
	sg := compose.NewGraph[string, string](compose.WithGenLocalState(func(ctx context.Context) *types.RewriteState {
		return &types.RewriteState{}
	}))

	// 定义图的节点

	// 节点：为分类器提示准备输入和提示词
	_ = sg.AddLambdaNode("prepare_classifier_input_and_prompt", compose.InvokableLambda(func(ctx context.Context, input string) ([]*schema.Message, error) {
		// 这个部分合并了 prepare_classifier_input 和 classifier_prompt 的逻辑
		inputMap := map[string]any{"input": input}
		messages, err := prompts.GetClassifierPrompt().Format(ctx, inputMap)
		if err != nil {
			return nil, fmt.Errorf("格式化分类器提示词失败: %w", err)
		}
		return messages, nil
	}), compose.WithStatePreHandler(func(ctx context.Context, out string, state *types.RewriteState) (string, error) {
		state.OriginalQuery = out
		return out, nil
	}), compose.WithNodeName("准备分类器输入和提示词"))

	// 这个lambda调用流式端点并聚合结果为单个消息。
	_ = sg.AddChatModelNode("classifier_model_stream", chatModel, compose.WithNodeName("分类器模型(流式)"), compose.WithStatePostHandler(
		func(ctx context.Context, out *schema.Message, state *types.RewriteState) (*schema.Message, error) {
			decision := strings.TrimSpace(strings.ToLower(out.Content))
			if decision == "valid" || decision == "invalid" {
				state.Decision = decision
			} else {
				state.Decision = "invalid"
			}
			fmt.Printf("分类器判定: %s
", state.Decision)
			return out, nil
		},
	))

	// 节点：为重写器提示准备输入 (有效分支)
	_ = sg.AddLambdaNode("prepare_rewriter_input", compose.InvokableLambda(func(ctx context.Context, _ *schema.Message) (map[string]any, error) {
		var query string
		err := compose.ProcessState(ctx, func(ctx context.Context, state *types.RewriteState) error {
			query = state.OriginalQuery
			return nil
		})
		if err != nil {
			return nil, err
		}
		return map[string]any{"input": query}, nil
	}), compose.WithNodeName("准备重写器输入"))

	_ = sg.AddChatTemplateNode("rewriter_prompt", prompts.GetRewriterPrompt(), compose.WithNodeName("重写器提示词"))

	_ = sg.AddChatModelNode("rewriter_model_stream", chatModel, compose.WithNodeName("重写器模型(流式)"),
		compose.WithStatePostHandler(
			func(ctx context.Context, out *schema.Message, state *types.RewriteState) (*schema.Message, error) {
				state.RewrittenQuery = out.Content
				return out, nil
			}))

	_ = sg.AddLambdaNode("prepare_intent_classifier_input", compose.InvokableLambda(
		func(ctx context.Context, _ *schema.Message) (map[string]any, error) {
			var rewrittenQuery string
			err := compose.ProcessState(ctx, func(ctx context.Context, state *types.RewriteState) error {
				rewrittenQuery = state.RewrittenQuery
				return nil
			})
			if err != nil {
				return nil, err
			}
			return map[string]any{"input": rewrittenQuery}, nil
		}), compose.WithNodeName("准备意图分类输入"))

	_ = sg.AddChatTemplateNode("intent_classifier_prompt", prompts.GetIntentClassifierPrompt(), compose.WithNodeName("意图分类器提示词"))

	_ = sg.AddChatModelNode("intent_classifier_model_stream", chatModel, compose.WithNodeName("意图分类器模型(流式)"), compose.WithStatePostHandler(
		func(ctx context.Context, out *schema.Message, state *types.RewriteState) (*schema.Message, error) {
			state.Intent = strings.TrimSpace(out.Content)
			return out, nil
		},
	))

	// --- Student Rules Branch ---
	_ = sg.AddLambdaNode("prepare_student_rules_input", compose.InvokableLambda(
		func(ctx context.Context, _ *schema.Message) (map[string]any, error) {
			var rewrittenQuery string
			err := compose.ProcessState(ctx, func(ctx context.Context, state *types.RewriteState) error {
				rewrittenQuery = state.RewrittenQuery
				return nil
			})
			if err != nil {
				return nil, err
			}
			return map[string]any{"input": rewrittenQuery}, nil
		}), compose.WithNodeName("准备学生守则输入"))

	_ = sg.AddChatTemplateNode("student_rules_prompt", prompts.GetStudentRulesPrompt(), compose.WithNodeName("学生守则提示词"))

	_ = sg.AddChatModelNode("student_rules_model_stream", chatModel, compose.WithNodeName("学生守则模型(流式)"))

	_ = sg.AddLambdaNode("output_student_rules", compose.InvokableLambda(func(ctx context.Context, msg *schema.Message) (string, error) {
		return msg.Content, nil
	}), compose.WithNodeName("输出学生守则答案"))

	// --- Employee Rules Branch ---
	_ = sg.AddLambdaNode("prepare_employee_rules_input", compose.InvokableLambda(
		func(ctx context.Context, _ *schema.Message) (map[string]any, error) {
			var rewrittenQuery string
			err := compose.ProcessState(ctx, func(ctx context.Context, state *types.RewriteState) error {
				rewrittenQuery = state.RewrittenQuery
				return nil
			})
			if err != nil {
				return nil, err
			}
			return map[string]any{"input": rewrittenQuery}, nil
		}), compose.WithNodeName("准备员工规范输入"))

	_ = sg.AddChatTemplateNode("employee_rules_prompt", prompts.GetEmployeeRulesFrompt(), compose.WithNodeName("员工规范提示词"))

	_ = sg.AddChatModelNode("employee_rules_model_stream", chatModel, compose.WithNodeName("员工规范模型(流式)"))

	_ = sg.AddLambdaNode("output_employee_rules", compose.InvokableLambda(func(ctx context.Context, msg *schema.Message) (string, error) {
		info := msg.Content
		log.Print("info是", info)
		log.Print("************info类型是：", reflect.TypeOf(info))
		log.Println("
")
		return info, nil
	}), compose.WithNodeName("输出员工规范答案"))

	// --- Other Scenario Branch ---
	_ = sg.AddLambdaNode("other_scenario_output", compose.InvokableLambda(
		func(ctx context.Context, _ *schema.Message) (string, error) {
			return "意图识别场景3:其他类场景", nil
		}), compose.WithNodeName("其他场景输出"))

	// 直通节点 (无效分支)
	_ = sg.AddLambdaNode("passthrough_node", compose.InvokableLambda(func(ctx context.Context, _ *schema.Message) (string, error) {
		var query string
		err := compose.ProcessState(ctx, func(ctx context.Context, state *types.RewriteState) error {
			query = state.OriginalQuery
			return nil
		})
		if err != nil {
			return "", err
		}
		return query, nil
	}), compose.WithNodeName("无效问题直通"))

	// 定义图的结构 (边)
	_ = sg.AddEdge(compose.START, "prepare_classifier_input_and_prompt")
	_ = sg.AddEdge("prepare_classifier_input_and_prompt", "classifier_model_stream")

	// 条件分支
	_ = sg.AddBranch("classifier_model_stream", compose.NewGraphBranch(
		func(ctx context.Context, input *schema.Message) (string, error) {
			var decision string
			err := compose.ProcessState(ctx, func(ctx context.Context, state *types.RewriteState) error {
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
	_ = sg.AddEdge("rewriter_model_stream", "prepare_intent_classifier_input")
	_ = sg.AddEdge("prepare_intent_classifier_input", "intent_classifier_prompt")
	_ = sg.AddEdge("intent_classifier_prompt", "intent_classifier_model_stream")

	// 意图条件分支
	_ = sg.AddBranch("intent_classifier_model_stream", compose.NewGraphBranch(
		func(ctx context.Context, input *schema.Message) (string, error) {
			var intent string
			err := compose.ProcessState(ctx, func(ctx context.Context, state *types.RewriteState) error {
				intent = state.Intent
				return nil
			})
			if err != nil {
				return "", err
			}
			switch {
			case strings.Contains(intent, "学生守则"):
				return "prepare_student_rules_input", nil
			case strings.Contains(intent, "员工规范"):
				return "prepare_employee_rules_input", nil
			default:
				return "other_scenario_output", nil
			}
		},
		map[string]bool{
			"prepare_student_rules_input":  true,
			"prepare_employee_rules_input": true,
			"other_scenario_output":        true,
		},
	))

	// 学生守则分支的边
	_ = sg.AddEdge("prepare_student_rules_input", "student_rules_prompt")
	_ = sg.AddEdge("student_rules_prompt", "student_rules_model_stream")
	_ = sg.AddEdge("student_rules_model_stream", "output_student_rules")
	_ = sg.AddEdge("output_student_rules", compose.END)

	// 员工规范分支的边
	_ = sg.AddEdge("prepare_employee_rules_input", "employee_rules_prompt")
	_ = sg.AddEdge("employee_rules_prompt", "employee_rules_model_stream")
	_ = sg.AddEdge("employee_rules_model_stream", "output_employee_rules")
	_ = sg.AddEdge("output_employee_rules", compose.END)

	// 其他场景分支的边
	_ = sg.AddEdge("other_scenario_output", compose.END)

	// 无效分支的边
	_ = sg.AddEdge("passthrough_node", compose.END)

	// 编译并返回图
	return sg.Compile(ctx)
}
