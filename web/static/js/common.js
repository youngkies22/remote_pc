// Utilitas bersama untuk seluruh halaman dashboard.

// api melakukan fetch JSON ke endpoint REST dengan menyertakan cookie sesi.
async function api(path, options = {}) {
  const res = await fetch(path, {
    credentials: "same-origin",
    headers: { "Content-Type": "application/json" },
    ...options,
  });
  if (res.status === 401) {
    window.location.href = "/login";
    throw new Error("unauthorized");
  }
  const text = await res.text();
  const data = text ? JSON.parse(text) : null;
  if (!res.ok) {
    throw new Error((data && data.error) || "request gagal");
  }
  return data;
}

// logout mengakhiri sesi lalu mengarahkan ke halaman login.
async function logout() {
  try {
    await api("/api/logout", { method: "POST" });
  } catch (_) {
    /* diabaikan */
  }
  window.location.href = "/login";
}

// escapeHtml mencegah injeksi HTML dari nilai yang berasal dari agent.
function escapeHtml(value) {
  return String(value == null ? "" : value)
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;");
}

// timeAgo memformat timestamp menjadi teks relatif berbahasa Indonesia.
function timeAgo(iso) {
  if (!iso) return "-";
  const then = new Date(iso).getTime();
  if (!then) return "-";
  const secs = Math.max(0, Math.floor((Date.now() - then) / 1000));
  if (secs < 10) return "baru saja";
  if (secs < 60) return `${secs} dtk lalu`;
  const mins = Math.floor(secs / 60);
  if (mins < 60) return `${mins} mnt lalu`;
  const hours = Math.floor(mins / 60);
  if (hours < 24) return `${hours} jam lalu`;
  return `${Math.floor(hours / 24)} hari lalu`;
}
