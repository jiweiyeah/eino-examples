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
	"bytes"
	"context"
	"fmt"
	"github.com/cloudwego/eino-examples/internal/logs"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/schema"
	"io"
	"log"
	"net/http"
)

func main() {
	systemPrompt := "你是一个{role}"

	ctx := context.Background()

	// 创建模板
	template := prompt.FromMessages(schema.FString,
		schema.SystemMessage(systemPrompt),
		//schema.MessagesPlaceholder("history_key", false),
		&schema.Message{
			Role: schema.User,
			Content: "[原始问题]:[{query}]，[工具调用答案]:[{toolresult}]，请你分析" +
				"[工具调用答案]是否能回答[原始问题]。如果无法回复，请直接说无法回复",
		},
	)

	// 准备变量
	variables := map[string]any{
		"role":  "专业的助手",
		"query": "什么是宇宙",
		"history_key": []*schema.Message{{Role: schema.User, Content: "告诉我油画是什么?"},
			{Role: schema.Assistant, Content: "油画是xxx"}},
		"toolresult": "噶哈哈haaaaaaaaaaaa哈哈",
	}

	// 格式化模板
	messages, err := template.Format(context.Background(), variables)
	if err != nil {
		panic(err)
	}
	chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		APIKey:  "sk-rhlzvcpnvpbrlsvsggqbjwyosibwvqwxbotgfrbtzkeybfdr",
		Model:   "Qwen/Qwen3-8B",
		BaseURL: "https://api.siliconflow.cn/v1",
	})
	if err != nil {
		logs.Errorf("failed to create chat model: %v", err)
		return
	}

	msg, err := chatModel.Generate(ctx, messages)
	if err != nil {
		logs.Errorf("failed to generate message: %v", err)
	}
	fmt.Println(msg)
}

// init 会在 main 之前执行，设置全局 HTTP 日志拦截
func init() {
	log.Println("[INIT] 安装 LogRoundTripper 到 http.DefaultTransport")

	defaultTransport := http.DefaultTransport

	http.DefaultTransport = &LogRoundTripper{
		Proxied: defaultTransport,
	}

}

// LogRoundTripper 是一个自定义的 RoundTripper，用于日志记录 HTTP 请求和响应。
// 这个定义仍然在这里，但如之前讨论，为了实际生效，它需要被注入到 eino-ext 库内部的 http.Client.Transport 中。
type LogRoundTripper struct {
	Proxied http.RoundTripper // 原始的 RoundTripper，用于链式调用
}

// RoundTrip 实现了 http.RoundTripper 接口。
// 在请求发送前和响应接收后打印详细信息。
func (lrt *LogRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	log.Printf("[HTTP] 请求: %s %s", req.Method, req.URL.String())

	for name, headers := range req.Header {
		for _, h := range headers {
			log.Printf("[HTTP] 请求头: %v: %v", name, h)
		}
	}

	// 打印请求体（如果存在）
	var requestBodyBytes []byte
	if req.Body != nil {
		var err error
		requestBodyBytes, err = io.ReadAll(req.Body)
		if err != nil {
			log.Printf("[HTTP] 读取请求体失败: %v", err)
		} else {
			log.Printf("[HTTP] 请求体: %s", string(requestBodyBytes))
			// 重置请求体（否则后续无法再次读取）
			req.Body = io.NopCloser(bytes.NewBuffer(requestBodyBytes))
		}
	}

	resp, err := lrt.Proxied.RoundTrip(req)
	if err != nil {
		log.Printf("[HTTP] 请求错误: %v", err)
		return resp, err
	}

	log.Printf("[HTTP] 响应状态: %s", resp.Status)

	//responseBodyBytes, err := io.ReadAll(resp.Body)
	//if err != nil {
	//	log.Printf("[HTTP] 读取响应体失败: %v", err)
	//} else {
	//	log.Printf("[HTTP] 响应体: %s", string(responseBodyBytes))
	//}
	//resp.Body = io.NopCloser(bytes.NewBuffer(responseBodyBytes))

	return resp, nil
}
