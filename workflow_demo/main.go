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
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/cloudwego/eino-ext/devops"

	"github.com/cloudwego/eino-examples/internal/logs"
	"github.com/cloudwego/eino-examples/workflow_demo/graph"
)

func main() {
	ctx := context.Background()

	// 初始化 eino devops 服务
	err := devops.Init(ctx)
	if err != nil {
		logs.Errorf("[eino dev] 初始化失败, err=%v", err)
		return
	}

	// Register chain, graph and state_graph for demo use
	// 编译工作流图
	simpleGraph, err := graph.NewSimpleGraph(ctx)
	if err != nil {
		logs.Errorf("编译图失败, err: %v", err)
		return
	}

	// 从控制台读取输入
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("请输入内容: ")
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	// 使用输入内容调用工作流
	res, err := simpleGraph.Invoke(ctx, input)
	if err != nil {
		logs.Errorf("调用图失败, err: %v", err)
		return
	}
	logs.Infof("工作流输出: %s", res)

	// 阻塞进程退出
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs

	// 退出
	logs.Infof("[eino dev] 关闭\n")
}
