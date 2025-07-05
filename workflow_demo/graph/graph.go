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
	"fmt"

	"github.com/cloudwego/eino/compose"

	"github.com/cloudwego/eino-examples/internal/logs"
)

// NewSimpleGraph 编译一个简单的图并返回可运行实例
func NewSimpleGraph(ctx context.Context) (compose.Runnable[string, string], error) {
	// 创建一个新图，输入和输出都是字符串类型
	g := compose.NewGraph[string, string]()

	// 添加一个 lambda 节点 "node_1"
	_ = g.AddLambdaNode("node_1", compose.InvokableLambda(func(ctx context.Context, input string) (output string, err error) {
		logs.Infof("--- 节点 1 ---")
		logs.Infof("输入: %s", input)
		output = input + " 由节点1处理,"
		logs.Infof("输出: %s", output)
		return output, nil
	}))

	// 创建一个子图 sg
	sg := compose.NewGraph[string, string]()
	// 在子图中添加一个 lambda 节点 "sg_node_1"
	_ = sg.AddLambdaNode("sg_node_1", compose.InvokableLambda(func(ctx context.Context, input string) (output string, err error) {
		logs.Infof("--- 节点 2 (子图 sg_node_1) ---")
		logs.Infof("输入: %s", input)
		output = input + " 由sg_node_1处理,"
		logs.Infof("输出: %s", output)
		return output, nil
	}))

	// 在子图中添加入口到 "sg_node_1" 的边
	_ = sg.AddEdge(compose.START, "sg_node_1")

	// 在子图中添加 "sg_node_1" 到出口的边
	_ = sg.AddEdge("sg_node_1", compose.END)

	// 将子图 sg 添加为父图 g 的一个节点 "node_2"
	_ = g.AddGraphNode("node_2", sg)

	// 在父图 g 中添加一个 lambda 节点 "node_3"
	_ = g.AddLambdaNode("node_3", compose.InvokableLambda(func(ctx context.Context, input string) (output string, err error) {
		logs.Infof("--- 节点 3 ---")
		logs.Infof("输入: %s", input)
		output = input + " 由节点3处理,"
		logs.Infof("输出: %s", output)
		return output, nil
	}))

	// 在父图 g 中添加入口到 "node_1" 的边
	_ = g.AddEdge(compose.START, "node_1")

	// 在父图 g 中添加 "node_1" 到 "node_2" 的边
	_ = g.AddEdge("node_1", "node_2")

	// 在父图 g 中添加 "node_2" 到 "node_3" 的边
	_ = g.AddEdge("node_2", "node_3")

	// 在父图 g 中添加 "node_3" 到出口的边
	_ = g.AddEdge("node_3", compose.END)

	// 编译图
	return g.Compile(ctx)
}

// When using eino debugging plugin, in the input box, you need to specify the concrete type of 'any' in map[string]any. For example, you can input the following data for debugging:
// 使用 eino 调试插件时，需要在输入框中指定 map[string]any 中 'any' 的具体类型。例如，您可以输入以下数据进行调试：
//{
//	"name": {
//		"_value": "alice",
//		"_eino_go_type": "string"
//	},
//	"score": {
//		"_value": "99",
//		"_eino_go_type": "int"
//	}
//}

// RegisterAnyInputGraph 注册一个接受任意输入类型的图
func RegisterAnyInputGraph(ctx context.Context) {
	// 创建一个新图，输入为 map[string]any，输出为字符串
	g := compose.NewGraph[map[string]any, string]()

	// 添加一个 lambda 节点 "node_1"，处理 map[string]any 类型的输入
	_ = g.AddLambdaNode("node_1", compose.InvokableLambda(func(ctx context.Context, input map[string]any) (output string, err error) {
		for k, v := range input {
			switch v.(type) {
			case string:
				output += k + ":" + v.(string) + ","
			case int:
				output += k + ":" + fmt.Sprintf("%d", v.(int))
			default:
				return "", fmt.Errorf("不支持的类型: %T", v)
			}
		}

		return output, nil
	}))

	// 添加一个 lambda 节点 "node_2"
	_ = g.AddLambdaNode("node_2", compose.InvokableLambda(func(ctx context.Context, input string) (output string, err error) {
		return input + " 由节点2处理,", nil
	}))

	// 添加入口到 "node_1" 的边
	_ = g.AddEdge(compose.START, "node_1")

	// 添加 "node_1" 到 "node_2" 的边
	_ = g.AddEdge("node_1", "node_2")

	// 添加 "node_2" 到出口的边
	_ = g.AddEdge("node_2", compose.END)

	// 编译图
	r, err := g.Compile(ctx)
	if err != nil {
		logs.Errorf("编译图失败, err=%v", err)
		return
	}

	// 调用图
	message, err := r.Invoke(ctx, map[string]any{"name": "bob", "score": 100})
	if err != nil {
		logs.Errorf("调用图失败, err=%v", err)
		return
	}

	logs.Infof("eino 任意输入图的输出是: %v", message)
}
