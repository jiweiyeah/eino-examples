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

package graph

import (
	"context"

	"github.com/cloudwego/eino/compose"

	"github.com/cloudwego/eino-examples/internal/logs"
)

// nodeState 定义了节点的状态结构
type nodeState struct {
	Messages []string // Messages 用于存储节点处理过程中的消息
}

// RegisterSimpleStateGraph 注册一个简单的状态图
func RegisterSimpleStateGraph(ctx context.Context) {
	// stateFunction 是一个状态生成函数，用于为每个图的调用初始化一个新的状态
	stateFunction := func(ctx context.Context) *nodeState {
		s := &nodeState{
			Messages: make([]string, 0, 3),
		}
		return s
	}

	// 创建一个新的图，并使用 WithGenLocalState 选项来提供状态生成函数
	sg := compose.NewGraph[string, string](compose.WithGenLocalState(stateFunction))

	// 添加 "node_1" 节点，并为其设置一个状态前置处理器
	_ = sg.AddLambdaNode("node_1", compose.InvokableLambda(func(ctx context.Context, input string) (output string, err error) {
		return input + " 由节点1处理,", nil
	}), compose.WithStatePreHandler(func(ctx context.Context, input string, state *nodeState) (string, error) {
		// 在节点处理前，将输入消息追加到状态的 Messages 中
		state.Messages = append(state.Messages, input)
		return input, nil
	}))

	// 添加 "node_2" 节点，并为其设置一个状态前置处理器
	_ = sg.AddLambdaNode("node_2", compose.InvokableLambda(func(ctx context.Context, input string) (output string, err error) {
		return input + " 由节点2处理,", nil
	}), compose.WithStatePreHandler(func(ctx context.Context, input string, state *nodeState) (string, error) {
		// 在节点处理前，将输入消息追加到状态的 Messages 中
		state.Messages = append(state.Messages, input)
		return input, nil
	}))

	// 添加 "node_3" 节点，并为其设置一个状态前置处理器
	_ = sg.AddLambdaNode("node_3", compose.InvokableLambda(func(ctx context.Context, input string) (output string, err error) {
		return input + " 由节点3处理,", nil
	}), compose.WithStatePreHandler(func(ctx context.Context, input string, state *nodeState) (string, error) {
		// 在节点处理前，将输入消息追加到状态的 Messages 中
		state.Messages = append(state.Messages, input)
		return input, nil
	}))

	// 定义图的边
	_ = sg.AddEdge(compose.START, "node_1")
	_ = sg.AddEdge("node_1", "node_2")
	_ = sg.AddEdge("node_2", "node_3")
	_ = sg.AddEdge("node_3", compose.END)

	// 编译图
	r, err := sg.Compile(ctx)
	if err != nil {
		logs.Errorf("编译状态图失败, err=%v", err)
		return
	}

	// 调用已编译的图
	message, err := r.Invoke(ctx, "eino state graph test")
	if err != nil {
		logs.Errorf("调用状态图失败, err=%v", err)
		return
	}

	logs.Infof("eino 简单状态图的输出是: %v", message)
}
