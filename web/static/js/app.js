// app.js - Simplified main application file for SPA

// Application state
const state = {
  sessionID: null,
  username: null,
  posts: [],
  currentPost: null,
}

// Make state available globally for template access
window.state = state

// Initialize the app when DOM is loaded
document.addEventListener("DOMContentLoaded", init)

async function init() {
  // Check user login status
  await checkUserStatus()

  // Set up navigation
  setupNavigation()

  // Handle the current route
  handleRoute()
}

// Check if user is logged in
async function checkUserStatus() {
  try {
    const response = await fetch("/user/status")
    const data = await response.json()

    state.sessionID = data.sessionID
    state.username = data.username

    // Update UI elements based on login status
    updateAuthBox()
    updateSidebar()
  } catch (error) {
    console.error("Error checking user status:", error)
  }
}

// Update the authentication box in the header
function updateAuthBox() {
  const authBox = document.getElementById("auth-box")
  authBox.innerHTML = templates.authBox(state.sessionID, state.username)

  // Add event listener for logout if user is logged in
  if (state.sessionID) {
    document
      .getElementById("logout-link")
      .addEventListener("click", handleLogout)
  }
}

// Update the sidebar categories and filters
function updateSidebar() {
  const sidebar = document.getElementById("sidebar")
  sidebar.innerHTML = templates.sidebar(state.sessionID)
}

// Set up SPA navigation
function setupNavigation() {
  // Handle link clicks with data-navigate attribute
  document.addEventListener("click", (event) => {
    const link = event.target.closest("[data-navigate]")
    if (link) {
      event.preventDefault()
      navigateTo(link.getAttribute("href"))
    }
  })

  // Handle browser back/forward buttons
  window.addEventListener("popstate", handleRoute)
}

// Navigate to a URL
function navigateTo(url) {
  history.pushState(null, "", url)
  handleRoute()
}

// Route handler - determines what to show based on URL
function handleRoute() {
  const path = window.location.pathname
  const searchParams = new URLSearchParams(window.location.search)

  // Show loading state
  document.getElementById("content").innerHTML = templates.loading()

  // Handle different routes
  if (path === "/" || path === "/index.html") {
    fetchPosts()
  } else if (path === "/post") {
    const postId = searchParams.get("id")
    if (postId && postId !== "undefined") {
      fetchSinglePost(postId)
    } else {
      showError("Post ID is missing or invalid")
    }
  } else if (path === "/filter") {
    fetchFilteredPosts(window.location.search)
  } else if (path === "/login") {
    showLoginForm()
  } else if (path === "/register") {
    showRegisterForm()
  } else if (path === "/createPost") {
    if (state.sessionID) {
      showCreatePostForm()
    } else {
      navigateTo("/login")
    }
  } else {
    showError("Page not found")
  }
}

// Fetch posts for homepage
async function fetchPosts() {
  const contentElement = document.getElementById("content")

  try {
    const response = await fetch("/?api=true", {
      headers: {
        Accept: "application/json",
        "X-Requested-With": "XMLHttpRequest",
      },
    })

    if (!response.ok) {
      throw new Error(`HTTP error! Status: ${response.status}`)
    }

    const data = await response.json()
    state.posts = data.Posts || []

    // Display posts using template
    contentElement.innerHTML = templates.homePage(state.posts)

    // Set up reaction buttons
    setupReactionButtons()
  } catch (error) {
    console.error("Error fetching posts:", error)
    contentElement.innerHTML = templates.error(
      "Failed to load posts. Please try again later."
    )
  }
}

// Fetch a single post
async function fetchSinglePost(postId) {
  const contentElement = document.getElementById("content")

  try {
    const response = await fetch(`/post?id=${postId}`, {
      headers: { Accept: "application/json" },
    })

    if (!response.ok) {
      throw new Error(`Server error: ${response.status}`)
    }

    const data = await response.json()
    state.currentPost = data.post || data.Post

    if (!state.currentPost || !state.currentPost.ID) {
      throw new Error("Invalid post data")
    }

    // Display post and comments using template
    contentElement.innerHTML = templates.postDetail(state.currentPost)

    // Set up reaction buttons
    setupReactionButtons()

    // Set up comment form if logged in
    const commentForm = document.getElementById("comment-form")
    if (commentForm) {
      commentForm.addEventListener("submit", handleAddComment)
    }
  } catch (error) {
    console.error("Error fetching post:", error)
    contentElement.innerHTML = templates.error(
      `Failed to load post: ${error.message}`
    )
  }
}

// Fetch filtered posts
async function fetchFilteredPosts(queryString) {
  const contentElement = document.getElementById("content")

  try {
    const response = await fetch(`/filter${queryString}`, {
      headers: { Accept: "application/json" },
    })

    if (!response.ok) {
      throw new Error(`HTTP error! Status: ${response.status}`)
    }

    const data = await response.json()
    state.posts = data.posts || []

    // Determine filter title
    let title = "Filtered Posts"
    if (data.category) {
      title = `${data.category} Posts`
    } else if (queryString.includes("user_created=true")) {
      title = "My Posts"
    } else if (queryString.includes("liked=true")) {
      title = "Liked Posts"
    }

    // Display posts using template
    contentElement.innerHTML = templates.filteredPosts(title, state.posts)
  } catch (error) {
    console.error("Error fetching filtered posts:", error)
    contentElement.innerHTML = templates.error(
      "Failed to load posts. Please try again later."
    )
  }
}

// Show login form
function showLoginForm() {
  const contentElement = document.getElementById("content")
  contentElement.innerHTML = templates.loginForm()
  document.getElementById("login-form").addEventListener("submit", handleLogin)
}

// Show registration form
function showRegisterForm() {
  const contentElement = document.getElementById("content")
  contentElement.innerHTML = templates.registerForm()
  document
    .getElementById("register-form")
    .addEventListener("submit", handleRegister)
}

// Show create post form
function showCreatePostForm() {
  const contentElement = document.getElementById("content")
  contentElement.innerHTML = templates.createPostForm()
  document
    .getElementById("create-post-form")
    .addEventListener("submit", handleCreatePost)
}

// Show error message
function showError(message) {
  const contentElement = document.getElementById("content")
  contentElement.innerHTML = templates.error(message)
}

// Handle login form submission
async function handleLogin(event) {
  event.preventDefault()

  const form = event.target
  const formData = new FormData(form)

  try {
    const response = await fetch("/login", {
      method: "POST",
      body: formData,
    })

    const data = await response.json()

    if (response.ok) {
      // Login successful
      state.sessionID = data.sessionID
      state.username = data.username

      updateAuthBox()
      updateSidebar()
      navigateTo("/")
    } else {
      // Show error message
      const errorElement = document.getElementById("login-error")
      errorElement.textContent = data.message
      errorElement.style.display = "block"
    }
  } catch (error) {
    console.error("Login error:", error)
    document.getElementById("login-error").textContent =
      "An error occurred. Please try again."
    document.getElementById("login-error").style.display = "block"
  }
}

// Handle register form submission
async function handleRegister(event) {
  event.preventDefault()

  const form = event.target
  const formData = new FormData(form)

  try {
    const response = await fetch("/register", {
      method: "POST",
      body: formData,
    })

    const data = await response.json()

    if (response.ok) {
      // Registration successful
      navigateTo("/login")
    } else {
      // Show error message
      const errorElement = document.getElementById("register-error")
      errorElement.textContent = data.message
      errorElement.style.display = "block"
    }
  } catch (error) {
    console.error("Registration error:", error)
    document.getElementById("register-error").textContent =
      "An error occurred. Please try again."
    document.getElementById("register-error").style.display = "block"
  }
}

// Handle create post submission
async function handleCreatePost(event) {
  event.preventDefault()

  const form = event.target
  const formData = new FormData(form)

  try {
    const response = await fetch("/createPost", {
      method: "POST",
      body: formData,
    })

    const data = await response.json()

    if (response.ok) {
      // Successfully created post
      setTimeout(() => {
        navigateTo(`/post?id=${data.id}`)
      }, 300)
    } else {
      alert(data.message || "Failed to create post")
    }
  } catch (error) {
    console.error("Create post error:", error)
    alert("An error occurred. Please try again.")
  }
}

// Handle adding comments
async function handleAddComment(event) {
  event.preventDefault()

  const form = event.target
  const formData = new FormData(form)
  const postId = formData.get("post_id")

  try {
    const response = await fetch("/comment", {
      method: "POST",
      body: formData,
    })

    if (response.ok) {
      // Comment added, refresh the post
      fetchSinglePost(postId)
    } else {
      const data = await response.json()
      alert(data.message || "Failed to add comment")
    }
  } catch (error) {
    console.error("Add comment error:", error)
    alert("An error occurred. Please try again.")
  }
}

// Handle logout
async function handleLogout(event) {
  event.preventDefault()

  try {
    const response = await fetch("/logout", {
      method: "POST",
    })

    if (response.ok) {
      // Logout successful
      state.sessionID = null
      state.username = null

      updateAuthBox()
      updateSidebar()
      navigateTo("/")
    }
  } catch (error) {
    console.error("Logout error:", error)
  }
}

// Setup reaction buttons
function setupReactionButtons() {
  document
    .querySelectorAll(".like-button, .dislike-button")
    .forEach((button) => {
      button.addEventListener("click", () => handleReaction(button))
    })
}

// Handle like/dislike reactions
async function handleReaction(button) {
  // Check if user is logged in
  if (!state.sessionID) {
    navigateTo("/login")
    return
  }

  const itemId = button.getAttribute("data-id")
  const isComment = button.getAttribute("data-for") === "comment"
  const reactionType = button.getAttribute("data-type")

  const formData = new FormData()
  formData.append("item_id", itemId)
  formData.append("is_comment", isComment)
  formData.append("type", reactionType)

  try {
    const response = await fetch("/like", {
      method: "POST",
      body: formData,
    })

    if (response.ok) {
      // Reaction recorded, refresh the current view
      if (window.location.pathname === "/post") {
        const postId = new URLSearchParams(window.location.search).get("id")
        fetchSinglePost(postId)
      } else {
        fetchPosts()
      }
    }
  } catch (error) {
    console.error("Reaction error:", error)
  }
}
