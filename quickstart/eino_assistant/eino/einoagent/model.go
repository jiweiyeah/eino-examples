package einoagent

import (
	"context"
	"github.com/cloudwego/eino-examples/internal/logs"
	"os"

	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
)

// newOpenAIChatModel 创建并返回一个 OpenAI 聊天模型 (ToolCallingChatModel 接口)
func newOpenAIChatModel(ctx context.Context) (model.ToolCallingChatModel, error) {
	modelConfig := &openai.ChatModelConfig{
		Model:   os.Getenv("OPENAI_CHAT_MODEL"),
		BaseURL: os.Getenv("OPENAI_BASE_URL"),
		APIKey:  os.Getenv("OPENAI_API_KEY"),
	}

	cm, err := openai.NewChatModel(ctx, modelConfig)
	if err != nil {
		return nil, err
	}

	logs.Infof("创建 OpenAI 模型成功")
	// 因为 *openai.ChatModel 已经实现了 model.ToolCallingChatModel 接口，可以直接返回
	return cm, nil
}

// newArkChatModel 创建并返回一个 Ark 聊天模型 (ToolCallingChatModel 接口)
func newArkChatModel(ctx context.Context) (model.ToolCallingChatModel, error) {
	modelConfig := &ark.ChatModelConfig{
		Model:  os.Getenv("ARK_CHAT_MODEL"),
		APIKey: os.Getenv("ARK_API_KEY"),
	}

	cm, err := ark.NewChatModel(ctx, modelConfig)
	if err != nil {
		return nil, err
	}

	logs.Infof("创建 Ark 模型成功")
	// 因为 *ark.ChatModel 已经实现了 model.ToolCallingChatModel 接口，可以直接返回
	return cm, nil
}

// newChatModel 作为工厂函数，根据环境变量选择创建不同类型的聊天模型
func newChatModel(ctx context.Context) (cm model.ToolCallingChatModel, err error) {
	provider := os.Getenv("CHAT_MODEL_PROVIDER")

	switch provider {
	case "ark":
		return newArkChatModel(ctx)
	case "openai":
		return newOpenAIChatModel(ctx)
	default:
		// 默认使用 OpenAI 模型
		logs.Errorf("未设置 CHAT_MODEL_PROVIDER 环境变量或其值无效，默认使用 OpenAI 模型。")
		return newOpenAIChatModel(ctx)
	}
}
