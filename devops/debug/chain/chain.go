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

package chain

import (
	"context"

	"github.com/cloudwego/eino/compose"

	"github.com/cloudwego/eino-examples/internal/logs"
)

// RegisterSimpleChain 注册一个简单的调用链
func RegisterSimpleChain(ctx context.Context) {
	// 创建一个新的调用链，输入和输出都为字符串类型
	chain := compose.NewChain[string, string]()

	// 定义第一个调用节点 c1
	c1 := compose.InvokableLambda(func(ctx context.Context, input string) (output string, err error) {
		return input + " 由节点1处理,", nil
	})

	// 定义第二个调用节点 c2
	c2 := compose.InvokableLambda(func(ctx context.Context, input string) (output string, err error) {
		return input + " 由节点2处理,", nil
	})

	// 将 c1 和 c2 节点追加到调用链中，并为它们命名
	chain.AppendLambda(c1, compose.WithNodeName("c1")).
		AppendLambda(c2, compose.WithNodeName("c2"))

	// 编译调用链
	r, err := chain.Compile(ctx)
	if err != nil {
		logs.Infof("编译链失败, err=%v", err)
		return
	}

	// 调用已编译的链
	message, err := r.Invoke(ctx, "eino chain test")
	if err != nil {
		logs.Infof("调用链失败, err=%v", err)
		return
	}

	logs.Infof("eino 简单调用链的输出是: %v", message)
}
