// Halaman "/version": tampilkan info build server (commit & waktu compile)
// supaya admin bisa memastikan deployment memang memakai kode terbaru.

function kvRow(label, value) {
  return `<div class="k">${escapeHtml(label)}</div><div>${escapeHtml(value)}</div>`;
}

function fmtBuildTime(iso) {
  if (!iso || iso === "unknown") return "tidak diketahui (build tanpa info waktu)";
  const d = new Date(iso);
  if (isNaN(d.getTime())) return iso;
  return d.toLocaleString("id-ID", { dateStyle: "full", timeStyle: "medium" });
}

async function loadVersion() {
  const body = document.getElementById("version-body");
  try {
    const v = await api("/api/version");
    body.innerHTML = [
      kvRow("Versi Aplikasi", v.app_version || "dev"),
      kvRow("Waktu Build", fmtBuildTime(v.build_time)),
      kvRow("Git Commit", v.git_commit || "unknown"),
      kvRow("Versi Go", v.go_version || "-"),
    ].join("");
  } catch (err) {
    body.innerHTML = kvRow("Error", err.message);
  }
}

async function init() {
  try {
    const me = await api("/api/me");
    document.getElementById("whoami").textContent = me.username + " (" + me.role + ")";
  } catch (_) { /* redirect ditangani api() */ }
  loadVersion();
}

init();
