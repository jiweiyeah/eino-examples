document.addEventListener("DOMContentLoaded", () => {
  const chatForm = document.getElementById("chat-form");
  const userInput = document.getElementById("user-input");
  const chatContainer = document.getElementById("chat-container");
  const sendButton = document.getElementById("send-button");
  const newChatBtn = document.getElementById("new-chat-btn");
  const mobileNavBtn = document.getElementById("mobile-nav");
  const sidebar = document.getElementById("sidebar");
  const mainContent = document.getElementById("main-content");
  const connectWsBtn = document.getElementById("connect-ws-btn");
  const disconnectWsBtn = document.getElementById("disconnect-ws-btn");
  const connectionStatus = document.getElementById("connection-status");
  const historyList = document.createElement("div");
  historyList.className = "history-list";
  sidebar.insertBefore(historyList, sidebar.querySelector(".sidebar-footer"));

  // 存储聊天历史
  let chatMessages = [];
  let currentChatId = null;
  let socket = null;
  let isConnected = false;
  let currentAiResponseElement = null;

  // 处理侧边栏切换
  mobileNavBtn.addEventListener("click", () => {
    sidebar.classList.toggle("expanded");
  });

  // 新对话按钮
  newChatBtn.addEventListener("click", () => {
    clearChat();
    currentChatId = null;
    chatMessages = [];
    userInput.focus();
    // 取消历史记录的选中状态
    const activeItem = historyList.querySelector(".history-item.active");
    if (activeItem) {
      activeItem.classList.remove("active");
    }
  });

  // 连接WebSocket按钮
  connectWsBtn.addEventListener("click", connectWebSocket);

  // 断开WebSocket按钮
  disconnectWsBtn.addEventListener("click", disconnectWebSocket);

  // 动态调整文本域高度
  userInput.addEventListener("input", () => {
    // 启用/禁用发送按钮
    sendButton.disabled = userInput.value.trim() === "" || !isConnected;

    // 调整高度
    userInput.style.height = "24px";
    userInput.style.height = Math.min(userInput.scrollHeight, 200) + "px";
  });

  // 处理表单提交
  chatForm.addEventListener("submit", async (e) => {
    e.preventDefault();

    const message = userInput.value.trim();
    if (!message || !isConnected) return;

    // 清空输入框并重置高度
    userInput.value = "";
    userInput.style.height = "24px";
    sendButton.disabled = true;

    // 添加用户消息到聊天界面
    addUserMessage(message);
    chatMessages.push({ role: "user", content: message });

    // 显示AI正在输入的指示器
    const typingIndicator = addTypingIndicator();

    try {
      // 发送消息到WebSocket
      sendChatMessage(message, typingIndicator);
    } catch (error) {
      console.error("发送消息时出错:", error);
      // 移除输入指示器
      if (typingIndicator) {
        typingIndicator.remove();
      }
      // 显示错误消息
      addSystemMessage("抱歉，发生了错误。请稍后再试。");
    }
  });

  function clearChat() {
    // 清空聊天历史
    while (chatContainer.children.length > 1) {
      chatContainer.removeChild(chatContainer.lastChild);
    }
  }

  // 添加用户消息到聊天界面
  function addUserMessage(message) {
    const messageElement = document.createElement("div");
    messageElement.className = "chat-message user-message";
    messageElement.innerHTML = `
      <div class="message-content">
        <div class="avatar user-avatar">
          <i class="fas fa-user"></i>
        </div>
        <div>${formatMessage(message)}</div>
      </div>
    `;

    chatContainer.appendChild(messageElement);

    // 滚动到底部
    scrollToBottom();
  }

  // 添加AI消息到聊天界面
  function addAIMessage(message) {
    const messageElement = document.createElement("div");
    messageElement.className = "chat-message ai-message";
    messageElement.innerHTML = `
      <div class="message-content">
        <div class="avatar ai-avatar">
          <i class="fas fa-robot"></i>
        </div>
        <div class="whitespace-pre-wrap">${formatMessage(message)}</div>
      </div>
    `;

    chatContainer.appendChild(messageElement);

    // 滚动到底部
    scrollToBottom();
  }

  // 添加系统消息到聊天界面
  function addSystemMessage(message) {
    const messageElement = document.createElement("div");
    messageElement.className = "chat-message ai-message";
    messageElement.innerHTML = `
      <div class="message-content">
        <div class="avatar" style="background-color: #f97316;">
          <i class="fas fa-info-circle"></i>
        </div>
        <div>${formatMessage(message)}</div>
      </div>
    `;

    chatContainer.appendChild(messageElement);

    // 滚动到底部
    scrollToBottom();
  }

  // 添加"正在输入"指示器
  function addTypingIndicator() {
    const indicatorElement = document.createElement("div");
    indicatorElement.className =
      "chat-message ai-message typing-indicator-container";
    indicatorElement.innerHTML = `
      <div class="message-content">
        <div class="avatar ai-avatar">
          <i class="fas fa-robot"></i>
        </div>
        <div class="typing-indicator">正在思考</div>
      </div>
    `;

    chatContainer.appendChild(indicatorElement);

    // 滚动到底部
    scrollToBottom();

    return indicatorElement;
  }

  // 连接WebSocket
  function connectWebSocket() {
    if (socket !== null) {
      return; // 已经连接
    }

    // 获取当前主机
    const host = window.location.host;
    const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
    const wsUrl = `${protocol}//${host}/ws`;

    try {
      connectionStatus.textContent = "正在连接...";
      socket = new WebSocket(wsUrl);

      // 连接打开事件
      socket.onopen = () => {
        isConnected = true;
        connectionStatus.textContent = "已连接";
        connectionStatus.classList.add("connected");
        connectWsBtn.disabled = true;
        disconnectWsBtn.disabled = false;
        userInput.disabled = false;
        sendButton.disabled = userInput.value.trim() === "";
        addSystemMessage("WebSocket连接已建立。您现在可以开始聊天了。");

        // 连接后加载历史记录
        requestChatHistory();
      };

      // 接收消息事件
      socket.onmessage = (event) => {
        const data = JSON.parse(event.data);
        handleWebSocketMessage(data);
      };

      // 连接关闭事件
      socket.onclose = () => {
        handleDisconnect();
      };

      // 连接错误事件
      socket.onerror = (error) => {
        console.error("WebSocket错误:", error);
        connectionStatus.textContent = "连接错误";
        connectionStatus.classList.remove("connected");
        connectionStatus.classList.add("error");
        addSystemMessage("WebSocket连接出错。请尝试重新连接。");
        handleDisconnect();
      };
    } catch (error) {
      console.error("创建WebSocket连接时出错:", error);
      connectionStatus.textContent = "连接失败";
      connectionStatus.classList.add("error");
      addSystemMessage("无法创建WebSocket连接。请稍后再试。");
    }
  }

  // 断开WebSocket连接
  function disconnectWebSocket() {
    if (socket === null) {
      return; // 已经断开连接
    }

    try {
      socket.close();
    } catch (error) {
      console.error("关闭WebSocket连接时出错:", error);
    }

    handleDisconnect();
    addSystemMessage("WebSocket连接已断开。");
  }

  // 处理断开连接
  function handleDisconnect() {
    isConnected = false;
    socket = null;
    connectionStatus.textContent = "未连接";
    connectionStatus.classList.remove("connected", "error");
    connectWsBtn.disabled = false;
    disconnectWsBtn.disabled = true;
    userInput.disabled = true;
    sendButton.disabled = true;
  }

  // 处理WebSocket消息
  function handleWebSocketMessage(data) {
    switch (data.type) {
      case "chat_chunk":
        handleChatChunk(data.content);
        break;
      case "chat_end":
        finalizeChatResponse();
        break;
      case "error":
        addSystemMessage(`错误: ${data.message}`);
        break;
      case "history_list":
        displayHistoryList(data.history);
        break;
      case "chat_history":
        loadChatData(data.id, data.messages);
        break;
      case "save_success":
        console.log("聊天记录保存成功");
        break;
      default:
        console.warn("收到未知类型的WebSocket消息:", data);
    }
  }

  // 处理聊天块
  function handleChatChunk(content) {
    if (!currentAiResponseElement) {
      // 创建新的AI消息元素
      const messageElement = document.createElement("div");
      messageElement.className = "chat-message ai-message";
      messageElement.innerHTML = `
        <div class="message-content">
          <div class="avatar ai-avatar">
            <i class="fas fa-robot"></i>
          </div>
          <div class="ai-response whitespace-pre-wrap"></div>
        </div>
      `;

      chatContainer.appendChild(messageElement);
      currentAiResponseElement = {
        element: messageElement,
        responseDiv: messageElement.querySelector(".ai-response"),
        content: "",
      };
    }

    // 更新内容
    currentAiResponseElement.content += content;
    currentAiResponseElement.responseDiv.innerHTML = formatMessage(
      currentAiResponseElement.content
    );
    scrollToBottom();
  }

  // 完成聊天响应
  function finalizeChatResponse() {
    if (currentAiResponseElement) {
      // 保存消息到历史记录
      chatMessages.push({
        role: "assistant",
        content: currentAiResponseElement.content,
      });
      currentAiResponseElement = null;
      saveChat();
    }
  }

  // 发送聊天消息
  function sendChatMessage(message, typingIndicator) {
    if (!socket || socket.readyState !== WebSocket.OPEN) {
      if (typingIndicator) {
        typingIndicator.remove();
      }
      addSystemMessage("WebSocket未连接，无法发送消息。");
      return;
    }

    // 移除输入指示器
    if (typingIndicator) {
      typingIndicator.remove();
    }

    const chatCommand = {
      type: "chat",
      payload: {
        message: message,
      },
    };

    socket.send(JSON.stringify(chatCommand));
  }

  // 请求聊天历史列表
  function requestChatHistory() {
    if (!socket || socket.readyState !== WebSocket.OPEN) {
      return;
    }

    const historyCommand = {
      type: "history",
    };

    socket.send(JSON.stringify(historyCommand));
  }

  // 显示历史记录列表
  function displayHistoryList(history) {
    historyList.innerHTML = "";
    history.forEach((item) => addHistoryItem(item));
  }

  // 添加历史记录项
  function addHistoryItem(item, isActive = false) {
    const historyItem = document.createElement("div");
    historyItem.className = "history-item";
    historyItem.textContent = item.title;
    historyItem.dataset.id = item.id;

    if (isActive) {
      historyItem.classList.add("active");
    }

    historyItem.addEventListener("click", () => {
      loadChat(item.id);
    });

    historyList.prepend(historyItem);
    if (isActive) {
      const activeItem = historyList.querySelector(".history-item.active");
      if (activeItem) {
        activeItem.classList.remove("active");
      }
      historyItem.classList.add("active");
    }
  }

  // 加载聊天记录
  function loadChat(id) {
    if (!socket || socket.readyState !== WebSocket.OPEN) {
      addSystemMessage("WebSocket未连接，无法加载聊天记录。");
      return;
    }

    const loadCommand = {
      type: "load",
      payload: {
        id: id,
      },
    };

    socket.send(JSON.stringify(loadCommand));
  }

  // 加载聊天数据
  function loadChatData(id, messages) {
    clearChat();
    chatMessages = messages;
    currentChatId = id;

    messages.forEach((msg) => {
      if (msg.role === "user") {
        addUserMessage(msg.content);
      } else {
        addAIMessage(msg.content);
      }
    });

    // 设置当前激活的历史记录项
    const activeItem = historyList.querySelector(".history-item.active");
    if (activeItem) {
      activeItem.classList.remove("active");
    }
    const newItem = historyList.querySelector(`[data-id="${id}"]`);
    if (newItem) {
      newItem.classList.add("active");
    }
  }

  // 保存聊天记录
  function saveChat() {
    if (!socket || socket.readyState !== WebSocket.OPEN) {
      console.error("WebSocket未连接，无法保存聊天记录。");
      return;
    }

    if (currentChatId === null) {
      currentChatId = Date.now().toString();
      const title = chatMessages[0].content.substring(0, 30);
      addHistoryItem({ id: currentChatId, title: title }, true);
    }

    const saveCommand = {
      type: "save",
      payload: {
        id: currentChatId,
        messages: chatMessages,
      },
    };

    socket.send(JSON.stringify(saveCommand));
  }

  // 滚动聊天历史到底部
  function scrollToBottom() {
    chatContainer.scrollTop = chatContainer.scrollHeight;
  }

  // 格式化消息内容，处理换行、链接和代码块
  function formatMessage(text) {
    if (!text) return "";

    // 转义HTML
    let formatted = escapeHtml(text);

    // 将URL转换为链接
    formatted = formatted.replace(
      /(https?:\/\/[^\s]+)/g,
      '<a href="$1" target="_blank" class="text-blue-600 hover:underline">$1</a>'
    );

    // 处理换行符
    formatted = formatted.replace(/\n/g, "<br>");

    // 处理代码块 (```code```)
    formatted = formatted.replace(
      /```([\s\S]*?)```/g,
      '<pre class="code-block"><code>$1</code></pre>'
    );

    // 处理内联代码 (`code`)
    formatted = formatted.replace(
      /`([^`]+)`/g,
      '<code style="background-color: rgba(0,0,0,0.05); padding: 2px 4px; border-radius: 3px;">$1</code>'
    );

    return formatted;
  }

  // 转义HTML字符
  function escapeHtml(unsafe) {
    return unsafe
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;")
      .replace(/"/g, "&quot;")
      .replace(/'/g, "&#039;");
  }

  // 处理Enter键发送消息，Shift+Enter换行
  userInput.addEventListener("keydown", (e) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      if (!sendButton.disabled) {
        chatForm.dispatchEvent(new Event("submit"));
      }
    }
  });

  // 滚动时隐藏/显示侧边栏
  mainContent.addEventListener("scroll", () => {
    if (mainContent.scrollTop > 50) {
      mobileNavBtn.classList.add("scrolled");
    } else {
      mobileNavBtn.classList.remove("scrolled");
    }
  });
});
