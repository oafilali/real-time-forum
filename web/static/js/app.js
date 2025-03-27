// app.js - Simplified main application file for SPA

// Application state
const state = {
  sessionID: null,
  username: null,
  posts: [],
  currentPost: null,
};

// Initialize the app when DOM is loaded
document.addEventListener("DOMContentLoaded", init);

async function init() {
  // Check user login status
  await checkUserStatus();

  // Set up navigation
  setupNavigation();

  // Handle the current route
  handleRoute();
}

// Check if user is logged in
async function checkUserStatus() {
  try {
    const response = await fetch("/user/status");
    const data = await response.json();

    state.sessionID = data.sessionID;
    state.username = data.username;

    // Update UI elements based on login status
    updateAuthBox();
    updateSidebar();
  } catch (error) {
    console.error("Error checking user status:", error);
  }
}

// Update the authentication box in the header
function updateAuthBox() {
  const authBox = document.getElementById("auth-box");

  if (state.sessionID && state.username) {
    // User is logged in
    authBox.innerHTML = `
        <p>Logged in as: <strong>${state.username}</strong></p>
        <a href="/createPost" data-navigate>Create Post</a>
        <a href="#" id="logout-link">Logout</a>
      `;

    document
      .getElementById("logout-link")
      .addEventListener("click", handleLogout);
  } else {
    // User is not logged in
    authBox.innerHTML = `
        <a href="/login" data-navigate>Login</a>
        <a href="/register" data-navigate>Register</a>
      `;
  }
}

// Update the sidebar categories and filters
function updateSidebar() {
  const sidebar = document.getElementById("sidebar");

  let sidebarHTML = `
      <h2>Categories:</h2>
      <p><a href="/filter?category=General" data-navigate>General</a></p>
      <p><a href="/filter?category=Local%20News%20%26%20Events" data-navigate>Local News & Events</a></p>
      <p><a href="/filter?category=Viking%20line" data-navigate>Viking line</a></p>
      <p><a href="/filter?category=Travel" data-navigate>Travel</a></p>
      <p><a href="/filter?category=Sailing" data-navigate>Sailing</a></p>
      <p><a href="/filter?category=Cuisine%20%26%20food" data-navigate>Cuisine & food</a></p>
      <p><a href="/filter?category=Politics" data-navigate>Politics</a></p>
    `;

  // Add user-specific filters if logged in
  if (state.sessionID) {
    sidebarHTML += `
        <h2>Filters:</h2>
        <p><a href="/filter?user_created=true" data-navigate>My Posts</a></p>
        <p><a href="/filter?liked=true" data-navigate>Liked Posts</a></p>
      `;
  }

  // Add home button
  sidebarHTML += `
      <div class="category-button">
        <a href="/" data-navigate class="back-to-home">Back to Home</a>
      </div>
    `;

  sidebar.innerHTML = sidebarHTML;
}

// Set up SPA navigation
function setupNavigation() {
  // Handle link clicks with data-navigate attribute
  document.addEventListener("click", (event) => {
    const link = event.target.closest("[data-navigate]");
    if (link) {
      event.preventDefault();
      navigateTo(link.getAttribute("href"));
    }
  });

  // Handle browser back/forward buttons
  window.addEventListener("popstate", handleRoute);
}

// Navigate to a URL
function navigateTo(url) {
  history.pushState(null, "", url);
  handleRoute();
}

// Route handler - determines what to show based on URL
function handleRoute() {
  const path = window.location.pathname;
  const searchParams = new URLSearchParams(window.location.search);

  // Show loading state
  document.getElementById("content").innerHTML =
    '<div class="loading">Loading...</div>';

  // Handle different routes
  if (path === "/" || path === "/index.html") {
    fetchPosts();
  } else if (path === "/post") {
    const postId = searchParams.get("id");
    if (postId && postId !== "undefined") {
      fetchSinglePost(postId);
    } else {
      showError("Post ID is missing or invalid");
    }
  } else if (path === "/filter") {
    fetchFilteredPosts(window.location.search);
  } else if (path === "/login") {
    showLoginForm();
  } else if (path === "/register") {
    showRegisterForm();
  } else if (path === "/createPost") {
    if (state.sessionID) {
      showCreatePostForm();
    } else {
      navigateTo("/login");
    }
  } else {
    showError("Page not found");
  }
}

// Fetch posts for homepage
async function fetchPosts() {
  const contentElement = document.getElementById("content");

  try {
    const response = await fetch("/?api=true", {
      headers: {
        Accept: "application/json",
        "X-Requested-With": "XMLHttpRequest",
      },
    });

    if (!response.ok) {
      throw new Error(`HTTP error! Status: ${response.status}`);
    }

    const data = await response.json();
    state.posts = data.Posts || [];

    if (state.posts.length > 0) {
      // Display posts
      contentElement.innerHTML = `
          <h2>All posts</h2>
          ${state.posts.map((post) => getPostTemplate(post)).join("")}
        `;

      // Set up reaction buttons
      setupReactionButtons();
    } else {
      contentElement.innerHTML = `
          <h2>All posts</h2>
          <p>No posts available.</p>
        `;
    }
  } catch (error) {
    console.error("Error fetching posts:", error);
    contentElement.innerHTML =
      '<div class="error">Failed to load posts. Please try again later.</div>';
  }
}

// Template for post in feed
function getPostTemplate(post) {
  return `
      <div class="post">
        <h2><a href="/post?id=${post.ID}" data-navigate>${post.Title}</a></h2>
        <p>${post.Content}</p>
        <div class="post-meta">
          <div class="left">
            <span class="username">Posted by: <strong>${
              post.Username
            }</strong></span>
            <span class="date">Date: ${post.Date || "Unknown"}</span>
          </div>
          <div class="right">
            <button class="like-button" data-id="${
              post.ID
            }" data-type="like" data-for="post">👍 ${post.Likes || 0}</button>
            <button class="dislike-button" data-id="${
              post.ID
            }" data-type="dislike" data-for="post">👎 ${
    post.Dislikes || 0
  }</button>
          </div>
        </div>
      </div>
    `;
}

// Fetch a single post
async function fetchSinglePost(postId) {
  const contentElement = document.getElementById("content");

  try {
    const response = await fetch(`/post?id=${postId}`, {
      headers: { Accept: "application/json" },
    });

    if (!response.ok) {
      throw new Error(`Server error: ${response.status}`);
    }

    const data = await response.json();
    state.currentPost = data.post || data.Post;

    if (!state.currentPost || !state.currentPost.ID) {
      throw new Error("Invalid post data");
    }

    // Display post and comments
    contentElement.innerHTML = getPostDetailTemplate(state.currentPost);

    // Set up reaction buttons
    setupReactionButtons();

    // Set up comment form if logged in
    const commentForm = document.getElementById("comment-form");
    if (commentForm) {
      commentForm.addEventListener("submit", handleAddComment);
    }
  } catch (error) {
    console.error("Error fetching post:", error);
    contentElement.innerHTML = `<div class="error">Failed to load post: ${error.message}</div>`;
  }
}

// Template for single post view
function getPostDetailTemplate(post) {
  return `
      <div class="post" id="post-${post.ID}">
        <h2>Category: ${post.Category}</h2>
        <h3>${post.Title}</h3>
        <p class="author">Posted by: <strong>${post.Username}</strong> on ${
    post.Date || "Unknown date"
  }</p>
        <div class="post-content">${post.Content}</div>
        
        <div class="post-actions">
          <button class="like-button" data-id="${
            post.ID
          }" data-type="like" data-for="post">
            👍 <span>${post.Likes || 0}</span>
          </button>
          <button class="dislike-button" data-id="${
            post.ID
          }" data-type="dislike" data-for="post">
            👎 <span>${post.Dislikes || 0}</span>
          </button>
        </div>
        
        <h3>Comments (${post.Comments ? post.Comments.length : 0})</h3>
        <div id="comments-container">
          ${getCommentsHTML(post.Comments)}
        </div>
        
        ${getCommentFormHTML(post.ID)}
      </div>
    `;
}

// Generate HTML for comments
function getCommentsHTML(comments) {
  if (!comments || comments.length === 0) {
    return "<p>No comments yet. Be the first to comment!</p>";
  }

  return comments
    .map(
      (comment) => `
      <div class="comment">
        <p class="comment-author"><strong>${
          comment.Username
        }</strong> commented:</p>
        <p class="comment-content">${comment.Content}</p>
        <div class="comment-actions">
          <button class="like-button" data-id="${
            comment.ID
          }" data-type="like" data-for="comment">
            👍 <span>${comment.Likes || 0}</span>
          </button>
          <button class="dislike-button" data-id="${
            comment.ID
          }" data-type="dislike" data-for="comment">
            👎 <span>${comment.Dislikes || 0}</span>
          </button>
        </div>
      </div>
    `
    )
    .join("");
}

// Generate HTML for comment form
function getCommentFormHTML(postId) {
  if (state.sessionID) {
    return `
        <form id="comment-form">
          <h4>Add a Comment</h4>
          <input type="hidden" name="post_id" value="${postId}">
          <textarea name="content" class="comment-box" placeholder="Write your comment here..." required></textarea>
          <button type="submit">Submit Comment</button>
        </form>
      `;
  } else {
    return `
        <div class="login-prompt">
          <p>You must be logged in to add a comment. 
             <a href="/login" data-navigate>Login</a> or 
             <a href="/register" data-navigate>Register</a>
          </p>
        </div>
      `;
  }
}

// Fetch filtered posts
async function fetchFilteredPosts(queryString) {
  const contentElement = document.getElementById("content");

  try {
    const response = await fetch(`/filter${queryString}`, {
      headers: { Accept: "application/json" },
    });

    if (!response.ok) {
      throw new Error(`HTTP error! Status: ${response.status}`);
    }

    const data = await response.json();
    state.posts = data.posts || [];

    // Determine filter title
    let title = "Filtered Posts";
    if (data.category) {
      title = `${data.category} Posts`;
    } else if (queryString.includes("user_created=true")) {
      title = "My Posts";
    } else if (queryString.includes("liked=true")) {
      title = "Liked Posts";
    }

    // Display posts
    if (state.posts.length > 0) {
      contentElement.innerHTML = `
          <h2>${title}</h2>
          ${state.posts
            .map(
              (post) => `
            <div class="post">
              <h2><a href="/post?id=${post.id || post.ID}" data-navigate>${
                post.title || post.Title
              }</a></h2>
              <p>Category: ${post.category || post.Category}</p>
            </div>
          `
            )
            .join("")}
        `;
    } else {
      contentElement.innerHTML = `
          <h2>${title}</h2>
          <p>No posts found.</p>
        `;
    }
  } catch (error) {
    console.error("Error fetching filtered posts:", error);
    contentElement.innerHTML =
      '<div class="error">Failed to load posts. Please try again later.</div>';
  }
}

// Show login form
function showLoginForm() {
  const contentElement = document.getElementById("content");

  contentElement.innerHTML = `
      <div class="auth-form">
        <h1>Login</h1>
        <div id="login-error" class="error" style="display: none;"></div>
        <form id="login-form">
          <label for="email">Email Address:</label>
          <input type="email" id="email" name="email" required placeholder="Enter your email">
          
          <label for="password">Password:</label>
          <input type="password" id="password" name="password" required placeholder="Enter your password">
          
          <button type="submit">Login</button>
        </form>
        <p>Don't have an account? <a href="/register" data-navigate>Register here</a></p>
        <div class="category-button">
          <a href="/" data-navigate class="back-to-home">Back to Home</a>
        </div>
      </div>
    `;

  document.getElementById("login-form").addEventListener("submit", handleLogin);
}

// Show registration form
function showRegisterForm() {
  const contentElement = document.getElementById("content");

  contentElement.innerHTML = `
      <div class="auth-form">
        <h1>Create Account</h1>
        <div id="register-error" class="error" style="display: none;"></div>
        <form id="register-form">
          <label for="username">Username:</label>
          <input type="text" id="username" name="username" required placeholder="Choose a username">
          
          <label for="email">Email Address:</label>
          <input type="email" id="email" name="email" required placeholder="Enter your email">
          
          <label for="password">Password:</label>
          <input type="password" id="password" name="password" required placeholder="Create a password">
          
          <button type="submit">Register</button>
        </form>
        <p>Already have an account? <a href="/login" data-navigate>Login here</a></p>
        <div class="category-button">
          <a href="/" data-navigate class="back-to-home">Back to Home</a>
        </div>
      </div>
    `;

  document
    .getElementById("register-form")
    .addEventListener("submit", handleRegister);
}

// Show create post form
function showCreatePostForm() {
  const contentElement = document.getElementById("content");

  contentElement.innerHTML = `
      <div class="form-container">
        <h2>Create a New Post</h2>
        <form id="create-post-form">
          <label for="title">Title:</label>
          <input type="text" id="title" name="title" required placeholder="Enter a descriptive title">
          
          <label for="content">Content:</label>
          <textarea id="content" name="content" required placeholder="Write your post content here..."></textarea>
          
          <label>Categories:</label>
          <div class="checkbox-container">
            <label><input type="checkbox" name="categories" value="General"> General</label>
            <label><input type="checkbox" name="categories" value="Local News & Events"> Local News & Events</label>
            <label><input type="checkbox" name="categories" value="Viking line"> Viking Line</label>
            <label><input type="checkbox" name="categories" value="Travel"> Travel</label>
            <label><input type="checkbox" name="categories" value="Sailing"> Sailing</label>
            <label><input type="checkbox" name="categories" value="Cuisine & food"> Cuisine & Food</label>
            <label><input type="checkbox" name="categories" value="Politics"> Politics</label>
          </div>
          
          <button type="submit">Publish Post</button>
        </form>
        <div class="category-button" style="margin-top: 20px;">
          <a href="/" data-navigate class="back-to-home">Cancel</a>
        </div>
      </div>
    `;

  document
    .getElementById("create-post-form")
    .addEventListener("submit", handleCreatePost);
}

// Show error message
function showError(message) {
  const contentElement = document.getElementById("content");

  contentElement.innerHTML = `
      <div class="error-container">
        <h1>Error</h1>
        <p>${message}</p>
        <div class="category-button">
          <a href="/" data-navigate class="back-to-home">Back to Home</a>
        </div>
      </div>
    `;
}

// Handle login form submission
async function handleLogin(event) {
  event.preventDefault();

  const form = event.target;
  const formData = new FormData(form);

  try {
    const response = await fetch("/login", {
      method: "POST",
      body: formData,
    });

    const data = await response.json();

    if (response.ok) {
      // Login successful
      state.sessionID = data.sessionID;
      state.username = data.username;

      updateAuthBox();
      updateSidebar();
      navigateTo("/");
    } else {
      // Show error message
      const errorElement = document.getElementById("login-error");
      errorElement.textContent = data.message;
      errorElement.style.display = "block";
    }
  } catch (error) {
    console.error("Login error:", error);
    document.getElementById("login-error").textContent =
      "An error occurred. Please try again.";
    document.getElementById("login-error").style.display = "block";
  }
}

// Handle register form submission
async function handleRegister(event) {
  event.preventDefault();

  const form = event.target;
  const formData = new FormData(form);

  try {
    const response = await fetch("/register", {
      method: "POST",
      body: formData,
    });

    const data = await response.json();

    if (response.ok) {
      // Registration successful
      navigateTo("/login");
    } else {
      // Show error message
      const errorElement = document.getElementById("register-error");
      errorElement.textContent = data.message;
      errorElement.style.display = "block";
    }
  } catch (error) {
    console.error("Registration error:", error);
    document.getElementById("register-error").textContent =
      "An error occurred. Please try again.";
    document.getElementById("register-error").style.display = "block";
  }
}

// Handle create post submission
async function handleCreatePost(event) {
  event.preventDefault();

  const form = event.target;
  const formData = new FormData(form);

  try {
    const response = await fetch("/createPost", {
      method: "POST",
      body: formData,
    });

    const data = await response.json();

    if (response.ok) {
      // Successfully created post
      setTimeout(() => {
        navigateTo(`/post?id=${data.id}`);
      }, 300);
    } else {
      alert(data.message || "Failed to create post");
    }
  } catch (error) {
    console.error("Create post error:", error);
    alert("An error occurred. Please try again.");
  }
}

// Handle adding comments
async function handleAddComment(event) {
  event.preventDefault();

  const form = event.target;
  const formData = new FormData(form);
  const postId = formData.get("post_id");

  try {
    const response = await fetch("/comment", {
      method: "POST",
      body: formData,
    });

    if (response.ok) {
      // Comment added, refresh the post
      fetchSinglePost(postId);
    } else {
      const data = await response.json();
      alert(data.message || "Failed to add comment");
    }
  } catch (error) {
    console.error("Add comment error:", error);
    alert("An error occurred. Please try again.");
  }
}

// Handle logout
async function handleLogout(event) {
  event.preventDefault();

  try {
    const response = await fetch("/logout", {
      method: "POST",
    });

    if (response.ok) {
      // Logout successful
      state.sessionID = null;
      state.username = null;

      updateAuthBox();
      updateSidebar();
      navigateTo("/");
    }
  } catch (error) {
    console.error("Logout error:", error);
  }
}

// Setup reaction buttons
function setupReactionButtons() {
  document
    .querySelectorAll(".like-button, .dislike-button")
    .forEach((button) => {
      button.addEventListener("click", () => handleReaction(button));
    });
}

// Handle like/dislike reactions
async function handleReaction(button) {
  // Check if user is logged in
  if (!state.sessionID) {
    navigateTo("/login");
    return;
  }

  const itemId = button.getAttribute("data-id");
  const isComment = button.getAttribute("data-for") === "comment";
  const reactionType = button.getAttribute("data-type");

  const formData = new FormData();
  formData.append("item_id", itemId);
  formData.append("is_comment", isComment);
  formData.append("type", reactionType);

  try {
    const response = await fetch("/like", {
      method: "POST",
      body: formData,
    });

    if (response.ok) {
      // Reaction recorded, refresh the current view
      if (window.location.pathname === "/post") {
        const postId = new URLSearchParams(window.location.search).get("id");
        fetchSinglePost(postId);
      } else {
        fetchPosts();
      }
    }
  } catch (error) {
    console.error("Reaction error:", error);
  }
}
