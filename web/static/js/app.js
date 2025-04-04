// app.js - Main application for Forum SPA

// Global application state
const state = {
  sessionID: null,
  username: null,
  posts: [],
  currentPost: null,
};

// Make state available for templates
window.state = state;

// Initialize the application
document.addEventListener("DOMContentLoaded", async function () {
  await checkLogin();
  setupNavigationEvents();
  loadCurrentPage();
});

// AUTH & USER FUNCTIONS
// ---------------------

// Check if user is logged in
async function checkLogin() {
  try {
    const response = await fetch("/user/status");
    const data = await response.json();
    state.sessionID = data.sessionID;
    state.username = data.username;
    updateUI();
  } catch (error) {
    console.error("Login check failed:", error);
  }
}

// Update UI elements after login status change
function updateUI() {
  // Update auth box
  document.getElementById("auth-box").innerHTML = templates.authBox(
    state.sessionID,
    state.username
  );

  // Add logout event listener if logged in
  if (state.sessionID) {
    document
      .getElementById("logout-link")
      .addEventListener("click", handleLogout);
  }

  // Update sidebar
  document.getElementById("sidebar").innerHTML = templates.sidebar(
    state.sessionID
  );
}

// Handle logout
async function handleLogout(event) {
  event.preventDefault();

  try {
    const response = await fetch("/logout", { method: "POST" });
    if (response.ok) {
      state.sessionID = null;
      state.username = null;
      updateUI();
      navigate("/");
    }
  } catch (error) {
    console.error("Logout failed:", error);
  }
}

// NAVIGATION FUNCTIONS
// -------------------

// Set up navigation event listeners
function setupNavigationEvents() {
  // Handle link clicks with data-navigate attribute
  document.addEventListener("click", function (event) {
    const link = event.target.closest("[data-navigate]");
    if (link) {
      event.preventDefault();
      navigate(link.getAttribute("href"));
    }
  });

  // Handle browser back/forward
  window.addEventListener("popstate", loadCurrentPage);
}

// Navigate to a new page
function navigate(url) {
  history.pushState(null, "", url);
  loadCurrentPage();
}

// Determine what to show based on current URL
function loadCurrentPage() {
  const path = window.location.pathname;
  const searchParams = new URLSearchParams(window.location.search);
  const content = document.getElementById("content");

  // Show loading indicator
  content.innerHTML = templates.loading();

  // Route to correct page handler
  if (path === "/" || path === "/index.html") {
    loadHomePage();
  } else if (path === "/post") {
    const postId = searchParams.get("id");
    if (postId && postId !== "undefined") {
      loadPostPage(postId);
    } else {
      showErrorPage("Post ID is missing or invalid");
    }
  } else if (path === "/filter") {
    loadFilteredPosts(window.location.search);
  } else if (path === "/login") {
    showLoginPage();
  } else if (path === "/register") {
    showRegisterPage();
  } else if (path === "/createPost") {
    if (state.sessionID) {
      showCreatePostPage();
    } else {
      navigate("/login");
    }
  } else {
    showErrorPage("Page not found");
  }
}

// PAGE LOADERS
// ------------

// Load the home page with all posts
async function loadHomePage() {
  try {
    const response = await fetch("/?api=true", {
      headers: {
        Accept: "application/json",
        "X-Requested-With": "XMLHttpRequest",
      },
    });

    if (!response.ok) throw new Error("Failed to load posts");

    const data = await response.json();
    state.posts = data.Posts || [];

    document.getElementById("content").innerHTML = templates.homePage(
      state.posts
    );
    setupReactionButtons();
  } catch (error) {
    console.error("Error:", error);
    document.getElementById("content").innerHTML = templates.error(
      "Failed to load posts"
    );
  }
}

// Load a single post page
async function loadPostPage(postId) {
  try {
    const response = await fetch(`/post?id=${postId}`, {
      headers: { Accept: "application/json" },
    });

    if (!response.ok) throw new Error("Failed to load post");

    const data = await response.json();
    state.currentPost = data.post || data.Post;

    document.getElementById("content").innerHTML = templates.postDetail(
      state.currentPost
    );

    // Setup reaction buttons
    setupReactionButtons();

    // Setup comment form submission
    const commentForm = document.getElementById("comment-form");
    if (commentForm) {
      commentForm.addEventListener("submit", submitComment);
    }
  } catch (error) {
    console.error("Error:", error);
    document.getElementById("content").innerHTML = templates.error(
      "Failed to load post"
    );
  }
}

// Load filtered posts
async function loadFilteredPosts(queryString) {
  try {
    const response = await fetch(`/filter${queryString}`, {
      headers: { Accept: "application/json" },
    });

    if (!response.ok) throw new Error("Failed to load filtered posts");

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

    document.getElementById("content").innerHTML = templates.filteredPosts(
      title,
      state.posts
    );
  } catch (error) {
    console.error("Error:", error);
    document.getElementById("content").innerHTML = templates.error(
      "Failed to load filtered posts"
    );
  }
}

// FORM PAGES
// ----------

// Show login page
function showLoginPage() {
  document.getElementById("content").innerHTML = templates.loginForm();
  document.getElementById("login-form").addEventListener("submit", submitLogin);
}

// Show register page
function showRegisterPage() {
  document.getElementById("content").innerHTML = templates.registerForm();
  document
    .getElementById("register-form")
    .addEventListener("submit", submitRegister);
}

// Show create post page
function showCreatePostPage() {
  document.getElementById("content").innerHTML = templates.createPostForm();
  document
    .getElementById("create-post-form")
    .addEventListener("submit", submitPost);
}

// Show error page
function showErrorPage(message) {
  document.getElementById("content").innerHTML = templates.error(message);
}

// FORM SUBMISSION HANDLERS
// -----------------------

// Handle login form submission
async function submitLogin(event) {
  event.preventDefault();

  const formData = new FormData(event.target);

  try {
    const response = await fetch("/login", {
      method: "POST",
      body: formData,
    });

    const data = await response.json();

    if (response.ok && data.sessionID) {
      // Login successful
      state.sessionID = data.sessionID;
      state.username = data.username;
      updateUI();
      navigate("/");
    } else {
      // Login failed - show error
      const errorElement = document.getElementById("login-error");
      errorElement.textContent = data.message || "Login failed";
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
async function submitRegister(event) {
  event.preventDefault();

  const formData = new FormData(event.target);

  // Basic validation
  const age = parseInt(formData.get("age"));
  if (isNaN(age) || age <= 0) {
    const errorElement = document.getElementById("register-error");
    errorElement.textContent = "Please enter a valid age";
    errorElement.style.display = "block";
    return;
  }

  if (!formData.get("gender")) {
    const errorElement = document.getElementById("register-error");
    errorElement.textContent = "Please select a gender";
    errorElement.style.display = "block";
    return;
  }

  try {
    const response = await fetch("/register", {
      method: "POST",
      body: formData,
    });

    const data = await response.json();

    if (response.ok) {
      // Registration successful
      alert("Registration successful! Please log in.");
      navigate("/login");
    } else {
      // Registration failed - show error
      const errorElement = document.getElementById("register-error");
      errorElement.textContent = data.message || "Registration failed";
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
async function submitPost(event) {
  event.preventDefault();

  const formData = new FormData(event.target);

  try {
    const response = await fetch("/createPost", {
      method: "POST",
      body: formData,
    });

    const data = await response.json();

    if (response.ok && data.id) {
      // Post created successfully
      setTimeout(() => navigate(`/post?id=${data.id}`), 300);
    } else {
      alert(data.message || "Failed to create post");
    }
  } catch (error) {
    console.error("Create post error:", error);
    alert("An error occurred. Please try again.");
  }
}

// Handle comment submission
async function submitComment(event) {
  event.preventDefault();

  const formData = new FormData(event.target);
  const postId = formData.get("post_id");

  try {
    const response = await fetch("/comment", {
      method: "POST",
      body: formData,
    });

    if (response.ok) {
      // Comment added successfully, reload post
      loadPostPage(postId);
    } else {
      const data = await response.json();
      alert(data.message || "Failed to add comment");
    }
  } catch (error) {
    console.error("Comment error:", error);
    alert("An error occurred. Please try again.");
  }
}

// INTERACTIVE ELEMENTS
// -------------------

// Set up reaction buttons (likes/dislikes)
function setupReactionButtons() {
  const reactionButtons = document.querySelectorAll(
    ".like-button, .dislike-button"
  );

  reactionButtons.forEach((button) => {
    button.addEventListener("click", function () {
      submitReaction(this);
    });
  });
}

// Handle reaction submission (like/dislike)
async function submitReaction(button) {
  // Check if user is logged in
  if (!state.sessionID) {
    navigate("/login");
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
      // Reaction successful, reload current view
      if (window.location.pathname === "/post") {
        const postId = new URLSearchParams(window.location.search).get("id");
        loadPostPage(postId);
      } else {
        loadHomePage();
      }
    }
  } catch (error) {
    console.error("Reaction error:", error);
  }
}
