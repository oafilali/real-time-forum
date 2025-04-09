// chat-ui.js - Chat user interface functionality

// Global variables for chat UI
let currentChatUser = null;
let allUsers = [];
let isLoading = false;
let userListRefreshInterval = null;

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
      if (
        response.status === 401 &&
        window.appCore &&
        window.appCore.checkLogin
      ) {
        window.appCore.checkLogin();
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
    if (!window.chatMessages || !window.chatMessages.getLastMessagesData) {
      return null;
    }

    const lastMessagesData = window.chatMessages.getLastMessagesData(userId);
    if (!lastMessagesData) return null;

    let latestTime = null;
    Object.values(lastMessagesData).forEach((msg) => {
      const msgTime = new Date(msg.timestamp).getTime();
      if (!latestTime || msgTime > latestTime) {
        latestTime = msgTime;
      }
    });
    return latestTime;
  };

  // Sort users:
  // 1. Users with unread messages first
  // 2.Onl ine users next
  // 3. Users with recent messages next (sorted by timestamp)
  // 4. Then alphabetically
  const sortedUsers = [...allUsers].sort((a, b) => {
    if (!window.chatMessages) return 0;

    const aHasUnread = window.chatMessages.hasUnreadMessages(a.id);
    const bHasUnread = window.chatMessages.hasUnreadMessages(b.id);

    // Users with unread messages first
    if (aHasUnread && !bHasUnread) return -1;
    if (!aHasUnread && bHasUnread) return 1;

    const aOnline = window.chatMessages.isUserOnline(a.id);
    const bOnline = window.chatMessages.isUserOnline(b.id);

    // Then online users first
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
    if (user.id === window.state.sessionID) return;

    const userItem = document.createElement("div");
    userItem.className = "user-item";

    if (window.chatMessages && window.chatMessages.isUserOnline(user.id)) {
      userItem.classList.add("online");
    } else {
      userItem.classList.add("offline");
    }

    // Add notification indicator if user has unread messages
    if (window.chatMessages && window.chatMessages.hasUnreadMessages(user.id)) {
      userItem.classList.add("has-new-message");
    }

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
  // Ensure userId is an integer
  userId = parseInt(userId, 10);
  currentChatUser = { id: userId, name: username };

  // Clear unread messages for this user
  if (window.chatMessages) {
    window.chatMessages.clearUnreadMessages(userId);
  }

  // Update the user list to remove the notification
  updateUsersList();

  // Update main content area with chat interface
  const content = document.getElementById("content");
  if (!content) {
    return;
  }

  content.innerHTML = window.templates.chatInterface(username);

  // Set up back button
  document
    .getElementById("back-to-posts")
    .addEventListener("click", function () {
      if (window.appPages) {
        window.appPages.loadHomePage();
      } else {
        window.location.href = "/";
      }
      currentChatUser = null;
    });

  // Set up message send button
  document
    .getElementById("send-message-button")
    .addEventListener("click", function () {
      if (window.chatMessages && window.chatMessages.sendMessage) {
        window.chatMessages.sendMessage();
      }
    });

  // Set up enter key to send message
  document
    .getElementById("message-input")
    .addEventListener("keydown", function (e) {
      if (e.key === "Enter" && !e.shiftKey) {
        e.preventDefault();
        if (window.chatMessages && window.chatMessages.sendMessage) {
          window.chatMessages.sendMessage();
        }
      }
    });

  // Request message history
  const socket = window.chatConnection ? window.chatConnection.socket() : null;
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
  if (window.chatMessages && window.chatMessages.displayMessage) {
    messages.forEach((message) => {
      window.chatMessages.displayMessage(message);
    });
  }

  // Scroll to bottom
  messagesContainer.scrollTop = messagesContainer.scrollHeight;
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

    if (message.sender_id === window.state.sessionID) {
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
      const socket = window.chatConnection
        ? window.chatConnection.socket()
        : null;
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

// Setup user list refresh
function setupUserListRefresh() {
  // Refresh user list every 30 seconds if user is logged in
  if (userListRefreshInterval) {
    clearInterval(userListRefreshInterval);
  }

  if (window.state && window.state.sessionID > 0) {
    userListRefreshInterval = setInterval(() => {
      fetchAllUsers();
    }, 30000); // Every 30 seconds
  }
}

// Get current chat user
function getCurrentUser() {
  return currentChatUser;
}

// Export chat UI functions
window.chatUI = {
  currentUser: getCurrentUser,
  fetchAllUsers,
  updateUsersList,
  openChat,
  displayLocalMessage,
  displayMessageHistory,
  displayMoreMessageHistory,
  setupScrollListener,
  setupUserListRefresh,
  escapeHTML,
};

// Initialize chat UI when DOM is loaded
document.addEventListener("DOMContentLoaded", function () {
  // Setup user list refresh
  setupUserListRefresh();

  // Request notification permission
  if ("Notification" in window && Notification.permission === "default") {
    setTimeout(() => {
      Notification.requestPermission();
    }, 5000);
  }
});
