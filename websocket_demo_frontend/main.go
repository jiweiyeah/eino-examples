package main

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/cloudwego/eino-examples/internal/logs"
	"github.com/cloudwego/eino-examples/websocket_demo_frontend/graph"
	"github.com/cloudwego/eino/compose"
	"github.com/gorilla/websocket"
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

// WebSocket消息类型
type wsCommand struct {
	Type    string          `json:"type"`    // "chat", "save", "history", "load"
	Payload json.RawMessage `json:"payload"` // 不同类型的消息有不同的payload
}

// WebSocket连接管理器
type wsConnectionManager struct {
	connections map[*websocket.Conn]bool
	mutex       sync.Mutex
}

// 创建新的连接管理器
func newWSConnectionManager() *wsConnectionManager {
	return &wsConnectionManager{
		connections: make(map[*websocket.Conn]bool),
	}
}

// 添加连接
func (m *wsConnectionManager) addConnection(conn *websocket.Conn) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.connections[conn] = true
	logs.Infof("新的WebSocket连接，当前连接数: %d", len(m.connections))
}

// 移除连接
func (m *wsConnectionManager) removeConnection(conn *websocket.Conn) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if _, ok := m.connections[conn]; ok {
		delete(m.connections, conn)
		conn.Close()
		logs.Infof("WebSocket连接关闭，当前连接数: %d", len(m.connections))
	}
}

// WebSocket升级器
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // 允许所有来源的WebSocket连接
	},
}

func main() {
	ctx := context.Background()

	// 编译工作流图
	workflow, err := graph.NewConditionalRewriterGraphStream(ctx)
	if err != nil {
		log.Fatalf("编译图失败, err: %v", err)
	}

	// 创建WebSocket连接管理器
	wsManager := newWSConnectionManager()

	// 设置WebSocket处理器
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			logs.Errorf("WebSocket升级失败: %v", err)
			return
		}

		// 添加新连接
		wsManager.addConnection(conn)

		// 处理连接关闭
		defer wsManager.removeConnection(conn)

		// 处理WebSocket消息
		for {
			messageType, message, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					logs.Errorf("WebSocket读取错误: %v", err)
				}
				break
			}

			if messageType != websocket.TextMessage {
				continue
			}

			// 解析命令
			var cmd wsCommand
			if err := json.Unmarshal(message, &cmd); err != nil {
				logs.Errorf("解析WebSocket消息失败: %v", err)
				sendErrorMessage(conn, "无效的消息格式")
				continue
			}

			// 根据命令类型处理
			switch cmd.Type {
			case "chat":
				var chatReq ChatRequest
				if err := json.Unmarshal(cmd.Payload, &chatReq); err != nil {
					sendErrorMessage(conn, "无效的聊天请求格式")
					continue
				}
				handleChatRequest(r.Context(), conn, workflow, chatReq)
			case "save":
				var saveReq SaveChatRequest
				if err := json.Unmarshal(cmd.Payload, &saveReq); err != nil {
					sendErrorMessage(conn, "无效的保存请求格式")
					continue
				}
				handleSaveChat(conn, saveReq)
			case "history":
				handleGetChatHistory(conn)
			case "load":
				var loadReq struct {
					ID string `json:"id"`
				}
				if err := json.Unmarshal(cmd.Payload, &loadReq); err != nil {
					sendErrorMessage(conn, "无效的加载请求格式")
					continue
				}
				handleLoadChat(conn, loadReq.ID)
			default:
				sendErrorMessage(conn, "未知的命令类型")
			}
		}
	})

	// 提供静态文件服务
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/", fs)

	// 启动服务器
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("WebSocket服务器启动在 http://localhost:%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

// 发送错误消息
func sendErrorMessage(conn *websocket.Conn, errMsg string) {
	response := map[string]interface{}{
		"type":    "error",
		"message": errMsg,
	}
	if err := conn.WriteJSON(response); err != nil {
		logs.Errorf("发送错误消息失败: %v", err)
	}
}

// 处理聊天请求
func handleChatRequest(ctx context.Context, conn *websocket.Conn, workflow compose.Runnable[string, string], req ChatRequest) {
	// 检查消息是否为空
	if req.Message == "" {
		sendErrorMessage(conn, "消息不能为空")
		return
	}

	// 使用工作流处理消息
	stream, err := workflow.Stream(ctx, req.Message)
	if err != nil {
		logs.Errorf("调用图失败, err: %v", err)
		sendErrorMessage(conn, "处理消息时出错")
		return
	}

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

		// 发送WebSocket消息
		response := map[string]interface{}{
			"type":    "chat_chunk",
			"content": chunk,
		}
		if err := conn.WriteJSON(response); err != nil {
			logs.Errorf("发送WebSocket消息失败: %v", err)
			break
		}
	}

	// 发送流结束标记
	endResponse := map[string]interface{}{
		"type": "chat_end",
	}
	if err := conn.WriteJSON(endResponse); err != nil {
		logs.Errorf("发送WebSocket流结束消息失败: %v", err)
	}
}

// 处理保存聊天记录
func handleSaveChat(conn *websocket.Conn, req SaveChatRequest) {
	if req.ID == "" || len(req.Messages) == 0 {
		sendErrorMessage(conn, "ID和消息不能为空")
		return
	}

	// 确保目录存在
	os.MkdirAll("chat_logs", 0755)

	filePath := filepath.Join("chat_logs", req.ID+".json")
	file, err := os.Create(filePath)
	if err != nil {
		sendErrorMessage(conn, "无法创建聊天记录文件")
		return
	}
	defer file.Close()

	if err := json.NewEncoder(file).Encode(req.Messages); err != nil {
		sendErrorMessage(conn, "无法写入聊天记录")
		return
	}

	// 发送成功响应
	response := map[string]interface{}{
		"type":    "save_success",
		"message": "聊天记录保存成功",
	}
	if err := conn.WriteJSON(response); err != nil {
		logs.Errorf("发送保存成功消息失败: %v", err)
	}
}

// 获取聊天历史列表
func handleGetChatHistory(conn *websocket.Conn) {
	// 确保目录存在
	os.MkdirAll("chat_logs", 0755)

	files, err := os.ReadDir("chat_logs")
	if err != nil {
		sendErrorMessage(conn, "无法读取聊天记录目录")
		return
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

	// 发送历史记录响应
	response := map[string]interface{}{
		"type":    "history_list",
		"history": history,
	}
	if err := conn.WriteJSON(response); err != nil {
		logs.Errorf("发送历史记录列表失败: %v", err)
	}
}

// 加载指定聊天记录
func handleLoadChat(conn *websocket.Conn, id string) {
	if id == "" {
		sendErrorMessage(conn, "缺少聊天ID")
		return
	}

	filePath := filepath.Join("chat_logs", id+".json")
	content, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			sendErrorMessage(conn, "聊天记录不存在")
		} else {
			sendErrorMessage(conn, "无法读取聊天记录文件")
		}
		return
	}

	var messages []wsMessage
	if err := json.Unmarshal(content, &messages); err != nil {
		sendErrorMessage(conn, "无法解析聊天记录")
		return
	}

	// 发送聊天记录响应
	response := map[string]interface{}{
		"type":     "chat_history",
		"id":       id,
		"messages": messages,
	}
	if err := conn.WriteJSON(response); err != nil {
		logs.Errorf("发送聊天记录失败: %v", err)
	}
}

