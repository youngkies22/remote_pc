// Logika halaman dashboard: memuat statistik + daftar device dan auto-refresh.

const REFRESH_MS = 3000;

function meter(percent) {
  const p = Math.max(0, Math.min(100, Number(percent) || 0));
  return `<div class="meter" title="${p}%"><span style="width:${p}%"></span></div>`;
}

function deviceRow(d) {
  const badge = d.status === "online"
    ? '<span class="badge online">online</span>'
    : '<span class="badge offline">offline</span>';
  const win = d.windows_version || d.os || "-";
  const wakeBtn = d.status !== "online" && d.mac
    ? `<button class="btn" style="font-size:0.75rem;padding:0.25rem 0.5rem;margin-left:0.5rem"
         onclick="event.stopPropagation();wake('${d.id}', this)" title="Kirim Wake-on-LAN (butuh diaktifkan di BIOS PC target)">⚡ Wake</button>`
    : "";
  return `<tr onclick="location.href='/device/${encodeURIComponent(d.id)}'">
    <td>${escapeHtml(d.hostname || "-")}</td>
    <td>${escapeHtml(d.username || "-")}</td>
    <td>${escapeHtml(d.ip || "-")}</td>
    <td>${escapeHtml(win)}</td>
    <td>${meter(d.metrics && d.metrics.cpu_percent)}</td>
    <td>${meter(d.metrics && d.metrics.ram_percent)}</td>
    <td>${badge}${wakeBtn}</td>
    <td class="muted">${timeAgo(d.last_seen)}</td>
  </tr>`;
}

async function wake(id, btn) {
  const original = btn.textContent;
  btn.disabled = true;
  btn.textContent = "Mengirim…";
  try {
    await api(`/api/devices/${encodeURIComponent(id)}/wake`, { method: "POST" });
    btn.textContent = "Terkirim ✓";
  } catch (err) {
    alert("Gagal mengirim Wake-on-LAN: " + err.message);
    btn.textContent = original;
    btn.disabled = false;
  }
}

async function refresh() {
  try {
    const [stats, devices] = await Promise.all([
      api("/api/stats"),
      api("/api/devices"),
    ]);
    document.getElementById("stat-total").textContent = stats.total;
    document.getElementById("stat-online").textContent = stats.online;
    document.getElementById("stat-offline").textContent = stats.offline;

    const body = document.getElementById("device-rows");
    body.innerHTML = devices.length
      ? devices.map(deviceRow).join("")
      : '<tr><td colspan="8" class="soon">Belum ada perangkat.</td></tr>';

    document.getElementById("refresh-note").textContent =
      "diperbarui " + new Date().toLocaleTimeString("id-ID");
  } catch (err) {
    document.getElementById("refresh-note").textContent = "gagal memuat: " + err.message;
  }
}

async function init() {
  try {
    const me = await api("/api/me");
    document.getElementById("whoami").textContent = me.username + " (" + me.role + ")";
  } catch (_) { /* redirect ditangani api() */ }
  refresh();
  setInterval(refresh, REFRESH_MS);
}

init();
