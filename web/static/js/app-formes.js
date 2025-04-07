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
      window.state.sessionID = data.sessionID;
      window.state.username = data.username;
      window.appCore.updateUI();

      // Start WebSocket connection after successful login
      if (window.chatConnection) {
        window.chatConnection.connect();
        window.chatConnection.startChecking();
      }

      window.appCore.navigate("/");
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
      window.appCore.navigate("/login");
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
      setTimeout(() => window.appCore.navigate(`/post?id=${data.id}`), 300);
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
      window.appPages.loadPostPage(postId);
    } else {
      const data = await response.json();
      alert(data.message || "Failed to add comment");
    }
  } catch (error) {
    console.error("Comment error:", error);
    alert("An error occurred. Please try again.");
  }
}

// Handle reaction submission (like/dislike)
async function submitReaction(button) {
  // Check if user is logged in
  if (!window.state.sessionID) {
    window.appCore.navigate("/login");
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
        window.appPages.loadPostPage(postId);
      } else {
        window.appPages.loadHomePage();
      }
    }
  } catch (error) {
    console.error("Reaction error:", error);
  }
}

// Expose functions to global scope
window.appForms = {
  submitLogin,
  submitRegister,
  submitPost,
  submitComment,
  submitReaction,
};
