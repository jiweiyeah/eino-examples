package reconstruct_refactored

import (
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/schema"
)

// GetClassifierPrompt 返回分类器提示词
func GetClassifierPrompt() *prompt.DefaultChatTemplate {
	return prompt.FromMessages(
		schema.FString,
		schema.SystemMessage("你是一个查询分类器。判断用户的输入是一个有效问题还是无效的垃圾信息/辱骂。请只回答 valid 或 invalid。"),
		schema.UserMessage("用户输入: {input}"),
	)
}

// GetRewriterPrompt 返回重写器提示词
func GetRewriterPrompt() *prompt.DefaultChatTemplate {
	return prompt.FromMessages(
		schema.FString,
		schema.SystemMessage("你是一位专业的查询重写专家。请将用户的问题改写得更清晰、更适合搜索引擎。"),
		schema.UserMessage("用户问题: {input}"),
	)
}

// GetIntentClassifierPrompt 返回意图分类器提示词
func GetIntentClassifierPrompt() *prompt.DefaultChatTemplate {
	return prompt.FromMessages(
		schema.FString,
		schema.SystemMessage("你是一个意图分类器。根据用户问题判断意图场景。1、若用户的问题属于学生守则类的场景, 返回学生守则；2、若用户的问题属于员工规范类的场景，则返回员工规范；3、否则属于其他场景，返回其他。请只返回这三个词中的一个。"),
		schema.UserMessage("用户问题: {input}"),
	)
}

// GetStudentRulesPrompt 返回学生守则提示词
func GetStudentRulesPrompt() *prompt.DefaultChatTemplate {
	return prompt.FromMessages(
		schema.FString,
		schema.SystemMessage("你是一个AI助手，专门回答关于学生守则的问题。请根据用户的问题，提供详细和准确的回答。"),
		schema.UserMessage("问题: {input}"),
	)
}

// GetEmployeeRulesPrompt 返回员工规范提示词
func GetEmployeeRulesFrompt() *prompt.DefaultChatTemplate {
	return prompt.FromMessages(
		schema.FString,
		schema.SystemMessage("你是一个AI助手，专门回答关于员工规范的问题。请根据用户的问题，提供详细和准确的回答。"),
		schema.UserMessage("问题: {input}"),
	)
}

