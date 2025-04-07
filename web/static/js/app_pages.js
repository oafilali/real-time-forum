// Load the home page with all posts
async function loadHomePage() {
  try {
    const response = await fetch("/?api=true", {
      headers: {
        Accept: "application/json",
        "X-Requested-With": "XMLHttpRequest",
      },
    });

    if (!response.ok) {
      throw new Error("Failed to load posts");
    }

    const data = await response.json();
    window.state.posts = data.Posts || [];

    document.getElementById("content").innerHTML = templates.homePage(
      window.state.posts
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

    if (!response.ok) {
      throw new Error("Failed to load post");
    }

    const data = await response.json();
    window.state.currentPost = data.post || data.Post;

    document.getElementById("content").innerHTML = templates.postDetail(
      window.state.currentPost
    );

    // Setup reaction buttons
    setupReactionButtons();

    // Setup comment form submission
    const commentForm = document.getElementById("comment-form");
    if (commentForm) {
      commentForm.addEventListener("submit", window.appForms.submitComment);
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

    if (!response.ok) {
      throw new Error("Failed to load filtered posts");
    }

    const data = await response.json();
    window.state.posts = data.posts || [];

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
      window.state.posts
    );
  } catch (error) {
    console.error("Error:", error);
    document.getElementById("content").innerHTML = templates.error(
      "Failed to load filtered posts"
    );
  }
}

// Show login page
function showLoginPage() {
  document.getElementById("content").innerHTML = templates.loginForm();
  document
    .getElementById("login-form")
    .addEventListener("submit", window.appForms.submitLogin);
}

// Show register page
function showRegisterPage() {
  document.getElementById("content").innerHTML = templates.registerForm();
  document
    .getElementById("register-form")
    .addEventListener("submit", window.appForms.submitRegister);
}

// Show create post page
function showCreatePostPage() {
  document.getElementById("content").innerHTML = templates.createPostForm();
  document
    .getElementById("create-post-form")
    .addEventListener("submit", window.appForms.submitPost);
}

// Show error page
function showErrorPage(message) {
  document.getElementById("content").innerHTML = templates.error(message);
}

// Set up reaction buttons (likes/dislikes)
function setupReactionButtons() {
  const reactionButtons = document.querySelectorAll(
    ".like-button, .dislike-button"
  );

  reactionButtons.forEach((button) => {
    button.addEventListener("click", function () {
      window.appForms.submitReaction(this);
    });
  });
}

// Expose functions to global scope
window.appPages = {
  loadHomePage,
  loadPostPage,
  loadFilteredPosts,
  showLoginPage,
  showRegisterPage,
  showCreatePostPage,
  showErrorPage,
  setupReactionButtons,
};
