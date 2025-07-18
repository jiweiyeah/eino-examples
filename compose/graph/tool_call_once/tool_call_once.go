/*
 * Copyright 2025 CloudWeGo Authors
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
	"errors"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	"github.com/cloudwego/eino-examples/internal/gptr"
	"github.com/cloudwego/eino-examples/internal/logs"
)

func main() {
	OPENAI_BASE_URL := "https://api.siliconflow.cn/v1"
	OPENAI_API_KEY := "sk-rhlzvcpnvpbrlsvsggqbjwyosibwvqwxbotgfrbtzkeybfdr"
	OPENAI_MODEL_NAME := "deepseek-ai/DeepSeek-R1"

	openAIBaseURL := OPENAI_BASE_URL
	openAIAPIKey := OPENAI_API_KEY
	modelName := OPENAI_MODEL_NAME

	ctx := context.Background()

	systemTpl := `你是一名房产经纪人，结合用户的薪酬和工作，使用 user_info API，为其提供相关的房产信息。邮箱是必须的`
	chatTpl := prompt.FromMessages(schema.FString,
		schema.SystemMessage(systemTpl),
		schema.MessagesPlaceholder("message_histories", true),
		schema.UserMessage("{query}"),
	)

	chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		BaseURL:     openAIBaseURL,
		APIKey:      openAIAPIKey,
		Model:       modelName,
		Temperature: gptr.Of(float32(0.7)),
	})
	if err != nil {
		logs.Fatalf("NewChatModel failed, err=%v", err)
	}

	userInfoTool := utils.NewTool(
		&schema.ToolInfo{
			Name: "user_info",
			Desc: "根据用户的姓名和邮箱，查询用户的公司、职位、薪酬信息",
			ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
				"name": {
					Type: "string",
					Desc: "用户的姓名",
				},
				"email": {
					Type: "string",
					Desc: "用户的邮箱",
				},
			}),
		},
		func(ctx context.Context, input *userInfoRequest) (output *userInfoResponse, err error) {
			return &userInfoResponse{
				Name:     input.Name,
				Email:    input.Email,
				Company:  "Awesome company",
				Position: "CEO",
				Salary:   "9999",
			}, nil
		})

	info, err := userInfoTool.Info(ctx)
	if err != nil {
		logs.Fatalf("Get ToolInfo failed, err=%v", err)
	}

	err = chatModel.BindTools([]*schema.ToolInfo{info})
	if err != nil {
		logs.Fatalf("BindTools failed, err=%v", err)
	}

	toolsNode, err := compose.NewToolNode(ctx, &compose.ToolsNodeConfig{
		Tools: []tool.BaseTool{userInfoTool},
	})
	if err != nil {
		logs.Fatalf("NewToolNode failed, err=%v", err)
	}

	takeOne := compose.InvokableLambda(func(ctx context.Context, input []*schema.Message) (*schema.Message, error) {
		if len(input) == 0 {
			return nil, errors.New("input is empty")
		}
		return input[0], nil
	})

	const (
		nodeModel     = "node_model"
		nodeTools     = "node_tools"
		nodeTemplate  = "node_template"
		nodeConverter = "node_converter"
	)

	branch := compose.NewStreamGraphBranch(func(ctx context.Context, input *schema.StreamReader[*schema.Message]) (string, error) {
		defer input.Close()
		msg, err := input.Recv()
		if err != nil {
			return "", err
		}

		if len(msg.ToolCalls) > 0 {
			return nodeTools, nil
		}

		return compose.END, nil
	}, map[string]bool{compose.END: true, nodeTools: true})

	graph := compose.NewGraph[map[string]any, *schema.Message]()

	_ = graph.AddChatTemplateNode(nodeTemplate, chatTpl)
	_ = graph.AddChatModelNode(nodeModel, chatModel)
	_ = graph.AddToolsNode(nodeTools, toolsNode)
	_ = graph.AddLambdaNode(nodeConverter, takeOne)

	_ = graph.AddEdge(compose.START, nodeTemplate)
	_ = graph.AddEdge(nodeTemplate, nodeModel)
	_ = graph.AddBranch(nodeModel, branch)
	_ = graph.AddEdge(nodeTools, nodeConverter)
	_ = graph.AddEdge(nodeConverter, compose.END)

	r, err := graph.Compile(ctx)
	if err != nil {
		logs.Fatalf("Compile failed, err=%v", err)
	}

	out, err := r.Invoke(ctx, map[string]any{"query": "我叫 zhangsan, 邮箱是 zhangsan@bytedance.com, 帮我推荐一处房产"})
	if err != nil {
		logs.Fatalf("Invoke failed, err=%v", err)
	}

	logs.Infof("result content: %v", out.Content)
}

type userInfoRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type userInfoResponse struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Company  string `json:"company"`
	Position string `json:"position"`
	Salary   string `json:"salary"`
}
