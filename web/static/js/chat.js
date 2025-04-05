// chat.js - WebSocket functionality for private messaging

let socket = null;
let currentChatUser = null;
const onlineUsers = [];
let allUsers = [];

// For debugging
const DEBUG = true;
function debug(message, data) {
  if (DEBUG) {
    console.log(`[CHAT DEBUG] ${message}`, data || "");
  }
}

// Fetch all registered users
async function fetchAllUsers() {
  try {
    debug("Attempting to fetch users from /user/all");
    const response = await fetch("/user/all", {
      headers: { Accept: "application/json" },
    });

    if (response.ok) {
      const data = await response.json();
      debug("Received user data:", data);
      allUsers = data.users || [];
      updateUsersList();
      return true;
    } else {
      debug("Failed to fetch users, status:", response.status);
      return false;
    }
  } catch (error) {
    debug("Error fetching users:", error);
    return false;
  }
}

// Connect to WebSocket when user is logged in
function connectWebSocket() {
  if (socket !== null) {
    debug("WebSocket already connected, skipping connection");
    return;
  }

  const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
  const wsUrl = `${protocol}//${window.location.host}/ws`;
  debug("Attempting WebSocket connection to:", wsUrl);

  // Update status to connecting
  const statusElement = document.getElementById("chat-status");
  if (statusElement) {
    statusElement.textContent = "Connecting...";
    statusElement.className = "connecting";
  }

  try {
    socket = new WebSocket(wsUrl);

    socket.onopen = function () {
      debug("WebSocket connected successfully");
      if (statusElement) {
        statusElement.textContent = "Connected";
        statusElement.className = "connected";
      }

      // Fetch users or show mock data
      fetchAllUsers().then((success) => {
        if (!success) {
          debug("Using mock user data instead");
          mockFetchAllUsers();
        }
      });
    };

    socket.onclose = function (event) {
      debug("WebSocket closed with code:", event.code);
      if (statusElement) {
        statusElement.textContent = "Disconnected";
        statusElement.className = "disconnected";
      }
      socket = null;

      // Try to reconnect after a delay
      setTimeout(connectWebSocket, 5000);
    };

    socket.onerror = function (error) {
      debug("WebSocket error:", error);
      if (statusElement) {
        statusElement.textContent = "Error";
        statusElement.className = "disconnected";
      }
    };

    socket.onmessage = function (event) {
      try {
        const data = JSON.parse(event.data);
        debug("Received WebSocket message:", data);

        if (data.type === "user_list") {
          // Update online users list
          onlineUsers.length = 0;
          data.users.forEach((user) => {
            onlineUsers.push(user.id);
          });
          updateUsersList();
        } else if (data.type === "message") {
          // Handle received message
          debug("Received chat message:", data);
          displayMessage(data);

          // If this is a new message and not from current chat, show notification
          if (
            data.sender_id !== state.sessionID &&
            (!currentChatUser || data.sender_id !== currentChatUser.id)
          ) {
            showMessageNotification(data);
          }
        } else if (data.type === "history") {
          debug("Received message history:", data);
          displayMessageHistory(data.messages);
        }
      } catch (e) {
        debug("Error processing WebSocket message:", e);
      }
    };
  } catch (e) {
    debug("Error creating WebSocket:", e);
    if (statusElement) {
      statusElement.textContent = "Failed";
      statusElement.className = "disconnected";
    }
  }
}

// Show notification for new message
function showMessageNotification(message) {
  // Find the username from users list
  let senderName = "User";
  const userItem = document.querySelector(
    `.user-item[data-user-id="${message.sender_id}"]`
  );
  if (userItem) {
    senderName = userItem.dataset.username;
  }

  debug("Showing notification for message from:", senderName);

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

// Update the list of users (both online and offline)
function updateUsersList() {
  const usersList = document.getElementById("users-list");
  if (!usersList) {
    debug("users-list element not found");
    return;
  }

  debug("Updating users list with:", { allUsers, onlineUsers });
  usersList.innerHTML = "";

  if (allUsers.length === 0) {
    const emptyMessage = document.createElement("p");
    emptyMessage.className = "empty-users-message";
    emptyMessage.textContent = "No users found";
    usersList.appendChild(emptyMessage);
    return;
  }

  allUsers.forEach((user) => {
    // Don't show current user
    if (user.id === state.sessionID) return;

    const userItem = document.createElement("div");
    userItem.className = "user-item";
    if (onlineUsers.includes(user.id)) {
      userItem.classList.add("online");
    } else {
      userItem.classList.add("offline");
    }

    userItem.textContent = user.username;
    userItem.dataset.userId = user.id;
    userItem.dataset.username = user.username;

    userItem.addEventListener("click", function () {
      debug("User clicked:", user);
      openChat(user.id, user.username);
      // Remove highlight when clicked
      this.classList.remove("has-new-message");
    });

    usersList.appendChild(userItem);
  });

  debug("Users list updated with", allUsers.length, "users");
}

// Open chat with a specific user - shows in main content area
function openChat(userId, username) {
  currentChatUser = { id: userId, name: username };
  debug("Opening chat with:", currentChatUser);

  // Update main content area with chat interface
  const content = document.getElementById("content");
  if (!content) {
    debug("Content element not found");
    return;
  }

  content.innerHTML = templates.chatInterface(username);

  // Set up back button
  document
    .getElementById("back-to-posts")
    .addEventListener("click", function () {
      debug("Back to posts clicked");
      loadHomePage();
      currentChatUser = null;
    });

  // Set up message send button
  document
    .getElementById("send-message-button")
    .addEventListener("click", sendMessage);

  // Set up enter key to send message
  document
    .getElementById("message-input")
    .addEventListener("keydown", function (e) {
      if (e.key === "Enter" && !e.shiftKey) {
        e.preventDefault();
        sendMessage();
      }
    });

  // Request message history
  if (socket && socket.readyState === WebSocket.OPEN) {
    debug("Requesting message history with:", userId);
    socket.send(
      JSON.stringify({
        type: "get_history",
        receiverID: userId,
      })
    );
  } else {
    debug("Socket not ready, cannot request history");
    const messagesContainer = document.getElementById("messages-container");
    if (messagesContainer) {
      messagesContainer.innerHTML = `
        <div class="chat-empty-state">
          <h3>Chat connection error</h3>
          <p>Not connected to chat server. Please refresh the page and try again.</p>
        </div>
      `;
    }
  }

  // Focus the message input
  setTimeout(() => {
    const messageInput = document.getElementById("message-input");
    if (messageInput) messageInput.focus();
  }, 100);
}

// Send a message
function sendMessage() {
  if (!socket || socket.readyState !== WebSocket.OPEN) {
    alert("WebSocket not connected");
    debug("Cannot send message, WebSocket not connected");
    return;
  }

  if (!currentChatUser) {
    alert("Please select a user to chat with");
    debug("Cannot send message, no chat user selected");
    return;
  }

  const messageInput = document.getElementById("message-input");
  if (!messageInput) {
    debug("Message input element not found");
    return;
  }

  const content = messageInput.value.trim();

  if (!content) {
    debug("Empty message, not sending");
    return;
  }

  const message = {
    type: "message",
    receiverID: currentChatUser.id,
    content: content,
  };

  // Log to console for debugging
  debug("Sending message:", message);

  // Add message locally immediately
  displayLocalMessage(content);

  // Send via websocket
  try {
    socket.send(JSON.stringify(message));
    messageInput.value = "";
    messageInput.focus();
  } catch (e) {
    debug("Error sending message:", e);
    alert("Failed to send message: " + e.message);
  }
}

// Display a locally sent message before confirmation
function displayLocalMessage(content) {
  const messagesContainer = document.getElementById("messages-container");
  if (!messagesContainer) {
    debug("Messages container not found");
    return;
  }

  // Clear empty state if it exists
  const emptyState = messagesContainer.querySelector(".chat-empty-state");
  if (emptyState) {
    emptyState.remove();
  }

  const messageElem = document.createElement("div");
  messageElem.className = "message outgoing";

  const time = new Date().toLocaleTimeString();
  messageElem.innerHTML = `
    <div class="message-text">${escapeHTML(content)}</div>
    <div class="message-time">${time} (Sending...)</div>
  `;

  messagesContainer.appendChild(messageElem);
  messagesContainer.scrollTop = messagesContainer.scrollHeight;
  debug("Local message displayed");
}

// Display a received message
function displayMessage(message) {
  debug("Displaying message:", message);
  // Only display if it's part of the current chat
  if (
    currentChatUser &&
    (message.sender_id === currentChatUser.id ||
      message.receiver_id === currentChatUser.id)
  ) {
    const messagesContainer = document.getElementById("messages-container");
    if (!messagesContainer) {
      debug("Messages container not found");
      return;
    }

    // Clear empty state if it exists
    const emptyState = messagesContainer.querySelector(".chat-empty-state");
    if (emptyState) {
      emptyState.remove();
    }

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
    debug("Message displayed in chat");
  } else {
    debug("Message not displayed (not for current chat)");
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
  if (!messagesContainer) {
    debug("Messages container not found for history");
    return;
  }

  messagesContainer.innerHTML = "";

  if (!messages || messages.length === 0) {
    debug("No message history to display");
    messagesContainer.innerHTML = `
      <div class="chat-empty-state">
        <h3>Start a conversation</h3>
        <p>No messages yet. Send a message to start the conversation.</p>
      </div>
    `;
    return;
  }

  debug("Displaying message history:", messages.length, "messages");
  // Display newest messages last
  messages.reverse().forEach((message) => {
    displayMessage(message);
  });
}

// Initialize chat when DOM is loaded
document.addEventListener("DOMContentLoaded", function () {
  debug("DOM loaded, checking session");
  // Connect WebSocket if logged in
  if (state && state.sessionID > 0) {
    debug("Session active, connecting WebSocket");
    connectWebSocket();
  } else {
    debug("Not logged in, skipping WebSocket connection");
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
  debug("State updated event received", { sessionID: state.sessionID });
  if (state && state.sessionID > 0 && !socket) {
    debug("User logged in, connecting WebSocket");
    connectWebSocket();
  }
});

// Add an event to app.js to trigger state updates
function triggerStateUpdate() {
  debug("Triggering state update event");
  const event = new Event("stateUpdated");
  window.dispatchEvent(event);
}

// Mock user data until backend API is available
function mockFetchAllUsers() {
  debug("Using mock user data");
  if (state && state.sessionID > 0) {
    // Create some dummy users
    allUsers = [
      { id: 1001, username: "Alice" },
      { id: 1002, username: "Bob" },
      { id: 1003, username: "Charlie" },
      { id: 1004, username: "David" },
      { id: 1005, username: "Eve" },
    ];

    // Randomly set some as online
    onlineUsers.length = 0;
    allUsers.forEach((user) => {
      if (Math.random() > 0.5) {
        onlineUsers.push(user.id);
      }
    });

    // Update the UI
    updateUsersList();
    debug("Mock users data applied");
  }
}

// Add loadHomePage reference from app.js
window.loadHomePage = loadHomePage;

// Update the original updateUI function in app.js to connect WebSocket on login
const originalUpdateUI = updateUI;
window.updateUI = function () {
  originalUpdateUI();
  debug("UI updated, triggering state update");
  triggerStateUpdate();

  // If logged in, attempt to fetch users (or use mock data)
  if (state && state.sessionID > 0) {
    debug("User logged in, fetching users");
    // Try to fetch real users first
    fetchAllUsers().catch(() => {
      debug("Failed to fetch users, using mock data");
      // Fall back to mock data if API doesn't exist yet
      setTimeout(mockFetchAllUsers, 1000);
    });
  }
};
