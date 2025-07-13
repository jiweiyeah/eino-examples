package main

import (
	"context"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/schema"
)

func newChatTemplate(ctx context.Context) (ctp prompt.ChatTemplate, err error) {
	systemPrompt := "你是一个{role}" // 工具人
	userPrompt := "[原始问题]:{query}，你的任务就是调用工具，输出选择的工具和答案，以及对调用工具的答案是否正确进行分析。如果无法回答，请直接输出无法回答"
	template := prompt.FromMessages(schema.FString,
		schema.SystemMessage(systemPrompt),
		&schema.Message{
			Role:    schema.User,
			Content: userPrompt,
		},
	)
	return template, nil
}
