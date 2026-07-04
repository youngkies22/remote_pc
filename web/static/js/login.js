// Menangani submit form login.
document.getElementById("loginForm").addEventListener("submit", async (e) => {
  e.preventDefault();
  const errorBox = document.getElementById("error");
  errorBox.classList.remove("error");
  errorBox.textContent = "";

  const username = document.getElementById("username").value.trim();
  const password = document.getElementById("password").value;

  try {
    await api("/api/login", {
      method: "POST",
      body: JSON.stringify({ username, password }),
    });
    window.location.href = "/";
  } catch (err) {
    errorBox.textContent = err.message || "Login gagal";
    errorBox.classList.add("error");
  }
});
