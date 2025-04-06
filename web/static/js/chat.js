// chat.js - WebSocket functionality for private messaging

let socket = null
let currentChatUser = null
const onlineUsers = []
let allUsers = []

// Debug toggle - change to false in production
const DEBUG = false
function debug(message, data) {
  if (DEBUG) {
    console.log(`[CHAT DEBUG] ${message}`, data || "")
  }
}

// Fetch all registered users
async function fetchAllUsers() {
  try {
    debug("Fetching users from /user/all")
    const response = await fetch("/user/all", {
      headers: { Accept: "application/json" },
    })

    if (response.ok) {
      const data = await response.json()
      debug("Received user data:", data)
      allUsers = data.users || []
      updateUsersList()
      return true
    } else {
      debug("Failed to fetch users, status:", response.status)
      return false
    }
  } catch (error) {
    debug("Error fetching users:", error)
    return false
  }
}

// Connect to WebSocket when user is logged in
function connectWebSocket() {
  if (socket !== null && socket.readyState === WebSocket.OPEN) {
    debug("WebSocket already connected")
    return
  }

  const protocol = window.location.protocol === "https:" ? "wss:" : "ws:"
  const wsUrl = `${protocol}//${window.location.host}/ws`
  debug("Connecting to WebSocket:", wsUrl)

  // Update status indicator
  const statusElement = document.getElementById("chat-status")
  if (statusElement) {
    statusElement.textContent = "Connecting..."
    statusElement.className = "connecting"
  }

  try {
    socket = new WebSocket(wsUrl)

    socket.onopen = function () {
      debug("WebSocket connected successfully")
      if (statusElement) {
        statusElement.textContent = "Connected"
        statusElement.className = "connected"
      }

      // Fetch users after connection
      fetchAllUsers()
    }

    socket.onclose = function (event) {
      debug("WebSocket closed with code:", event.code)
      if (statusElement) {
        statusElement.textContent = "Disconnected"
        statusElement.className = "disconnected"
      }
      socket = null

      // Try to reconnect after a delay
      setTimeout(connectWebSocket, 5000)
    }

    socket.onerror = function (error) {
      debug("WebSocket error:", error)
      if (statusElement) {
        statusElement.textContent = "Error"
        statusElement.className = "disconnected"
      }
    }

    socket.onmessage = function (event) {
      try {
        debug("Raw message received:", event.data)
        const data = JSON.parse(event.data)
        debug("Parsed WebSocket message:", data)

        if (data.type === "user_list") {
          // Call our broadcastUserList function to handle user list updates
          broadcastUserList(data.users)
        } else if (data.type === "message") {
          // Handle received message
          debug("Received chat message:", data)
          displayMessage(data)

          // Show notification if message is not from current chat
          if (
            data.sender_id !== state.sessionID &&
            (!currentChatUser || data.sender_id !== currentChatUser.id)
          ) {
            showMessageNotification(data)
          }
        } else if (data.type === "history") {
          debug("Received message history:", data)
          if (Array.isArray(data.messages)) {
            displayMessageHistory(data.messages)
          } else {
            debug("Invalid message history format:", data.messages)
            displayMessageHistory([]) // Empty history
          }
        } else if (data.type === "more_history") {
          debug("Received more message history:", data)
          if (Array.isArray(data.messages)) {
            displayMoreMessageHistory(data.messages)
          } else {
            debug("Invalid more message history format:", data.messages)
          }
        }
      } catch (e) {
        debug("Error processing WebSocket message:", e)
        debug("Raw message was:", event.data)
      }
    }
  } catch (e) {
    debug("Error creating WebSocket:", e)
    if (statusElement) {
      statusElement.textContent = "Failed"
      statusElement.className = "disconnected"
    }
  }
}

// Update the broadcast user list function to automatically update the UI
function broadcastUserList(users) {
  debug("Received updated user list:", users)
  onlineUsers.length = 0

  if (Array.isArray(users)) {
    users.forEach((user) => {
      onlineUsers.push(user.id)
    })
  }

  // Update the UI immediately
  updateUsersList()
}

// Show notification for new message
function showMessageNotification(message) {
  // Find the username from users list
  let senderName = "User"
  const userItem = document.querySelector(
    `.user-item[data-user-id="${message.sender_id}"]`
  )
  if (userItem) {
    senderName = userItem.dataset.username
  }

  debug("Showing notification for message from:", senderName)

  // Create notification if browser supports it
  if ("Notification" in window) {
    if (Notification.permission === "granted") {
      new Notification(`New message from ${senderName}`, {
        body:
          message.content.substring(0, 50) +
          (message.content.length > 50 ? "..." : ""),
      })
    } else if (Notification.permission !== "denied") {
      Notification.requestPermission()
    }
  }

  // Highlight the user in the list
  if (userItem) {
    userItem.classList.add("has-new-message")
  }
}

// Update the list of users
function updateUsersList() {
  const usersList = document.getElementById("users-list")
  if (!usersList) {
    debug("users-list element not found")
    return
  }

  debug("Updating users list with:", { allUsers, onlineUsers })
  usersList.innerHTML = ""

  if (allUsers.length === 0) {
    const emptyMessage = document.createElement("p")
    emptyMessage.className = "empty-users-message"
    emptyMessage.textContent = "No users found"
    usersList.appendChild(emptyMessage)
    return
  }

  // Sort users:
  // 1. Online users first
  // 2. Users with recent messages next (if we had message timestamps)
  // 3. Then alphabetically
  const sortedUsers = [...allUsers].sort((a, b) => {
    const aOnline = onlineUsers.includes(a.id)
    const bOnline = onlineUsers.includes(b.id)

    // Online users first
    if (aOnline && !bOnline) return -1
    if (!aOnline && bOnline) return 1

    // Then alphabetically
    return a.username.localeCompare(b.username)
  })

  sortedUsers.forEach((user) => {
    // Don't show current user
    if (user.id === state.sessionID) return

    const userItem = document.createElement("div")
    userItem.className = "user-item"

    if (onlineUsers.includes(user.id)) {
      userItem.classList.add("online")
    } else {
      userItem.classList.add("offline")
    }

    userItem.textContent = user.username
    userItem.dataset.userId = user.id
    userItem.dataset.username = user.username

    userItem.addEventListener("click", function () {
      debug("User clicked:", user)
      openChat(user.id, user.username)
      // Remove highlight when clicked
      this.classList.remove("has-new-message")
    })

    usersList.appendChild(userItem)
  })

  debug("Users list updated with", allUsers.length, "users")
}

// Open chat with a specific user
function openChat(userId, username) {
  // Ensure userId is an integer
  userId = parseInt(userId, 10)
  currentChatUser = { id: userId, name: username }
  debug("Opening chat with:", currentChatUser)

  // Update main content area with chat interface
  const content = document.getElementById("content")
  if (!content) {
    debug("Content element not found")
    return
  }

  content.innerHTML = templates.chatInterface(username)

  // Set up back button
  document
    .getElementById("back-to-posts")
    .addEventListener("click", function () {
      debug("Back to posts clicked")
      loadHomePage()
      currentChatUser = null
    })

  // Set up message send button
  document
    .getElementById("send-message-button")
    .addEventListener("click", sendMessage)

  // Set up enter key to send message
  document
    .getElementById("message-input")
    .addEventListener("keydown", function (e) {
      if (e.key === "Enter" && !e.shiftKey) {
        e.preventDefault()
        sendMessage()
      }
    })

  // Request message history
  if (socket && socket.readyState === WebSocket.OPEN) {
    debug("Requesting message history with:", userId)
    socket.send(
      JSON.stringify({
        type: "get_history",
        receiverID: userId,
      })
    )
  } else {
    debug("Socket not ready, cannot request history")
    const messagesContainer = document.getElementById("messages-container")
    if (messagesContainer) {
      messagesContainer.innerHTML = `
        <div class="chat-empty-state">
          <h3>Chat connection error</h3>
          <p>Not connected to chat server. Please refresh the page and try again.</p>
        </div>
      `
    }
  }

  // Set up scroll listener for loading more messages
  setTimeout(() => {
    setupScrollListener()
  }, 500)

  // Focus the message input
  setTimeout(() => {
    const messageInput = document.getElementById("message-input")
    if (messageInput) messageInput.focus()
  }, 100)
}

// Display a locally sent message before confirmation
function displayLocalMessage(content) {
  const messagesContainer = document.getElementById("messages-container")
  if (!messagesContainer) {
    debug("Messages container not found")
    return
  }

  // Clear empty state if it exists
  const emptyState = messagesContainer.querySelector(".chat-empty-state")
  if (emptyState) {
    emptyState.remove()
  }

  const messageElem = document.createElement("div")
  messageElem.className = "message outgoing"

  const time = new Date().toLocaleTimeString()
  messageElem.innerHTML = `
    <div class="message-text">${escapeHTML(content)}</div>
    <div class="message-time">${time} (Sending...)</div>
  `

  messagesContainer.appendChild(messageElem)
  messagesContainer.scrollTop = messagesContainer.scrollHeight
  debug("Local message displayed")
}

// Display a received message
function displayMessage(message) {
  debug("Displaying message:", message)

  // Only display if it's part of the current chat
  if (
    currentChatUser &&
    ((message.sender_id === currentChatUser.id &&
      message.receiverID === state.sessionID) ||
      (message.sender_id === state.sessionID &&
        message.receiverID === currentChatUser.id))
  ) {
    const messagesContainer = document.getElementById("messages-container")
    if (!messagesContainer) {
      debug("Messages container not found")
      return
    }

    // Clear empty state if it exists
    const emptyState = messagesContainer.querySelector(".chat-empty-state")
    if (emptyState) {
      emptyState.remove()
    }

    // Find existing "sending..." message if this is a confirmation of our sent message
    if (message.sender_id === state.sessionID) {
      const pendingMessages =
        messagesContainer.querySelectorAll(".message.outgoing")
      for (const pending of pendingMessages) {
        const timeElem = pending.querySelector(".message-time")
        if (timeElem && timeElem.textContent.includes("Sending...")) {
          // Update this message instead of creating a new one
          timeElem.textContent = new Date(
            message.timestamp
          ).toLocaleTimeString()
          // Add timestamp as data attribute
          timeElem.setAttribute("data-timestamp", message.timestamp)
          debug("Updated pending message with confirmation")
          return
        }
      }
    }

    // Create new message element
    const messageElem = document.createElement("div")
    messageElem.className = "message"

    if (message.sender_id === state.sessionID) {
      messageElem.classList.add("outgoing")
    } else {
      messageElem.classList.add("incoming")
    }

    const time = new Date(message.timestamp).toLocaleTimeString()
    messageElem.innerHTML = `
      <div class="message-text">${escapeHTML(message.content)}</div>
      <div class="message-time" data-timestamp="${
        message.timestamp
      }">${time}</div>
    `

    messagesContainer.appendChild(messageElem)
    messagesContainer.scrollTop = messagesContainer.scrollHeight
    debug("Message displayed in chat")
  } else {
    debug("Message not displayed (not for current chat)")
  }
}

// Display more message history (prepend to existing messages)
function displayMoreMessageHistory(messages) {
  const messagesContainer = document.getElementById("messages-container")
  if (!messagesContainer) {
    debug("Messages container not found for more history")
    return
  }

  // Remove any loading indicator
  const loadingIndicator = messagesContainer.querySelector(".message-loading")
  if (loadingIndicator) {
    loadingIndicator.remove()
  }

  if (!messages || messages.length === 0) {
    debug("No more message history to display")
    return
  }

  debug("Displaying more message history:", messages.length, "messages")

  // Remember the old scroll height and position
  const oldScrollHeight = messagesContainer.scrollHeight
  const oldScrollTop = messagesContainer.scrollTop

  // Create a document fragment to batch DOM updates
  const fragment = document.createDocumentFragment()

  // Display messages in reverse order (oldest first)
  messages.forEach((message) => {
    // Create message element
    const messageElem = document.createElement("div")
    messageElem.className = "message"

    if (message.sender_id === state.sessionID) {
      messageElem.classList.add("outgoing")
    } else {
      messageElem.classList.add("incoming")
    }

    const time = new Date(message.timestamp).toLocaleTimeString()
    messageElem.innerHTML = `
      <div class="message-text">${escapeHTML(message.content)}</div>
      <div class="message-time" data-timestamp="${
        message.timestamp
      }">${time}</div>
    `

    fragment.appendChild(messageElem)
  })

  // Prepend all messages at once
  messagesContainer.insertBefore(fragment, messagesContainer.firstChild)

  // Adjust scroll position to maintain the user's view position
  const newScrollHeight = messagesContainer.scrollHeight
  messagesContainer.scrollTop =
    oldScrollTop + (newScrollHeight - oldScrollHeight)
}

// Helper function to escape HTML special characters
function escapeHTML(text) {
  return text
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#039;")
}

// Display message history
function displayMessageHistory(messages) {
  const messagesContainer = document.getElementById("messages-container")
  if (!messagesContainer) {
    debug("Messages container not found for history")
    return
  }

  messagesContainer.innerHTML = ""

  if (!messages || messages.length === 0) {
    debug("No message history to display")
    messagesContainer.innerHTML = `
      <div class="chat-empty-state">
        <h3>Start a conversation</h3>
        <p>No messages yet. Send a message to start the conversation.</p>
      </div>
    `
    return
  }

  debug("Displaying message history:", messages.length, "messages")

  // Display messages
  messages.forEach((message) => {
    displayMessage(message)
  })

  // Scroll to bottom
  messagesContainer.scrollTop = messagesContainer.scrollHeight
}

// Implement the scroll loading with debounce
// Add a debounce function to limit how often a function can be called
function debounce(func, wait) {
  let timeout
  return function () {
    const context = this
    const args = arguments
    clearTimeout(timeout)
    timeout = setTimeout(() => {
      func.apply(context, args)
    }, wait)
  }
}

// Improve scrolling function with debounce
function setupScrollListener() {
  const messagesContainer = document.getElementById("messages-container")
  if (!messagesContainer) return

  let isLoading = false
  let oldestMessageTimestamp = null

  // Find the oldest message timestamp if there are messages
  const findOldestMessageTimestamp = () => {
    const messages = messagesContainer.querySelectorAll(".message")
    if (messages.length > 0) {
      const firstMessage = messages[0]
      const timeElem = firstMessage.querySelector(".message-time")
      if (timeElem) {
        return timeElem.getAttribute("data-timestamp")
      }
    }
    return null
  }

  // Load more messages function
  const loadMoreMessages = () => {
    if (isLoading || !currentChatUser) return

    // Determine if we're at the top of the container
    if (messagesContainer.scrollTop < 50) {
      isLoading = true

      // Show loading indicator
      const loadingIndicator = document.createElement("div")
      loadingIndicator.className = "message-loading"
      loadingIndicator.textContent = "Loading more messages..."
      messagesContainer.prepend(loadingIndicator)

      // Get the oldest message timestamp
      oldestMessageTimestamp = findOldestMessageTimestamp()

      // Remember scroll position
      const oldScrollHeight = messagesContainer.scrollHeight

      // Request more messages using the timestamp
      if (socket && socket.readyState === WebSocket.OPEN) {
        socket.send(
          JSON.stringify({
            type: "get_more_history",
            receiverID: currentChatUser.id,
            timestamp: oldestMessageTimestamp,
          })
        )

        // Simulated response timing (in a real implementation, this would be handled in the message event)
        setTimeout(() => {
          // If no response was received, remove the loading indicator
          if (messagesContainer.querySelector(".message-loading")) {
            loadingIndicator.remove()
            isLoading = false
          }
        }, 3000)
      } else {
        // If not connected, remove indicator and allow trying again
        loadingIndicator.remove()
        isLoading = false
      }
    }
  }

  // Apply debounce to prevent rapid firing of the scroll event
  const debouncedLoadMore = debounce(loadMoreMessages, 300)

  // Add scroll event listener
  messagesContainer.addEventListener("scroll", debouncedLoadMore)
}

// Send a message
function sendMessage() {
  if (!socket || socket.readyState !== WebSocket.OPEN) {
    alert("WebSocket not connected. Please refresh the page.")
    debug("Cannot send message, WebSocket not connected")
    return
  }

  if (!currentChatUser) {
    alert("Please select a user to chat with")
    debug("Cannot send message, no chat user selected")
    return
  }

  const messageInput = document.getElementById("message-input")
  if (!messageInput) {
    debug("Message input element not found")
    return
  }

  const content = messageInput.value.trim()

  if (!content) {
    debug("Empty message, not sending")
    return
  }

  // Make sure receiverID is explicitly an integer
  const receiverID = parseInt(currentChatUser.id, 10)

  const message = {
    type: "message",
    receiverID: receiverID,
    content: content,
  }

  // Log for debugging
  debug("Sending message:", message)

  // Add message locally immediately for better UX
  displayLocalMessage(content)

  // Send via websocket
  try {
    const jsonString = JSON.stringify(message)
    debug("Sending stringified message:", jsonString)
    socket.send(jsonString)
    messageInput.value = ""
    messageInput.focus()
  } catch (e) {
    debug("Error sending message:", e)
    alert("Failed to send message: " + e.message)
  }
}

// Initialize chat when DOM is loaded
document.addEventListener("DOMContentLoaded", function () {
  debug("DOM loaded, checking session")
  // Check if we need to connect WebSocket
  if (window.state && window.state.sessionID > 0) {
    debug("Session found, connecting WebSocket")
    setTimeout(connectWebSocket, 500) // Short delay to ensure state is fully initialized
  } else {
    debug("No active session, skipping WebSocket connection")
  }

  // Request notification permission
  if ("Notification" in window && Notification.permission === "default") {
    // Wait a moment before requesting permission to avoid overwhelming the user
    setTimeout(() => {
      Notification.requestPermission()
    }, 5000)
  }
})

// Watch for login status changes
function checkAndConnectWebSocket() {
  debug("Checking if WebSocket connection needed")
  if (
    window.state &&
    window.state.sessionID > 0 &&
    (!socket || socket.readyState !== WebSocket.OPEN)
  ) {
    debug("User logged in, connecting WebSocket")
    connectWebSocket()
  }
}

// Set up a regular check for WebSocket connection
setInterval(checkAndConnectWebSocket, 2000)
