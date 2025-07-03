/*
 * Copyright 2024 CloudWeGo Authors
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
	"fmt"
	"log"
	"math/rand"
	"os"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/joho/godotenv"

	"github.com/cloudwego/eino-examples/internal/gptr"
	"github.com/cloudwego/eino-examples/internal/logs"
)

func main() {
	// load .env file
	err := godotenv.Load("../../.env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	// 从环境变量中获取 OpenAI 的 baseURL、API Key 和模型名称
	openAPIBaseURL := os.Getenv("OPENAI_BASE_URL")
	openAPIAK := os.Getenv("OPENAI_API_KEY")
	modelName := os.Getenv("OPENAI_MODEL_NAME")

	ctx := context.Background()
	// 构建分支函数
	const randLimit = 2
	// branchCond 是一个条件函数，用于决定执行哪个分支
	branchCond := func(ctx context.Context, input map[string]any) (string, error) { // nolint: byted_all_nil_return
		if rand.Intn(randLimit) == 1 {
			return "b1", nil // 随机返回 "b1" 或 "b2"
		}

		return "b2", nil
	}

	// b1 是一个 lambda 函数，作为分支 1
	b1 := compose.InvokableLambda(func(ctx context.Context, kvs map[string]any) (map[string]any, error) {
		logs.Infof("hello in branch lambda 01")
		if kvs == nil {
			return nil, fmt.Errorf("nil map")
		}

		kvs["role"] = "cat" // 将角色设置为 "cat"
		return kvs, nil
	})

	// b2 是一个 lambda 函数，作为分支 2
	b2 := compose.InvokableLambda(func(ctx context.Context, kvs map[string]any) (map[string]any, error) {
		logs.Infof("hello in branch lambda 02")
		if kvs == nil {
			return nil, fmt.Errorf("nil map")
		}

		kvs["role"] = "dog" // 将角色设置为 "dog"
		return kvs, nil
	})

	// 构建并行节点
	parallel := compose.NewParallel()
	parallel.
		// 添加一个 lambda 函数到并行节点，用于设置角色
		AddLambda("role", compose.InvokableLambda(func(ctx context.Context, kvs map[string]any) (string, error) {
			// 可以根据输入 kvs 更改角色
			role, ok := kvs["role"].(string)
			if !ok || role == "" {
				role = "bird" // 如果没有角色，默认为 "bird"
			}

			return role, nil
		})).
		// 添加另一个 lambda 函数到并行节点，用于设置输入问题
		AddLambda("input", compose.InvokableLambda(func(ctx context.Context, kvs map[string]any) (string, error) {
			return "你的叫声是怎样的？", nil
		}))

	// 创建聊天模型节点
	cm, err := openai.NewChatModel(context.Background(), &openai.ChatModelConfig{
		BaseURL:     openAPIBaseURL,
		APIKey:      openAPIAK,
		Model:       modelName,
		Temperature: gptr.Of(float32(0.7)),
	})
	if err != nil {
		log.Panic(err)
		return
	}

	// 创建一个 "角色扮演" 链，它接收一个 map 作为输入，输出一个 schema.Message
	rolePlayerChain := compose.NewChain[map[string]any, *schema.Message]()
	rolePlayerChain.
		// 添加聊天模板，用于格式化输入
		AppendChatTemplate(prompt.FromMessages(schema.FString, schema.SystemMessage(`You are a {role}.`), schema.UserMessage(`{input}`))).
		// 添加聊天模型
		AppendChatModel(cm)

	// =========== 构建主链 ===========
	chain := compose.NewChain[map[string]any, string]()
	chain.
		// 添加一个 lambda 函数作为链的开始，用于准备数据
		AppendLambda(compose.InvokableLambda(func(ctx context.Context, kvs map[string]any) (map[string]any, error) {
			// 在这里可以做一些逻辑来准备下一个节点的输入
			// 这里只是直接传递
			logs.Infof("in view lambda: %v", kvs)
			return kvs, nil
		})).
		// 添加分支节点，根据 branchCond 的结果选择 b1 或 b2
		AppendBranch(compose.NewChainBranch(branchCond).AddLambda("b1", b1).AddLambda("b2", b2)). // nolint: byted_use_receiver_without_nilcheck
		// 添加一个 Passthrough 节点，它会将输入直接传递给下一个节点
		AppendPassthrough().
		// 添加并行节点
		AppendParallel(parallel).
		// 添加一个图节点，这里是之前创建的 rolePlayerChain
		AppendGraph(rolePlayerChain).
		// 添加最后一个 lambda 函数，用于处理最终结果
		AppendLambda(compose.InvokableLambda(func(ctx context.Context, m *schema.Message) (string, error) {
			// 在这里可以对输出做一些检查或其他逻辑
			logs.Infof("in view of messages: %v", m.Content)
			return m.Content, nil
		}))

	// 编译链
	r, err := chain.Compile(ctx)
	if err != nil {
		log.Panic(err)
		return
	}

	// 执行链
	output, err := r.Invoke(context.Background(), map[string]any{})
	if err != nil {
		log.Panic(err)
		return
	}

	logs.Infof("output is : %v", output)
}
