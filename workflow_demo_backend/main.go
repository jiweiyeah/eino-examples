package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"github.com/cloudwego/eino-examples/internal/logs"
	"github.com/cloudwego/eino-examples/workflow_demo_backend/graph"
	"github.com/cloudwego/eino-ext/devops"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

func main() {
	ctx := context.Background()

	// 初始化 eino devops 服务
	err := devops.Init(ctx)
	if err != nil {
		logs.Errorf("[eino dev] 初始化失败, err=%v", err)
		return
	}

	Invokeuse(ctx)

}

func Invokeuse(ctx context.Context) {

	rewriterGraph, err := graph.NewConditionalRewriterGraph(ctx)
	if err != nil {
		panic(err)
	}
	question := "你是谁"
	output, err := rewriterGraph.Stream(ctx, question)
	if err != nil {
		panic(err)
	}
	log.Println("输出结果是：", output)
}
func Streamuse(ctx context.Context) {
	// 编译工作流图
	workflow, err := graph.NewConditionalRewriterGraphStream(ctx)
	if err != nil {
		log.Fatalf("编译图失败, err: %v", err)
	}

	// 从控制台读取输入并持续对话
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("\n请输入您的问题 (输入 'exit' 来结束对话): ")
		userInput, _ := reader.ReadString('\n')
		userInput = strings.TrimSpace(userInput)

		if userInput == "exit" {
			fmt.Println("再见！")
			break
		}

		if userInput == "" {
			continue
		}

		// 使用输入内容以流式方式调用工作流
		stream, err := workflow.Stream(ctx, userInput)
		if err != nil {
			logs.Errorf("调用图失败, err: %v", err)
			continue
		}

		// 处理流式输出
		fmt.Print("AI: ")
		for {
			chunk, err := stream.Recv()
			if err != nil {
				if err == io.EOF {
					break // 流结束
				}
				logs.Errorf("从流中接收数据时出错, err: %v", err)
				break
			}
			fmt.Print(chunk)
		}
		fmt.Println() // 在每次输出后换行
	}
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
