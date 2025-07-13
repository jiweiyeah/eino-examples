package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/cloudwego/eino-ext/devops"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudwego/eino-examples/internal/logs"
	"github.com/cloudwego/eino-examples/workflow_demo_frontend/graph"
)

type ChatRequest struct {
	Message string `json:"message"`
}

type ChatResponse struct {
	Answer string `json:"answer"`
}

type SaveChatRequest struct {
	ID       string      `json:"id"`
	Messages []wsMessage `json:"messages"`
}

type wsMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func main() {
	ctx := context.Background()

	// 初始化 eino devops 服务
	// 跳过初始化以避免端口冲突

	err := devops.Init(ctx)
	if err != nil {
		logs.Errorf("[eino dev] 初始化失败, err=%v", err)
		return
	}

	// 编译工作流图
	workflow, err := graph.NewConditionalRewriterGraphStream(ctx)
	if err != nil {
		log.Fatalf("编译图失败, err: %v", err)
	}

	// 设置HTTP服务器
	http.HandleFunc("/api/chat", func(w http.ResponseWriter, r *http.Request) {
		// 允许跨域请求
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		// 处理预检请求
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// 只接受POST请求
		if r.Method != "POST" {
			http.Error(w, "只支持POST方法", http.StatusMethodNotAllowed)
			return
		}

		// 解析请求
		var req ChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "无效的请求格式", http.StatusBadRequest)
			return
		}

		// 检查消息是否为空
		if req.Message == "" {
			http.Error(w, "消息不能为空", http.StatusBadRequest)
			return
		}

		// 使用工作流处理消息
		stream, err := workflow.Stream(r.Context(), req.Message)
		if err != nil {
			logs.Errorf("调用图失败, err: %v", err)
			http.Error(w, "处理消息时出错", http.StatusInternalServerError)
			return
		}

		// 设置响应头为Server-Sent Events
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		// 处理流式输出
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "服务器不支持流式输出", http.StatusInternalServerError)
			return
		}

		// 处理客户端断开连接
		notify := r.Context().Done()
		go func() {
			<-notify
			logs.Infof("客户端断开连接")
		}()

		// 发送流式响应
		for {
			chunk, err := stream.Recv()
			if err != nil {
				if err == io.EOF {
					break // 流结束
				}
				logs.Errorf("从流中接收数据时出错, err: %v", err)
				break
			}
			log.Println("chunk：-----", chunk)

			// 发送事件
			fmt.Fprintf(w, "data: %s\n\n", chunk)
			flusher.Flush()
		}
	})

	// 新增API：保存聊天记录
	http.HandleFunc("/api/history", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		switch r.Method {
		case "POST":
			saveChatHistory(w, r)
		case "GET":
			getChatHistoryList(w, r)
		default:
			http.Error(w, "方法不支持", http.StatusMethodNotAllowed)
		}
	})

	// 新增API：获取指定聊天记录
	http.HandleFunc("/api/history/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		if r.Method == "GET" {
			getChatHistory(w, r)
		} else {
			http.Error(w, "方法不支持", http.StatusMethodNotAllowed)
		}
	})

	// 添加 /chat/ 路径处理，重定向到根路径
	http.HandleFunc("/chat/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/", http.StatusMovedPermanently)
	})

	// 提供静态文件服务
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/", fs)

	// 启动服务器
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("服务器启动在 http://localhost:%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func saveChatHistory(w http.ResponseWriter, r *http.Request) {
	var req SaveChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "无效的请求格式", http.StatusBadRequest)
		return
	}

	if req.ID == "" || len(req.Messages) == 0 {
		http.Error(w, "ID和消息不能为空", http.StatusBadRequest)
		return
	}

	filePath := filepath.Join("chat_logs", req.ID+".json")
	file, err := os.Create(filePath)
	if err != nil {
		http.Error(w, "无法创建聊天记录文件", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	if err := json.NewEncoder(file).Encode(req.Messages); err != nil {
		http.Error(w, "无法写入聊天记录", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func getChatHistoryList(w http.ResponseWriter, r *http.Request) {
	files, err := os.ReadDir("chat_logs")
	if err != nil {
		if os.IsNotExist(err) {
			os.MkdirAll("chat_logs", 0755)
			files = []os.DirEntry{}
		} else {
			http.Error(w, "无法读取聊天记录目录", http.StatusInternalServerError)
			return
		}
	}

	history := []map[string]string{}
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".json") {
			filePath := filepath.Join("chat_logs", file.Name())
			content, err := os.ReadFile(filePath)
			if err != nil {
				continue
			}
			var messages []wsMessage
			if err := json.Unmarshal(content, &messages); err == nil && len(messages) > 0 {
				history = append(history, map[string]string{
					"id":    strings.TrimSuffix(file.Name(), ".json"),
					"title": messages[0].Content,
				})
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(history)
}

func getChatHistory(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/history/")
	if id == "" {
		http.Error(w, "缺少聊天ID", http.StatusBadRequest)
		return
	}

	filePath := filepath.Join("chat_logs", id+".json")
	content, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "聊天记录不存在", http.StatusNotFound)
		} else {
			http.Error(w, "无法读取聊天记录文件", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(content)
}
