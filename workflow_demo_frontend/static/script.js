document.addEventListener("DOMContentLoaded", () => {
  const chatForm = document.getElementById("chat-form");
  const userInput = document.getElementById("user-input");
  const chatContainer = document.getElementById("chat-container");
  const sendButton = document.getElementById("send-button");
  const newChatBtn = document.getElementById("new-chat-btn");
  const mobileNavBtn = document.getElementById("mobile-nav");
  const sidebar = document.getElementById("sidebar");
  const mainContent = document.getElementById("main-content");

  // 存储聊天历史
  const chatMessages = [];

  // 处理侧边栏切换
  mobileNavBtn.addEventListener("click", () => {
    sidebar.classList.toggle("expanded");
  });

  // 新对话按钮
  newChatBtn.addEventListener("click", () => {
    // 清空聊天历史
    while (chatContainer.children.length > 1) {
      chatContainer.removeChild(chatContainer.lastChild);
    }
    // 清空存储的消息
    chatMessages.length = 0;
    userInput.focus();
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

    // 保存消息到历史记录
    chatMessages.push({ role: "user", content: message });

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

    // 保存消息到历史记录
    chatMessages.push({ role: "assistant", content: message });

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
      "<pre><code>$1</code></pre>"
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

  // 初始聚焦到输入框
  userInput.focus();
});
