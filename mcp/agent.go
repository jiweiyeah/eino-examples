package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/cloudwego/eino-examples/internal/logs"
	"github.com/cloudwego/eino-ext/devops"
	"github.com/cloudwego/eino/schema"
)

func main() {
	// 配置日志输出到文件
	logFile, err := os.OpenFile("agent.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("无法打开日志文件: %v", err)
	}
	defer logFile.Close()
	log.SetOutput(logFile)

	// 启动 MCP 服务端（工具服务）
	startMCPServer()

	// 等待服务端启动完成
	time.Sleep(1 * time.Second)

	// 创建上下文
	ctx := context.Background()
	// 初始化 eino devops 服务
	err = devops.Init(ctx)
	if err != nil {
		logs.Errorf("[eino dev] 初始化失败, err=%v", err)
		return
	}

	// 从 MCP 客户端中获取工具

	//1.新建图
	compile, err := buildAgent(ctx)
	if err != nil {
		logs.Errorf("[eino compile] err=%v", err)
	}

	input := map[string]any{
		"role":         "工具人",
		"query":        "请帮我重复 'Hello Eino!' 这句话",
		"chat_history": []*schema.Message{},
	}
	//流式调用
	stream, err := compile.Stream(ctx, input)
	if err != nil {
		logs.Errorf("[eino Stream] err=%v", err)
	}

	for {
		message, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				// finish
				break
			}
			// error
			log.Printf("failed to recv: %v\n", err)
			return
		}
		fmt.Print(message.Content)
		logs.Tokenf(message.Content)
	}

	//output, err := compile.Invoke(ctx, input)
	//if err != nil {
	//	logs.Errorf("[eino Invoke] %v", err)
	//}
	//log.Println("Echo 工具调用结果: ", output.Content)
	//
	//// 调用 reverse_string 工具的例子
	//input = map[string]any{
	//	"role":         "工具人",
	//	"query":        "请帮我把 'OpenAI' 这个词反过来写",
	//	"chat_history": []*schema.Message{},
	//}
	//
	//output, err = compile.Invoke(ctx, input)
	//if err != nil {
	//	logs.Errorf("[eino Invoke] %v", err)
	//}
	//log.Println("Reverse String 工具调用结果: ", output.Content)
	//
	// 调用 reverse_string 工具的例子
	input = map[string]any{
		"role":         "工具人",
		"query":        "2636*6261是多少",
		"chat_history": []*schema.Message{},
	}
	output, err := compile.Invoke(ctx, input)
	if err != nil {
		logs.Errorf("[eino Invoke] %v", err)
	}
	log.Println("计算器工具调用结果: ", output.Content)
	fmt.Print(output.Content)
	select {}
}

func init() {
	log.Println("[INIT] 安装 LogRoundTripper 到 http.DefaultTransport")

	defaultTransport := http.DefaultTransport

	http.DefaultTransport = &LogRoundTripper{
		Proxied: defaultTransport,
	}

}

type LogRoundTripper struct {
	Proxied http.RoundTripper // 原始的 RoundTripper，用于链式调用
}

func (lrt *LogRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	log.Printf("[HTTP] 请求: %s %s", req.Method, req.URL.String())
	//不用打印 apikey

	/*	for name, headers := range req.Header {
			for _, h := range headers {
				log.Printf("[HTTP] 请求头: %v: %v", name, h)
			}
		}
	*/
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

	responseBodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[HTTP] 读取响应体失败: %v", err)
	} else {
		log.Printf("[HTTP] 响应体: %s", string(responseBodyBytes))
	}
	resp.Body = io.NopCloser(bytes.NewBuffer(responseBodyBytes))

	return resp, nil
}
