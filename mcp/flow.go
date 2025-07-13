package main

import (
	"context"
	"github.com/cloudwego/eino-examples/internal/logs"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
)

func newLambda(ctx context.Context) (lba *compose.Lambda, err error) {
	// TODO Modify component configuration here.
	config := &react.AgentConfig{
		MaxStep:            25,
		ToolReturnDirectly: map[string]struct{}{}}
	chatModelIns11, err := newChatModel(ctx)
	if err != nil {
		return nil, err
	}
	config.ToolCallingModel = chatModelIns11
	tools := getMCPTool(ctx)
	if err != nil {
		return nil, err
	}
	config.ToolsConfig.Tools = tools
	ins, err := react.NewAgent(ctx, config)
	if err != nil {
		return nil, err
	}
	lba, err = compose.AnyLambda(ins.Generate, ins.Stream, nil, nil)
	if err != nil {
		return nil, err
	}
	return lba, nil
}

func newChatModel(ctx context.Context) (*openai.ChatModel, error) {
	openaiModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		APIKey:  "sk-rhlzvcpnvpbrlsvsggqbjwyosibwvqwxbotgfrbtzkeybfdr",
		Model:   "Qwen/Qwen3-8B",
		BaseURL: "https://api.siliconflow.cn/v1",
	})
	if err != nil {
		logs.Errorf("failed to create chat model: %v", err)
		return nil, err
	}
	return openaiModel, nil
}
