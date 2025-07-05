document.addEventListener("DOMContentLoaded", () => {
  const chatForm = document.getElementById("chat-form");
  const userInput = document.getElementById("user-input");
  const chatHistory = document.getElementById("chat-history");

  // 存储聊天历史
  const chatMessages = [];

  // 处理表单提交
  chatForm.addEventListener("submit", async (e) => {
    e.preventDefault();

    const message = userInput.value.trim();
    if (!message) return;

    // 清空输入框
    userInput.value = "";

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
      typingIndicator.remove();
      // 显示错误消息
      addSystemMessage("抱歉，发生了错误。请稍后再试。");
    }
  });

  // 添加用户消息到聊天界面
  function addUserMessage(message) {
    const messageElement = document.createElement("div");
    messageElement.className = "chat-message flex justify-end mb-4";
    messageElement.innerHTML = `
            <div class="mr-3 bg-blue-500 p-3 rounded-lg text-white max-w-[80%]">
                <div class="text-sm text-blue-200 mb-1 text-right">您</div>
                <div>${escapeHtml(message)}</div>
            </div>
            <div class="flex-shrink-0 bg-blue-600 h-10 w-10 rounded-full flex items-center justify-center text-white">
                <i class="fas fa-user"></i>
            </div>
        `;

    const messagesContainer = chatHistory.querySelector(".space-y-4");
    messagesContainer.appendChild(messageElement);

    // 保存消息到历史记录
    chatMessages.push({ role: "user", content: message });

    // 滚动到底部
    scrollToBottom();
  }

  // 添加AI消息到聊天界面
  function addAIMessage(message) {
    const messageElement = document.createElement("div");
    messageElement.className = "chat-message flex mb-4";
    messageElement.innerHTML = `
            <div class="flex-shrink-0 bg-blue-500 h-10 w-10 rounded-full flex items-center justify-center text-white">
                <i class="fas fa-robot"></i>
            </div>
            <div class="ml-3 bg-blue-100 p-3 rounded-lg max-w-[80%]">
                <div class="text-sm text-gray-500 mb-1">AI</div>
                <div class="text-gray-800 whitespace-pre-wrap">${escapeHtml(
                  message
                )}</div>
            </div>
        `;

    const messagesContainer = chatHistory.querySelector(".space-y-4");
    messagesContainer.appendChild(messageElement);

    // 保存消息到历史记录
    chatMessages.push({ role: "assistant", content: message });

    // 滚动到底部
    scrollToBottom();
  }

  // 添加系统消息到聊天界面
  function addSystemMessage(message) {
    const messageElement = document.createElement("div");
    messageElement.className = "chat-message flex mb-4";
    messageElement.innerHTML = `
            <div class="flex-shrink-0 bg-gray-500 h-10 w-10 rounded-full flex items-center justify-center text-white">
                <i class="fas fa-info-circle"></i>
            </div>
            <div class="ml-3 bg-gray-100 p-3 rounded-lg max-w-[80%]">
                <div class="text-sm text-gray-500 mb-1">系统</div>
                <div class="text-gray-800">${escapeHtml(message)}</div>
            </div>
        `;

    const messagesContainer = chatHistory.querySelector(".space-y-4");
    messagesContainer.appendChild(messageElement);

    // 滚动到底部
    scrollToBottom();
  }

  // 添加"正在输入"指示器
  function addTypingIndicator() {
    const indicatorElement = document.createElement("div");
    indicatorElement.className =
      "chat-message flex mb-4 typing-indicator-container";
    indicatorElement.innerHTML = `
            <div class="flex-shrink-0 bg-blue-500 h-10 w-10 rounded-full flex items-center justify-center text-white">
                <i class="fas fa-robot"></i>
            </div>
            <div class="ml-3 bg-blue-100 p-3 rounded-lg">
                <div class="text-sm text-gray-500 mb-1">AI</div>
                <div class="text-gray-800 typing-indicator">正在思考</div>
            </div>
        `;

    const messagesContainer = chatHistory.querySelector(".space-y-4");
    messagesContainer.appendChild(indicatorElement);

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
      typingIndicator.remove();

      // 创建新的AI消息元素
      const messageElement = document.createElement("div");
      messageElement.className = "chat-message flex mb-4";
      messageElement.innerHTML = `
                <div class="flex-shrink-0 bg-blue-500 h-10 w-10 rounded-full flex items-center justify-center text-white">
                    <i class="fas fa-robot"></i>
                </div>
                <div class="ml-3 bg-blue-100 p-3 rounded-lg max-w-[80%]">
                    <div class="text-sm text-gray-500 mb-1">AI</div>
                    <div class="text-gray-800 whitespace-pre-wrap ai-response"></div>
                </div>
            `;

      const messagesContainer = chatHistory.querySelector(".space-y-4");
      messagesContainer.appendChild(messageElement);

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
            aiResponseElement.textContent = escapeHtml(aiResponse);
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
    chatHistory.scrollTop = chatHistory.scrollHeight;
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

  // 初始聚焦到输入框
  userInput.focus();
});
