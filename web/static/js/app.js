// app.js - Main application file for SPA

// Application state
const state = {
  sessionID: null,
  username: null,
  posts: [],
  currentPost: null,
  currentRoute: window.location.pathname,
};

// Initialize the app
document.addEventListener("DOMContentLoaded", init);

async function init() {
  // Check if user is logged in
  await checkUserStatus();

  // Add navigation event listeners
  setupNavigation();

  // Handle the current route
  handleRoute();
}

// Check user login status
async function checkUserStatus() {
  try {
    const response = await fetch("/user/status");
    const data = await response.json();

    state.sessionID = data.sessionID;
    state.username = data.username;

    // Update the auth box with user info
    updateAuthBox();
    updateSidebar();
  } catch (error) {
    console.error("Error checking user status:", error);
  }
}

// Update the auth box based on login status
function updateAuthBox() {
  const authBox = document.getElementById("auth-box");

  if (state.sessionID && state.username) {
    // User is logged in
    authBox.innerHTML = `
        <p>Logged in as: <strong>${state.username}</strong></p>
        <a href="/createPost" data-navigate>Create Post</a>
        <a href="#" id="logout-link">Logout</a>
      `;

    // Add logout event listener
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

// Update the sidebar based on login status
function updateSidebar() {
  const sidebar = document.getElementById("sidebar");

  sidebar.innerHTML = `
      <h2>Categories:</h2>
      <p><a href="/filter?category=General" data-navigate>General</a></p>
      <p><a href="/filter?category=Local%20News%20%26%20Events" data-navigate>Local News & Events</a></p>
      <p><a href="/filter?category=Viking%20line" data-navigate>Viking line</a></p>
      <p><a href="/filter?category=Travel" data-navigate>Travel</a></p>
      <p><a href="/filter?category=Sailing" data-navigate>Sailing</a></p>
      <p><a href="/filter?category=Cuisine%20%26%20food" data-navigate>Cuisine & food</a></p>
      <p><a href="/filter?category=Politics" data-navigate>Politics</a></p>
      ${
        state.sessionID
          ? `
        <h2>Filters:</h2>
        <p><a href="/filter?user_created=true" data-navigate>My Posts</a></p>
        <p><a href="/filter?liked=true" data-navigate>Liked Posts</a></p>
      `
          : ""
      }
      <div class="category-button">
        <a href="/" data-navigate class="back-to-home">Back to Home</a>
      </div>
    `;
}

// Setup navigation for SPA
function setupNavigation() {
  // Handle all links with data-navigate attribute
  document.addEventListener("click", (event) => {
    const link = event.target.closest("[data-navigate]");
    if (link) {
      event.preventDefault();
      const url = link.getAttribute("href");
      navigateTo(url);
    }
  });

  // Handle browser back/forward buttons
  window.addEventListener("popstate", () => {
    handleRoute();
  });
}

// Navigate to a new route
function navigateTo(url) {
  // Update the URL
  history.pushState(null, "", url);

  // Handle the new route
  handleRoute();
}

// Handle the current route
function handleRoute() {
  const path = window.location.pathname;
  const searchParams = new URLSearchParams(window.location.search);

  // Route handling
  if (path === "/" || path === "/index.html") {
    fetchPosts();
  } else if (path === "/post") {
    const postId = searchParams.get("id");
    if (postId) {
      fetchSinglePost(postId);
    } else {
      showError("Post ID is missing");
    }
  } else if (path === "/filter") {
    const category = searchParams.get("category");
    const userCreated = searchParams.get("user_created");
    const liked = searchParams.get("liked");

    if (category || userCreated || liked) {
      fetchFilteredPosts(window.location.search);
    } else {
      showError("Invalid filter parameters");
    }
  } else if (path === "/login") {
    showLoginForm();
  } else if (path === "/register") {
    showRegisterForm();
  } else if (path === "/createPost") {
    if (state.sessionID) {
      showCreatePostForm();
    } else {
      // Redirect to login if not authenticated
      navigateTo("/login");
    }
  } else {
    // 404 not found
    showError("Page not found");
  }
}

// Fetch posts for the homepage
async function fetchPosts() {
  const contentElement = document.getElementById("content");
  contentElement.innerHTML = '<div class="loading">Loading posts...</div>';

  try {
    const response = await fetch("/");
    const data = await response.json();

    state.posts = data.Posts || [];

    if (state.posts.length > 0) {
      contentElement.innerHTML = `
          <h2>All posts</h2>
          ${state.posts
            .map(
              (post) => `
            <div class="post">
              <h2><a href="/post?id=${post.ID}" data-navigate>${post.Title}</a></h2>
              <p>${post.Content}</p>
              <div class="post-meta">
                <div class="left">
                  <span class="username">${post.Username}</span>
                  <span class="date">${post.Date}</span>
                </div>
                <div class="right">
                  <button class="like-button" data-id="${post.ID}" data-type="like" data-for="post">👍 ${post.Likes}</button>
                  <button class="dislike-button" data-id="${post.ID}" data-type="dislike" data-for="post">👎 ${post.Dislikes}</button>
                </div>
              </div>
            </div>
          `
            )
            .join("")}
        `;

      // Add event listeners for like/dislike buttons
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

// Fetch a single post
async function fetchSinglePost(postId) {
  const contentElement = document.getElementById("content");
  contentElement.innerHTML = '<div class="loading">Loading post...</div>';

  try {
    const response = await fetch(`/post?id=${postId}`);
    const data = await response.json();

    state.currentPost = data.Post;

    if (state.currentPost) {
      contentElement.innerHTML = `
          <div class="post" id="post-${state.currentPost.ID}">
            <h2>Category: ${state.currentPost.Category}</h2>
            <h3>${state.currentPost.Title}</h3>
            <p>${state.currentPost.Username}:</p>
            <p>${state.currentPost.Content}</p>
  
            <div class="post-actions">
              <button class="like-button" data-id="${
                state.currentPost.ID
              }" data-type="like" data-for="post">
                👍 <span>${state.currentPost.Likes}</span>
              </button>
              <button class="dislike-button" data-id="${
                state.currentPost.ID
              }" data-type="dislike" data-for="post">
                👎 <span>${state.currentPost.Dislikes}</span>
              </button>
            </div>
  
            <h3>Comments</h3>
            <div id="comments-container">
              ${
                state.currentPost.Comments &&
                state.currentPost.Comments.length > 0
                  ? state.currentPost.Comments.map(
                      (comment) => `
                  <div class="comment">
                    <p>${comment.Username}:</p>
                    <p>${comment.Content}</p>
                    <div class="comment-actions">
                      <button class="like-button" data-id="${comment.ID}" data-type="like" data-for="comment">
                        👍 <span>${comment.Likes}</span>
                      </button>
                      <button class="dislike-button" data-id="${comment.ID}" data-type="dislike" data-for="comment">
                        👎 <span>${comment.Dislikes}</span>
                      </button>
                    </div>
                  </div>
                `
                    ).join("")
                  : "<p>No comments yet.</p>"
              }
            </div>
  
            ${
              state.sessionID
                ? `<form id="comment-form">
                  <input type="hidden" name="post_id" value="${state.currentPost.ID}">
                  <textarea name="content" class="comment-box" placeholder="Add a comment..." required></textarea>
                  <button type="submit">Add Comment</button>
                </form>`
                : `<p>You must be logged in to add a comment. <a href="/login" data-navigate>Login</a></p>`
            }
          </div>
        `;

      // Setup reaction buttons
      setupReactionButtons();

      // Setup comment form if logged in
      if (state.sessionID) {
        document
          .getElementById("comment-form")
          .addEventListener("submit", handleAddComment);
      }
    } else {
      contentElement.innerHTML = '<div class="error">Post not found.</div>';
    }
  } catch (error) {
    console.error("Error fetching post:", error);
    contentElement.innerHTML =
      '<div class="error">Failed to load post. Please try again later.</div>';
  }
}

// Fetch filtered posts
async function fetchFilteredPosts(queryString) {
  const contentElement = document.getElementById("content");
  contentElement.innerHTML = '<div class="loading">Loading posts...</div>';

  try {
    // Build the query string
    const response = await fetch(`/filter${queryString}`);
    const data = await response.json();

    state.posts = data.posts || [];

    let title = "";
    if (data.category) {
      title = `${data.category} Posts`;
    } else if (queryString.includes("user_created=true")) {
      title = "My Posts";
    } else if (queryString.includes("liked=true")) {
      title = "Liked Posts";
    } else {
      title = "Filtered Posts";
    }

    if (state.posts.length > 0) {
      contentElement.innerHTML = `
          <h2>${title}</h2>
          ${state.posts
            .map(
              (post) => `
            <div class="post">
              <h2><a href="/post?id=${post.ID}" data-navigate>${post.Title}</a></h2>
              <p>Category: ${post.Category}</p>
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
          <label for="email">Email:</label>
          <input type="email" id="email" name="email" required>
          
          <label for="password">Password:</label>
          <input type="password" id="password" name="password" required>
          
          <button type="submit">Login</button>
        </form>
        <p>Don't have an account? <a href="/register" data-navigate>Register here</a></p>
      </div>
    `;

  // Add form submit event listener
  document.getElementById("login-form").addEventListener("submit", handleLogin);
}

// Show register form
function showRegisterForm() {
  const contentElement = document.getElementById("content");

  contentElement.innerHTML = `
      <div class="auth-form">
        <h1>Register</h1>
        <div id="register-error" class="error" style="display: none;"></div>
        <form id="register-form">
          <label for="username">Username:</label>
          <input type="text" id="username" name="username" required>
          
          <label for="email">Email:</label>
          <input type="email" id="email" name="email" required>
          
          <label for="password">Password:</label>
          <input type="password" id="password" name="password" required>
          
          <button type="submit">Register</button>
        </form>
        <p>Already have an account? <a href="/login" data-navigate>Login here</a></p>
      </div>
    `;

  // Add form submit event listener
  document
    .getElementById("register-form")
    .addEventListener("submit", handleRegister);
}

// Show create post form
function showCreatePostForm() {
  const contentElement = document.getElementById("content");

  contentElement.innerHTML = `
      <div class="form-container">
        <h2>Create a Post</h2>
        <form id="create-post-form">
          <label for="title">Title:</label>
          <input type="text" id="title" name="title" required>
          
          <label for="content">Content:</label>
          <textarea id="content" name="content" required></textarea>
          
          <div class="checkbox-container">
            <label><input type="checkbox" name="categories" value="General"> General</label>
            <label><input type="checkbox" name="categories" value="Local News & Events"> Local News & Events</label>
            <label><input type="checkbox" name="categories" value="Viking line"> Viking Line</label>
            <label><input type="checkbox" name="categories" value="Travel"> Travel</label>
            <label><input type="checkbox" name="categories" value="Sailing"> Sailing</label>
            <label><input type="checkbox" name="categories" value="Cuisine & food"> Cuisine & Food</label>
            <label><input type="checkbox" name="categories" value="Politics"> Politics</label>
          </div>
          
          <button type="submit">Post</button>
        </form>
      </div>
    `;

  // Add form submit event listener
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
  const email = form.email.value;
  const password = form.password.value;

  // Create form data
  const formData = new FormData();
  formData.append("email", email);
  formData.append("password", password);

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

      // Update UI
      updateAuthBox();
      updateSidebar();

      // Redirect to home page
      navigateTo("/");
    } else {
      // Show error message
      const errorElement = document.getElementById("login-error");
      errorElement.textContent = data.message;
      errorElement.style.display = "block";
    }
  } catch (error) {
    console.error("Login error:", error);
    const errorElement = document.getElementById("login-error");
    errorElement.textContent = "An error occurred. Please try again.";
    errorElement.style.display = "block";
  }
}

// Handle register form submission
async function handleRegister(event) {
  event.preventDefault();

  const form = event.target;
  const username = form.username.value;
  const email = form.email.value;
  const password = form.password.value;

  // Create form data
  const formData = new FormData();
  formData.append("username", username);
  formData.append("email", email);
  formData.append("password", password);

  try {
    const response = await fetch("/register", {
      method: "POST",
      body: formData,
    });

    const data = await response.json();

    if (response.ok) {
      // Registration successful, redirect to login
      navigateTo("/login");
    } else {
      // Show error message
      const errorElement = document.getElementById("register-error");
      errorElement.textContent = data.message;
      errorElement.style.display = "block";
    }
  } catch (error) {
    console.error("Registration error:", error);
    const errorElement = document.getElementById("register-error");
    errorElement.textContent = "An error occurred. Please try again.";
    errorElement.style.display = "block";
  }
}

// Handle create post form submission
async function handleCreatePost(event) {
  event.preventDefault();

  const form = event.target;
  const title = form.title.value;
  const content = form.content.value;

  // Get selected categories
  const categoriesInputs = form.querySelectorAll(
    'input[name="categories"]:checked'
  );
  const categories = Array.from(categoriesInputs).map((input) => input.value);

  // Create form data
  const formData = new FormData();
  formData.append("title", title);
  formData.append("content", content);

  // Add categories
  categories.forEach((category) => {
    formData.append("categories", category);
  });

  try {
    const response = await fetch("/createPost", {
      method: "POST",
      body: formData,
    });

    const data = await response.json();

    if (response.ok) {
      // Post created successfully, navigate to the post
      navigateTo(`/post?id=${data.id}`);
    } else {
      alert(data.message || "Failed to create post");
    }
  } catch (error) {
    console.error("Create post error:", error);
    alert("An error occurred. Please try again.");
  }
}

// Handle adding a comment
async function handleAddComment(event) {
  event.preventDefault();

  const form = event.target;
  const postId = form.post_id.value;
  const content = form.content.value;

  // Create form data
  const formData = new FormData();
  formData.append("post_id", postId);
  formData.append("content", content);

  try {
    const response = await fetch("/comment", {
      method: "POST",
      body: formData,
    });

    const data = await response.json();

    if (response.ok) {
      // Comment added successfully, refresh the post
      fetchSinglePost(postId);
    } else {
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

      // Update UI
      updateAuthBox();
      updateSidebar();

      // Redirect to home page
      navigateTo("/");
    } else {
      console.error("Logout failed");
    }
  } catch (error) {
    console.error("Logout error:", error);
  }
}

// Setup reaction buttons (like/dislike)
function setupReactionButtons() {
  const likeButtons = document.querySelectorAll(".like-button");
  const dislikeButtons = document.querySelectorAll(".dislike-button");

  // Add event listeners for like buttons
  likeButtons.forEach((button) => {
    button.addEventListener("click", () => handleReaction(button));
  });

  // Add event listeners for dislike buttons
  dislikeButtons.forEach((button) => {
    button.addEventListener("click", () => handleReaction(button));
  });
}

// Handle a reaction (like/dislike)
async function handleReaction(button) {
  // Check if user is logged in
  if (!state.sessionID) {
    navigateTo("/login");
    return;
  }

  const itemId = button.getAttribute("data-id");
  const isComment = button.getAttribute("data-for") === "comment";
  const reactionType = button.getAttribute("data-type");

  // Create form data
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
      // Reaction recorded successfully, refresh the current view
      if (window.location.pathname === "/post") {
        const postId = new URLSearchParams(window.location.search).get("id");
        fetchSinglePost(postId);
      } else {
        fetchPosts();
      }
    } else {
      console.error("Reaction failed");
    }
  } catch (error) {
    console.error("Reaction error:", error);
  }
}
