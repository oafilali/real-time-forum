// chat.js - WebSocket functionality for private messaging

let socket = null;
let currentChatUser = null;
const onlineUsers = [];

// Connect to WebSocket when user is logged in
function connectWebSocket() {
  if (socket !== null) return;

  const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
  const wsUrl = `${protocol}//${window.location.host}/ws`;

  socket = new WebSocket(wsUrl);

  socket.onopen = function () {
    console.log("WebSocket connected");
    document.getElementById("chat-status").textContent = "Connected";
  };

  socket.onclose = function () {
    console.log("WebSocket disconnected");
    document.getElementById("chat-status").textContent = "Disconnected";
    socket = null;

    // Try to reconnect after a delay
    setTimeout(connectWebSocket, 5000);
  };

  socket.onerror = function (error) {
    console.error("WebSocket error:", error);
  };

  socket.onmessage = function (event) {
    const data = JSON.parse(event.data);

    if (data.type === "user_list") {
      // Update online users list
      updateOnlineUsers(data.users);
    } else if (data.type === "message") {
      // Handle received message
      displayMessage(data);
    } else if (data.type === "history") {
      // Display message history
      displayMessageHistory(data.messages);
    }
  };
}

// Update the list of online users
function updateOnlineUsers(users) {
  const usersList = document.getElementById("online-users-list");
  usersList.innerHTML = "";

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
    });

    usersList.appendChild(userItem);
  });
}

// Open chat with a specific user
function openChat(userId, username) {
  currentChatUser = { id: userId, name: username };

  // Update UI
  document.getElementById("chat-user-name").textContent = username;
  document.getElementById("chat-window").style.display = "block";

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
            <div class="message-text">${message.content}</div>
            <div class="message-time">${time}</div>
        `;

    messagesContainer.appendChild(messageElem);
    messagesContainer.scrollTop = messagesContainer.scrollHeight;
  }
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

// Initialize chat when DOM is loaded
document.addEventListener("DOMContentLoaded", function () {
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
});

// Connect when user logs in (event from app.js)
window.addEventListener("stateUpdated", function () {
  if (state && state.sessionID > 0 && !socket) {
    connectWebSocket();
  }
});
