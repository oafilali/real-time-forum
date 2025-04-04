// chat.js - WebSocket functionality for private messaging

let socket = null;
let currentChatUser = null;
const onlineUsers = [];

// Connect to WebSocket when user is logged in
function connectWebSocket() {
  if (socket !== null) return;

  const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
  const wsUrl = `${protocol}//${window.location.host}/ws`;

  // Update status to connecting
  const statusElement = document.getElementById("chat-status");
  statusElement.textContent = "Connecting...";
  statusElement.className = "connecting";

  socket = new WebSocket(wsUrl);

  socket.onopen = function () {
    console.log("WebSocket connected");
    statusElement.textContent = "Connected";
    statusElement.className = "connected";
  };

  socket.onclose = function () {
    console.log("WebSocket disconnected");
    statusElement.textContent = "Disconnected";
    statusElement.className = "disconnected";
    socket = null;

    // Try to reconnect after a delay
    setTimeout(connectWebSocket, 5000);
  };

  socket.onerror = function (error) {
    console.error("WebSocket error:", error);
    statusElement.textContent = "Error";
    statusElement.className = "disconnected";
  };

  socket.onmessage = function (event) {
    const data = JSON.parse(event.data);

    if (data.type === "user_list") {
      // Update online users list
      updateOnlineUsers(data.users);
    } else if (data.type === "message") {
      // Handle received message
      displayMessage(data);

      // If this is a new message and not from current chat, show notification
      if (
        data.sender_id !== state.sessionID &&
        (!currentChatUser || data.sender_id !== currentChatUser.id)
      ) {
        showMessageNotification(data);
      }
    } else if (data.type === "history") {
      // Display message history
      displayMessageHistory(data.messages);
    }
  };
}

// Show notification for new message
function showMessageNotification(message) {
  // Find the username from online users
  let senderName = "User";
  const userItem = document.querySelector(
    `.user-item[data-user-id="${message.sender_id}"]`
  );
  if (userItem) {
    senderName = userItem.dataset.username;
  }

  // Create notification if browser supports it
  if ("Notification" in window) {
    if (Notification.permission === "granted") {
      new Notification(`New message from ${senderName}`, {
        body:
          message.content.substring(0, 50) +
          (message.content.length > 50 ? "..." : ""),
      });
    } else if (Notification.permission !== "denied") {
      Notification.requestPermission();
    }
  }

  // Also highlight the user in the list
  if (userItem) {
    userItem.classList.add("has-new-message");
  }
}

// Update the list of online users
function updateOnlineUsers(users) {
  const usersList = document.getElementById("online-users-list");
  usersList.innerHTML = "";

  if (users.length <= 1) {
    // Only current user is online
    const emptyMessage = document.createElement("p");
    emptyMessage.className = "empty-users-message";
    emptyMessage.textContent = "No users online";
    usersList.appendChild(emptyMessage);
    return;
  }

  users.forEach((user) => {
    // Don't show current user
    if (user.id === state.sessionID) return;

    const userItem = document.createElement("div");
    userItem.className = "user-item";
    userItem.textContent = user.username;
    userItem.dataset.userId = user.id;
    userItem.dataset.username = user.username;

    userItem.addEventListener("click", function () {
      openChat(user.id, user.username);
      // Remove highlight when clicked
      this.classList.remove("has-new-message");
    });

    usersList.appendChild(userItem);
  });
}

// Open chat with a specific user
function openChat(userId, username) {
  currentChatUser = { id: userId, name: username };

  // Update UI
  document.getElementById("chat-user-name").textContent = username;
  document.getElementById("chat-window").style.display = "flex";

  // Clear previous messages
  document.getElementById("messages-container").innerHTML = "";

  // Request message history
  if (socket && socket.readyState === WebSocket.OPEN) {
    socket.send(
      JSON.stringify({
        type: "get_history",
        receiverID: userId,
      })
    );
  }

  // Focus the message input
  setTimeout(() => {
    document.getElementById("message-input").focus();
  }, 100);
}

// Send a message
function sendMessage() {
  if (!socket || socket.readyState !== WebSocket.OPEN) {
    alert("WebSocket not connected");
    return;
  }

  if (!currentChatUser) {
    alert("Please select a user to chat with");
    return;
  }

  const messageInput = document.getElementById("message-input");
  const content = messageInput.value.trim();

  if (!content) return;

  const message = {
    type: "message",
    receiverID: currentChatUser.id,
    content: content,
  };

  socket.send(JSON.stringify(message));
  messageInput.value = "";
  messageInput.focus();
}

// Display a received message
function displayMessage(message) {
  // Only display if it's part of the current chat
  if (
    currentChatUser &&
    (message.sender_id === currentChatUser.id ||
      message.receiver_id === currentChatUser.id)
  ) {
    const messagesContainer = document.getElementById("messages-container");
    const messageElem = document.createElement("div");

    messageElem.className = "message";
    if (message.sender_id === state.sessionID) {
      messageElem.classList.add("outgoing");
    } else {
      messageElem.classList.add("incoming");
    }

    const time = new Date(message.timestamp).toLocaleTimeString();
    messageElem.innerHTML = `
      <div class="message-text">${escapeHTML(message.content)}</div>
      <div class="message-time">${time}</div>
    `;

    messagesContainer.appendChild(messageElem);
    messagesContainer.scrollTop = messagesContainer.scrollHeight;
  }
}

// Helper function to escape HTML special characters
function escapeHTML(text) {
  return text
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#039;");
}

// Display message history
function displayMessageHistory(messages) {
  const messagesContainer = document.getElementById("messages-container");
  messagesContainer.innerHTML = "";

  // Display newest messages last
  messages.reverse().forEach((message) => {
    displayMessage(message);
  });
}

// Toggle mobile chat sidebar
function toggleChatSidebar() {
  const sidebar = document.querySelector(".chat-sidebar");
  sidebar.classList.toggle("active");
}

// Initialize chat when DOM is loaded
document.addEventListener("DOMContentLoaded", function () {
  // Add chat toggle button for mobile
  if (window.innerWidth <= 768) {
    const toggleButton = document.createElement("button");
    toggleButton.className = "chat-toggle";
    toggleButton.innerHTML = "💬";
    toggleButton.addEventListener("click", toggleChatSidebar);
    document.body.appendChild(toggleButton);
  }

  // Set up message send button
  document
    .getElementById("send-message-button")
    .addEventListener("click", sendMessage);

  // Send on Enter key
  document
    .getElementById("message-input")
    .addEventListener("keydown", function (e) {
      if (e.key === "Enter" && !e.shiftKey) {
        e.preventDefault();
        sendMessage();
      }
    });

  // Close chat button
  document.getElementById("close-chat").addEventListener("click", function () {
    document.getElementById("chat-window").style.display = "none";
    currentChatUser = null;
  });

  // Connect WebSocket if logged in
  if (state && state.sessionID > 0) {
    connectWebSocket();
  }

  // Request notification permission
  if ("Notification" in window && Notification.permission === "default") {
    // Wait a moment before requesting permission to avoid overwhelming the user
    setTimeout(() => {
      Notification.requestPermission();
    }, 5000);
  }
});

// Connect when user logs in (event from app.js)
window.addEventListener("stateUpdated", function () {
  if (state && state.sessionID > 0 && !socket) {
    connectWebSocket();
  }
});

// Add an event to app.js to trigger state updates
function triggerStateUpdate() {
  const event = new Event("stateUpdated");
  window.dispatchEvent(event);
}

// Update the original updateUI function in app.js
const originalUpdateUI = updateUI;
window.updateUI = function () {
  originalUpdateUI();
  triggerStateUpdate();
};
