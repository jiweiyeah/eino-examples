document.addEventListener("DOMContentLoaded", () => {
  const chatForm = document.getElementById("chat-form");
  const userInput = document.getElementById("user-input");
  const chatContainer = document.getElementById("chat-container");
  const sendButton = document.getElementById("send-button");
  const newChatBtn = document.getElementById("new-chat-btn");
  const mobileNavBtn = document.getElementById("mobile-nav");
  const sidebar = document.getElementById("sidebar");
  const mainContent = document.getElementById("main-content");
  const historyList = document.createElement("div");
  historyList.className = "history-list";
  sidebar.insertBefore(historyList, sidebar.querySelector(".sidebar-footer"));

  // 存储聊天历史
  let chatMessages = [];
  let currentChatId = null;

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

  // 动态调整文本域高度
  userInput.addEventListener("input", () => {
    // 启用/禁用发送按钮
    sendButton.disabled = userInput.value.trim() === "";

    // 调整高度
    userInput.style.height = "24px";
    userInput.style.height = Math.min(userInput.scrollHeight, 200) + "px";
  });

  // 处理表单提交
  chatForm.addEventListener("submit", async (e) => {
    e.preventDefault();

    const message = userInput.value.trim();
    if (!message) return;

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
      // 发送消息到API
      await sendMessage(message, typingIndicator);
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

  // 发送消息到API并处理流式响应
  async function sendMessage(message, typingIndicator) {
    try {
      const response = await fetch("/api/chat", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ message }),
      });

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      // 处理流式响应
      const reader = response.body.getReader();
      const decoder = new TextDecoder();
      let aiResponse = "";

      // 移除输入指示器
      if (typingIndicator) {
        typingIndicator.remove();
      }

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

      const aiResponseElement = messageElement.querySelector(".ai-response");

      // 读取流式响应
      while (true) {
        const { done, value } = await reader.read();

        if (done) {
          break;
        }

        // 解码数据
        const chunk = decoder.decode(value, { stream: true });

        // 处理SSE格式的数据
        const lines = chunk.split("\n\n");
        for (const line of lines) {
          if (line.startsWith("data: ")) {
            const data = line.substring(6);
            aiResponse += data;
            aiResponseElement.innerHTML = formatMessage(aiResponse);
            scrollToBottom();
          }
        }
      }

      // 保存消息到历史记录
      chatMessages.push({ role: "assistant", content: aiResponse });
      await saveChat();
    } catch (error) {
      console.error("处理响应时出错:", error);
      throw error;
    }
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

  async function saveChat() {
    if (currentChatId === null) {
      currentChatId = Date.now().toString();
      const title = chatMessages[0].content.substring(0, 30);
      addHistoryItem({ id: currentChatId, title: title }, true);
    }

    try {
      await fetch("/api/history", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          id: currentChatId,
          messages: chatMessages,
        }),
      });
    } catch (error) {
      console.error("保存聊天记录失败:", error);
    }
  }

  async function loadHistory() {
    try {
      const response = await fetch("/api/history");
      const history = await response.json();
      historyList.innerHTML = "";
      history.forEach((item) => addHistoryItem(item));
    } catch (error) {
      console.error("加载历史记录失败:", error);
    }
  }

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

  async function loadChat(id) {
    try {
      const response = await fetch(`/api/history/${id}`);
      const messages = await response.json();

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
    } catch (error) {
      console.error("加载聊天记录失败:", error);
    }
  }

  // 初始加载
  loadHistory();
});
