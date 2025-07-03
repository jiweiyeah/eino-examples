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
	"io"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/joho/godotenv"

	"github.com/cloudwego/eino-examples/internal/gptr"
	"github.com/cloudwego/eino-examples/internal/logs"
)

// 处理消息内容的函数，可以被其他地方调用
func processMessageContent(content string) string {
	processedContent := "处理后的内容: " + content
	logs.Infof("消息内容已处理: %s", processedContent)
	return processedContent
}

// 流式发送内容的函数
func streamContent(content string, sw *schema.StreamWriter[string]) {
	defer sw.Close()
	
	// 逐字符发送内容，模拟流式输出
	for _, char := range content {
		// 添加一点延迟以模拟真实的流式输出
		time.Sleep(50 * time.Millisecond)
		sw.Send(string(char), nil)
	}
}

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
	b1 := compose.StreamableLambda(func(ctx context.Context, kvs map[string]any) (*schema.StreamReader[map[string]any], error) {
		logs.Infof("hello in branch lambda 01")
		if kvs == nil {
			return nil, fmt.Errorf("nil map")
		}

		kvs["role"] = "cat" // 将角色设置为 "cat"
		sr, sw := schema.Pipe[map[string]any](1)
		sw.Send(kvs, nil)
		sw.Close()
		return sr, nil
	})

	// b2 是一个 lambda 函数，作为分支 2
	b2 := compose.StreamableLambda(func(ctx context.Context, kvs map[string]any) (*schema.StreamReader[map[string]any], error) {
		logs.Infof("hello in branch lambda 02")
		if kvs == nil {
			return nil, fmt.Errorf("nil map")
		}

		kvs["role"] = "dog" // 将角色设置为 "dog"
		sr, sw := schema.Pipe[map[string]any](1)
		sw.Send(kvs, nil)
		sw.Close()
		return sr, nil
	})

	// 构建并行节点
	parallel := compose.NewParallel()
	parallel.
		// 添加一个 lambda 函数到并行节点，用于设置角色
		AddLambda("role", compose.StreamableLambda(func(ctx context.Context, kvs map[string]any) (*schema.StreamReader[string], error) {
			// 可以根据输入 kvs 更改角色
			role, ok := kvs["role"].(string)
			if !ok || role == "" {
				role = "bird" // 如果没有角色，默认为 "bird"
			}

			sr, sw := schema.Pipe[string](1)
			sw.Send(role, nil)
			sw.Close()
			return sr, nil
		})).
		// 添加另一个 lambda 函数到并行节点，用于设置输入问题
		AddLambda("input", compose.StreamableLambda(func(ctx context.Context, kvs map[string]any) (*schema.StreamReader[string], error) {
			sr, sw := schema.Pipe[string](1)
			sw.Send("你的叫声是怎样的？", nil)
			sw.Close()
			return sr, nil
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
		AppendLambda(compose.StreamableLambda(func(ctx context.Context, kvs map[string]any) (*schema.StreamReader[map[string]any], error) {
			// 在这里可以做一些逻辑来准备下一个节点的输入
			// 这里只是直接传递
			logs.Infof("in view lambda: %v", kvs)
			sr, sw := schema.Pipe[map[string]any](1)
			sw.Send(kvs, nil)
			sw.Close()
			return sr, nil
		})).
		// 添加分支节点，根据 branchCond 的结果选择 b1 或 b2
		AppendBranch(compose.NewChainBranch(branchCond).AddLambda("b1", b1).AddLambda("b2", b2)). // nolint: byted_use_receiver_without_nilcheck
		// 添加一个 Passthrough 节点，它会将输入直接传递给下一个节点
		AppendPassthrough().
		// 添加并行节点
		AppendParallel(parallel).
		// 添加一个图节点，这里是之前创建的 rolePlayerChain
		AppendGraph(rolePlayerChain).
		// 添加最后一个 lambda 函数，用于处理最终结果，现在改为流式输出每个字符
		AppendLambda(compose.StreamableLambda(func(ctx context.Context, m *schema.Message) (*schema.StreamReader[string], error) {
			// 保存消息内容到变量中
			originalContent := m.Content
			logs.Infof("in view of messages: %v", originalContent)
			
			// 调用处理函数处理内容
			processedContent := processMessageContent(originalContent)
			
			// 创建一个缓冲区大小为内容长度的管道
			sr, sw := schema.Pipe[string](len(processedContent))
			
			// 启动一个 goroutine 来模拟流式输出
			go streamContent(processedContent, sw)
			
			return sr, nil
		}))

	// 编译链
	r, err := chain.Compile(ctx)
	if err != nil {
		log.Panic(err)
		return
	}

	// 执行链 - 改为使用Stream方法
	outputStream, err := r.Stream(context.Background(), map[string]any{})
	if err != nil {
		log.Panic(err)
		return
	}

	// 处理流式输出
	for {
		chunk, err := outputStream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Panic(err)
			return
		}
		// 直接打印每个字符，不换行
		fmt.Print(chunk)
	}
}
