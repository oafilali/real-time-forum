// chat.js - WebSocket functionality for private messaging

// Global variables for websocket messaging
let socket = null;
let currentChatUser = null;
const onlineUsers = [];
let allUsers = [];
let isLoading = false;
let lastMessagesData = {};

// Reconnection variables
let reconnectAttempts = 0;
const maxReconnectAttempts = 5;
const reconnectDelay = 1000; // 1 second delay

// Fetch all registered users
async function fetchAllUsers() {
  try {
    // Check if user is logged in
    if (!window.state || !window.state.sessionID) {
      return false;
    }

    const response = await fetch("/user/all", {
      headers: { Accept: "application/json" },
    });

    if (response.ok) {
      const data = await response.json();
      allUsers = data.users || [];
      updateUsersList();
      return true;
    } else {
      // If unauthorized, check login status
      if (response.status === 401 && typeof checkLogin === "function") {
        checkLogin();
      }
      return false;
    }
  } catch (error) {
    console.log("Error fetching users:", error);

    // Show error in users list
    const usersList = document.getElementById("users-list");
    if (usersList) {
      usersList.innerHTML =
        '<p class="empty-users-message">Error loading users</p>';
    }

    return false;
  }
}

// Connect to WebSocket when user is logged in
function connectWebSocket() {
  // Only connect if user is logged in and no active connection exists
  if (
    !window.state ||
    !window.state.sessionID ||
    (socket !== null && socket.readyState === WebSocket.OPEN)
  ) {
    return;
  }

  // Stop trying if max reconnect attempts reached
  if (reconnectAttempts >= maxReconnectAttempts) {
    const statusElement = document.getElementById("chat-status");
    if (statusElement) {
      statusElement.textContent = "Disconnected";
      statusElement.className = "disconnected";
    }
    return;
  }

  const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
  const wsUrl = `${protocol}//${window.location.host}/ws`;

  // Update connection status indicator
  const statusElement = document.getElementById("chat-status");
  if (statusElement) {
    statusElement.textContent = "Connecting...";
    statusElement.className = "connecting";
  }

  try {
    socket = new WebSocket(wsUrl);

    socket.onopen = function () {
      if (statusElement) {
        statusElement.textContent = "Connected";
        statusElement.className = "connected";
      }
      reconnectAttempts = 0; // Reset attempts on successful connection
      fetchAllUsers();
    };

    socket.onclose = function () {
      if (statusElement) {
        statusElement.textContent = "Disconnected";
        statusElement.className = "disconnected";
      }
      socket = null;

      // Try to reconnect if user is still logged in
      if (window.state && window.state.sessionID > 0) {
        reconnectAttempts++;
        const delay = reconnectDelay * Math.pow(2, reconnectAttempts - 1);
        setTimeout(connectWebSocket, delay);
      }
    };

    socket.onerror = function () {
      if (statusElement) {
        statusElement.textContent = "Error";
        statusElement.className = "disconnected";
      }
    };

    socket.onmessage = function (event) {
      try {
        const data = JSON.parse(event.data);

        if (data.type === "user_list") {
          handleUserList(data.users);
        } else if (data.type === "message") {
          handleMessage(data);
        } else if (data.type === "history") {
          if (Array.isArray(data.messages)) {
            displayMessageHistory(data.messages);
          } else {
            displayMessageHistory([]);
          }
        } else if (data.type === "more_history") {
          if (Array.isArray(data.messages)) {
            displayMoreMessageHistory(data.messages);
          }
        }
      } catch (e) {
        console.log("Error processing WebSocket message:", e);
      }
    };
  } catch (e) {
    console.log("Error creating WebSocket:", e);
    if (statusElement) {
      statusElement.textContent = "Failed";
      statusElement.className = "disconnected";
    }

    // Try to reconnect if user is still logged in
    if (window.state && window.state.sessionID > 0) {
      reconnectAttempts++;
      const delay = reconnectDelay * Math.pow(2, reconnectAttempts - 1);
      setTimeout(connectWebSocket, delay);
    }
  }
}

// Handle user list updates from the server
function handleUserList(users) {
  onlineUsers.length = 0;

  if (Array.isArray(users)) {
    users.forEach((user) => {
      onlineUsers.push(user.id);

      // Store last messages data if available
      if (user.lastMessages) {
        lastMessagesData[user.id] = user.lastMessages;
      }
    });
  }

  // Update the UI immediately
  updateUsersList();
}

// Handle incoming messages
function handleMessage(message) {
  displayMessage(message);

  // Show notification if message is not from current chat
  if (
    message.sender_id !== state.sessionID &&
    (!currentChatUser || message.sender_id !== currentChatUser.id)
  ) {
    showMessageNotification(message);
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

  // Highlight the user in the list
  if (userItem) {
    userItem.classList.add("has-new-message");
  }
}

// Update the list of users
function updateUsersList() {
  const usersList = document.getElementById("users-list");
  if (!usersList) {
    return;
  }

  usersList.innerHTML = "";

  if (allUsers.length === 0) {
    const emptyMessage = document.createElement("p");
    emptyMessage.className = "empty-users-message";
    emptyMessage.textContent = "No users found";
    usersList.appendChild(emptyMessage);
    return;
  }

  // Get latest message timestamp for a user
  const getLatestMessageTime = (userId) => {
    if (!lastMessagesData[userId]) return null;

    let latestTime = null;
    Object.values(lastMessagesData[userId]).forEach((msg) => {
      const msgTime = new Date(msg.timestamp).getTime();
      if (!latestTime || msgTime > latestTime) {
        latestTime = msgTime;
      }
    });
    return latestTime;
  };

  // Sort users:
  // 1. Online users first
  // 2. Users with recent messages next (sorted by timestamp)
  // 3. Then alphabetically
  const sortedUsers = [...allUsers].sort((a, b) => {
    const aOnline = onlineUsers.includes(a.id);
    const bOnline = onlineUsers.includes(b.id);

    // Online users first
    if (aOnline && !bOnline) return -1;
    if (!aOnline && bOnline) return 1;

    // Users with messages sorted by most recent
    const aLastMsg = getLatestMessageTime(a.id);
    const bLastMsg = getLatestMessageTime(b.id);

    if (aLastMsg && bLastMsg) {
      return bLastMsg - aLastMsg; // Most recent first
    }

    // Users with messages before those without
    if (aLastMsg && !bLastMsg) return -1;
    if (!aLastMsg && bLastMsg) return 1;

    // Alphabetical order for users without messages
    return a.username.localeCompare(b.username);
  });

  // Render the sorted list
  sortedUsers.forEach((user) => {
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
      openChat(user.id, user.username);
      // Remove highlight when clicked
      this.classList.remove("has-new-message");
    });

    usersList.appendChild(userItem);
  });
}

// Open chat with a specific user
function openChat(userId, username) {
  // Ensure userId is an integer
  userId = parseInt(userId, 10);
  currentChatUser = { id: userId, name: username };

  // Update main content area with chat interface
  const content = document.getElementById("content");
  if (!content) {
    return;
  }

  content.innerHTML = templates.chatInterface(username);

  // Set up back button
  document
    .getElementById("back-to-posts")
    .addEventListener("click", function () {
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
    socket.send(
      JSON.stringify({
        type: "get_history",
        receiverID: userId,
      })
    );
  } else {
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

  // Set up scroll listener for loading more messages
  setTimeout(() => {
    setupScrollListener();
  }, 500);

  // Focus the message input
  setTimeout(() => {
    const messageInput = document.getElementById("message-input");
    if (messageInput) messageInput.focus();
  }, 100);
}

// Display a locally sent message before confirmation
function displayLocalMessage(content) {
  const messagesContainer = document.getElementById("messages-container");
  if (!messagesContainer) {
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
}

// Display a received message
function displayMessage(message) {
  // Only display if it's part of the current chat
  if (
    currentChatUser &&
    ((message.sender_id === currentChatUser.id &&
      message.receiverID === state.sessionID) ||
      (message.sender_id === state.sessionID &&
        message.receiverID === currentChatUser.id))
  ) {
    const messagesContainer = document.getElementById("messages-container");
    if (!messagesContainer) {
      return;
    }

    // Clear empty state if it exists
    const emptyState = messagesContainer.querySelector(".chat-empty-state");
    if (emptyState) {
      emptyState.remove();
    }

    // Find existing "sending..." message if this is a confirmation of our sent message
    if (message.sender_id === state.sessionID) {
      const pendingMessages =
        messagesContainer.querySelectorAll(".message.outgoing");
      for (const pending of pendingMessages) {
        const timeElem = pending.querySelector(".message-time");
        if (timeElem && timeElem.textContent.includes("Sending...")) {
          // Update this message instead of creating a new one
          timeElem.textContent = new Date(
            message.timestamp
          ).toLocaleTimeString();
          // Add timestamp as data attribute
          timeElem.setAttribute("data-timestamp", message.timestamp);

          // If we have new message data, update the sorting of users
          if (!lastMessagesData[message.receiverID]) {
            lastMessagesData[message.receiverID] = {};
          }
          lastMessagesData[message.receiverID][message.id || Date.now()] = {
            timestamp: message.timestamp,
            content: message.content,
            sender_id: message.sender_id,
          };
          updateUsersList();

          return;
        }
      }
    }

    // Create new message element
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
            <div class="message-time" data-timestamp="${
              message.timestamp
            }">${time}</div>
        `;

    messagesContainer.appendChild(messageElem);
    messagesContainer.scrollTop = messagesContainer.scrollHeight;

    // Update last messages data for sorting
    const otherUser =
      message.sender_id === state.sessionID
        ? message.receiverID
        : message.sender_id;
    if (!lastMessagesData[otherUser]) {
      lastMessagesData[otherUser] = {};
    }
    lastMessagesData[otherUser][message.id || Date.now()] = {
      timestamp: message.timestamp,
      content: message.content,
      sender_id: message.sender_id,
    };
    updateUsersList();
  } else {
    // Still update the last messages data for user sorting
    if (message.sender_id !== state.sessionID) {
      const senderID = message.sender_id;
      if (!lastMessagesData[senderID]) {
        lastMessagesData[senderID] = {};
      }
      lastMessagesData[senderID][message.id || Date.now()] = {
        timestamp: message.timestamp,
        content: message.content,
        sender_id: message.sender_id,
      };
      updateUsersList();
    }
  }
}

// Implement throttle function for scroll events
function throttle(func, limit) {
  let lastFunc;
  let lastRan;
  return function () {
    const context = this;
    const args = arguments;
    if (!lastRan) {
      func.apply(context, args);
      lastRan = Date.now();
    } else {
      clearTimeout(lastFunc);
      lastFunc = setTimeout(function () {
        if (Date.now() - lastRan >= limit) {
          func.apply(context, args);
          lastRan = Date.now();
        }
      }, limit - (Date.now() - lastRan));
    }
  };
}

// Display more message history (prepend to existing messages)
function displayMoreMessageHistory(messages) {
  const messagesContainer = document.getElementById("messages-container");
  if (!messagesContainer) {
    return;
  }

  // Remove any loading indicator
  const loadingIndicator = messagesContainer.querySelector(".message-loading");
  if (loadingIndicator) {
    loadingIndicator.remove();
  }

  if (!messages || messages.length === 0) {
    // Add a "no more messages" indicator briefly
    const noMoreIndicator = document.createElement("div");
    noMoreIndicator.className = "message-loading";
    noMoreIndicator.textContent = "No more messages";
    messagesContainer.prepend(noMoreIndicator);

    // Remove it after a short delay
    setTimeout(() => {
      noMoreIndicator.remove();
    }, 1500);

    // Reset loading flag
    isLoading = false;
    return;
  }

  // Remember the old scroll height and position
  const oldScrollHeight = messagesContainer.scrollHeight;
  const oldScrollTop = messagesContainer.scrollTop;

  // Create a document fragment to batch DOM updates
  const fragment = document.createDocumentFragment();

  // Display messages in reverse order (oldest first)
  messages.forEach((message) => {
    // Create message element
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
            <div class="message-time" data-timestamp="${
              message.timestamp
            }">${time}</div>
        `;

    fragment.appendChild(messageElem);
  });

  // Prepend all messages at once
  messagesContainer.insertBefore(fragment, messagesContainer.firstChild);

  // Adjust scroll position to maintain the user's view position
  const newScrollHeight = messagesContainer.scrollHeight;
  messagesContainer.scrollTop =
    oldScrollTop + (newScrollHeight - oldScrollHeight);

  // Reset loading flag
  isLoading = false;
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
    return;
  }

  messagesContainer.innerHTML = "";

  if (!messages || messages.length === 0) {
    messagesContainer.innerHTML = `
            <div class="chat-empty-state">
                <h3>Start a conversation</h3>
                <p>No messages yet. Send a message to start the conversation.</p>
            </div>
        `;
    return;
  }

  // Display messages
  messages.forEach((message) => {
    displayMessage(message);
  });

  // Scroll to bottom
  messagesContainer.scrollTop = messagesContainer.scrollHeight;
}

// Improved scroll listener with throttle
function setupScrollListener() {
  const messagesContainer = document.getElementById("messages-container");
  if (!messagesContainer) return;

  let noMoreMessages = false;
  let oldestMessageTimestamp = null;

  // Find the oldest message timestamp if there are messages
  const findOldestMessageTimestamp = () => {
    const messages = messagesContainer.querySelectorAll(".message");
    if (messages.length > 0) {
      const firstMessage = messages[0];
      const timeElem = firstMessage.querySelector(".message-time");
      if (timeElem) {
        return timeElem.getAttribute("data-timestamp");
      }
    }
    return null;
  };

  // Load more messages function
  const loadMoreMessages = () => {
    // Don't load if we're already loading, or if we know there are no more messages
    if (isLoading || !currentChatUser || noMoreMessages) return;

    // Determine if we're at the top of the container (with a small threshold)
    if (messagesContainer.scrollTop < 50) {
      isLoading = true;

      // Show loading indicator
      const loadingIndicator = document.createElement("div");
      loadingIndicator.className = "message-loading";
      loadingIndicator.textContent = "Loading more messages...";
      messagesContainer.prepend(loadingIndicator);

      // Get the oldest message timestamp
      oldestMessageTimestamp = findOldestMessageTimestamp();

      // Request more messages using the timestamp
      if (socket && socket.readyState === WebSocket.OPEN) {
        socket.send(
          JSON.stringify({
            type: "get_more_history",
            receiverID: currentChatUser.id,
            timestamp: oldestMessageTimestamp,
          })
        );

        // Set a timeout to handle the case where no more messages are received
        setTimeout(() => {
          const loadingElem =
            messagesContainer.querySelector(".message-loading");
          if (loadingElem) {
            if (loadingElem.textContent === "Loading more messages...") {
              loadingElem.textContent = "No more messages";
              setTimeout(() => {
                loadingElem.remove();
                noMoreMessages = true; // Mark that we've reached the end
              }, 1500);
            }
            isLoading = false;
          }
        }, 3000);
      } else {
        // If not connected, remove indicator and allow trying again
        loadingIndicator.remove();
        isLoading = false;
      }
    }
  };

  // Apply throttle to prevent rapid firing of the scroll event
  const throttledLoadMore = throttle(loadMoreMessages, 300);

  // Add scroll event listener
  messagesContainer.addEventListener("scroll", throttledLoadMore);
}

// Send a message
function sendMessage() {
  if (!socket || socket.readyState !== WebSocket.OPEN) {
    alert("WebSocket not connected. Please refresh the page.");
    return;
  }

  if (!currentChatUser) {
    alert("Please select a user to chat with");
    return;
  }

  const messageInput = document.getElementById("message-input");
  if (!messageInput) {
    return;
  }

  const content = messageInput.value.trim();

  if (!content) {
    return;
  }

  // Make sure receiverID is explicitly an integer
  const receiverID = parseInt(currentChatUser.id, 10);

  const message = {
    type: "message",
    receiverID: receiverID,
    content: content,
  };

  // Add message locally immediately for better UX
  displayLocalMessage(content);

  // Send via websocket
  try {
    socket.send(JSON.stringify(message));
    messageInput.value = "";
    messageInput.focus();
  } catch (e) {
    console.log("Error sending message:", e);
    alert("Failed to send message: " + e.message);
  }
}

// Check WebSocket connection status
function checkAndConnectWebSocket() {
  // If user isn't logged in, stop checking and clean up
  if (!window.state || !window.state.sessionID || window.state.sessionID <= 0) {
    if (typeof stopWebSocketChecking === "function") {
      stopWebSocketChecking();
    }

    // Reset connection
    if (socket && socket.readyState === WebSocket.OPEN) {
      socket.close();
    }
    socket = null;
    reconnectAttempts = 0;

    // Update status indicator
    const statusElement = document.getElementById("chat-status");
    if (statusElement) {
      statusElement.textContent = "Not logged in";
      statusElement.className = "disconnected";
    }
    return;
  }

  // Only attempt to connect if needed
  if (
    (!socket || socket.readyState !== WebSocket.OPEN) &&
    reconnectAttempts < maxReconnectAttempts
  ) {
    connectWebSocket();
  }
}

// Function to start WebSocket checking interval
function startWebSocketChecking() {
  // Clear any existing interval first
  if (window.wsCheckInterval) {
    clearInterval(window.wsCheckInterval);
  }

  // Only set interval if user is logged in
  if (window.state && window.state.sessionID > 0) {
    window.wsCheckInterval = setInterval(checkAndConnectWebSocket, 10000);
  }
}

// Function to stop WebSocket checking interval
function stopWebSocketChecking() {
  if (window.wsCheckInterval) {
    clearInterval(window.wsCheckInterval);
    window.wsCheckInterval = null;
  }

  // Also close any existing connection
  if (socket && socket.readyState === WebSocket.OPEN) {
    socket.close();
    socket = null;
  }
}

// Setup user list refresh
function setupUserListRefresh() {
  // Refresh user list every 30 seconds if user is logged in
  if (window.userListRefreshInterval) {
    clearInterval(window.userListRefreshInterval);
  }

  if (window.state && window.state.sessionID > 0) {
    window.userListRefreshInterval = setInterval(() => {
      if (typeof fetchAllUsers === "function") {
        fetchAllUsers();
      }
    }, 30000); // Every 30 seconds
  }
}

// Initialize chat when DOM is loaded
document.addEventListener("DOMContentLoaded", function () {
  // Check if we need to connect WebSocket
  if (window.state && window.state.sessionID > 0) {
    setTimeout(connectWebSocket, 1000);
    setupUserListRefresh();
  }

  // Request notification permission
  if ("Notification" in window && Notification.permission === "default") {
    setTimeout(() => {
      Notification.requestPermission();
    }, 5000);
  }
});

// Only start the interval if user is already logged in
if (window.state && window.state.sessionID > 0) {
  startWebSocketChecking();
}

// Add cleanup for when page unloads
window.addEventListener("beforeunload", function () {
  stopWebSocketChecking();
});
