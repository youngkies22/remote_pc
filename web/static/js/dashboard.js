// Logika halaman dashboard PC: memuat statistik + daftar device Windows
// (dikelompokkan per subnet IP) dan auto-refresh. HP Android SENGAJA tidak
// ditampilkan di sini — punya halaman sendiri "/hp" (lihat dashboard-hp.js)
// supaya tidak tercampur dengan PC.

const REFRESH_MS = 1500;
const SMALL_BTN = "font-size:0.75rem;padding:0.25rem 0.5rem";

// Menyimpan device PC (sudah difilter, tanpa Android) terbaru agar aksi
// massal/grup tahu ID mana yang online.
let lastDevices = [];

function meter(percent) {
  const p = Math.max(0, Math.min(100, Number(percent) || 0));
  return `<div class="meter" title="${p}%"><span style="width:${p}%"></span></div>`;
}

// subnetOf mengembalikan label subnet /24 dari sebuah IP (3 oktet pertama).
// IP kosong/tidak valid dikelompokkan ke "Tanpa IP".
function subnetOf(ip) {
  const parts = String(ip || "").split(".");
  if (parts.length !== 4 || parts.some((p) => p === "" || isNaN(p))) return "Tanpa IP";
  return `${parts[0]}.${parts[1]}.${parts[2]}.0/24`;
}

function deviceRow(d) {
  const badge = d.status === "online"
    ? '<span class="badge online">online</span>'
    : '<span class="badge offline">offline</span>';
  const win = d.windows_version || d.os || "-";
  const online = d.status === "online";
  const id = encodeURIComponent(d.id);

  const wakeBtn = !online && d.mac
    ? `<button class="btn" style="${SMALL_BTN}"
         onclick="event.stopPropagation();wake('${d.id}', this)"
         title="Kirim Wake-on-LAN (butuh diaktifkan di BIOS PC target)">⚡</button>`
    : "";
  // Shutdown tidak & tidak akan didukung agent Android (di luar scope
  // monitor-only BYOD), jadi disembunyikan supaya guru tidak klik tombol
  // yang diam-diam tidak berefek. Pesan didukung (Tahap A3).
  const msgBtn = online
    ? `<button class="btn" style="${SMALL_BTN}"
         onclick="event.stopPropagation();messageOne('${d.id}')" title="Kirim pesan ke PC ini">💬</button>`
    : "";
  const offBtn = online
    ? `<button class="btn btn-danger" style="${SMALL_BTN}"
         onclick="event.stopPropagation();powerOne('${d.id}','shutdown','${escapeHtml(d.hostname || "")}')" title="Matikan PC ini">⏻</button>`
    : "";
  const delBtn = `<button class="btn btn-danger" style="${SMALL_BTN}"
         onclick="event.stopPropagation();deleteDevice('${d.id}','${escapeHtml(d.hostname || "")}')" title="Hapus device ini dari daftar">🗑</button>`;

  return `<tr onclick="location.href='/device/${id}'">
    <td>${escapeHtml(d.hostname || "-")}</td>
    <td>${escapeHtml(d.username || "-")}</td>
    <td>${escapeHtml(d.ip || "-")}</td>
    <td>${escapeHtml(win)}</td>
    <td>${meter(d.metrics && d.metrics.cpu_percent)}</td>
    <td>${meter(d.metrics && d.metrics.ram_percent)}</td>
    <td>${badge}</td>
    <td class="muted">${timeAgo(d.last_seen)}</td>
    <td class="text-end" style="white-space:nowrap">${wakeBtn}${msgBtn}${offBtn}${delBtn}</td>
  </tr>`;
}

// deleteDevice menghapus device dari daftar (mis. PC lama yang sudah tidak
// dipakai). Bisa dipanggil dari halaman ini maupun dashboard-hp.js.
async function deleteDevice(id, hostname) {
  if (!confirm(`Hapus "${hostname || id}" dari daftar?\n\nKalau device ini nyala lagi & connect, ia akan terdaftar ulang sbg device baru.`)) return;
  try {
    await api(`/api/devices/${encodeURIComponent(id)}`, { method: "DELETE" });
    if (typeof refresh === "function") refresh();
  } catch (err) {
    alert("Gagal menghapus: " + err.message);
  }
}

// groupHeaderRow membuat baris pemisah per subnet, lengkap dengan aksi grup.
function groupHeaderRow(subnet, list) {
  const onlineIDs = list.filter((d) => d.status === "online").map((d) => d.id);
  const idsAttr = escapeHtml(JSON.stringify(onlineIDs));
  const onlineCount = onlineIDs.length;
  const actions = onlineCount
    ? `<span style="float:right;display:inline-flex;gap:0.35rem">
         <button class="btn" style="${SMALL_BTN}" onclick='messageGroup(${idsAttr})' title="Kirim pesan ke grup ini">💬 Pesan</button>
         <button class="btn" style="${SMALL_BTN}" onclick='powerGroup(${idsAttr}, "restart")' title="Restart grup ini">⟳</button>
         <button class="btn btn-danger" style="${SMALL_BTN}" onclick='powerGroup(${idsAttr}, "shutdown")' title="Matikan grup ini">⏻</button>
       </span>`
    : "";
  return `<tr class="group-row"><td colspan="9">
    <strong>${escapeHtml(subnet)}</strong>
    <span class="muted" style="font-size:0.8rem">· ${list.length} PC (${onlineCount} online)</span>
    ${actions}
  </td></tr>`;
}

// buildRows mengelompokkan device per subnet lalu merangkai baris tabel.
function buildRows(devices) {
  const groups = new Map();
  for (const d of devices) {
    const key = subnetOf(d.ip);
    if (!groups.has(key)) groups.set(key, []);
    groups.get(key).push(d);
  }
  // Urutkan subnet secara alami (numerik per oktet), "Tanpa IP" di akhir.
  const keys = [...groups.keys()].sort((a, b) => {
    if (a === "Tanpa IP") return 1;
    if (b === "Tanpa IP") return -1;
    return a.localeCompare(b, undefined, { numeric: true });
  });

  let html = "";
  for (const key of keys) {
    const list = groups.get(key).sort((a, b) => {
      if (a.status !== b.status) return a.status === "online" ? -1 : 1;
      return (a.hostname || "").localeCompare(b.hostname || "");
    });
    html += groupHeaderRow(key, list) + list.map(deviceRow).join("");
  }
  return html;
}

async function wake(id, btn) {
  const original = btn.textContent;
  btn.disabled = true;
  btn.textContent = "…";
  try {
    await api(`/api/devices/${encodeURIComponent(id)}/wake`, { method: "POST" });
    btn.textContent = "✓";
  } catch (err) {
    alert("Gagal mengirim Wake-on-LAN: " + err.message);
    btn.textContent = original;
    btn.disabled = false;
  }
}

// --- Kirim pesan ---
async function messageOne(id) {
  const text = prompt("Tulis pesan yang akan muncul di layar PC ini:");
  if (!text) return;
  try {
    await api(`/api/devices/${encodeURIComponent(id)}/message`, {
      method: "POST",
      body: JSON.stringify({ title: "Pesan dari Guru", text }),
    });
    alert("Pesan terkirim.");
  } catch (err) {
    alert("Gagal mengirim pesan: " + err.message);
  }
}

async function messageBulk(ids, label) {
  const text = prompt(`Tulis pesan yang akan muncul di ${label}:`);
  if (!text) return;
  try {
    const res = await api("/api/devices/message-all", {
      method: "POST",
      body: JSON.stringify({ title: "Pesan dari Guru", text, ids }),
    });
    alert(`Pesan terkirim ke ${res.count} PC.`);
  } catch (err) {
    alert("Gagal mengirim pesan: " + err.message);
  }
}

function messageAll() {
  messageBulk(onlinePcIds(), "SEMUA PC online");
}
function messageGroup(ids) {
  messageBulk(ids, `grup ini (${ids.length} PC)`);
}

// --- Shutdown / restart ---
async function powerOne(id, action, hostname) {
  const word = action === "shutdown" ? "MEMATIKAN" : "MERESTART";
  if (!confirm(`Yakin ${word} "${hostname || id}"?`)) return;
  try {
    await api(`/api/devices/${encodeURIComponent(id)}/power`, {
      method: "POST",
      body: JSON.stringify({ action }),
    });
    alert("Perintah terkirim.");
  } catch (err) {
    alert("Gagal: " + err.message);
  }
}

async function powerBulk(ids, action, label) {
  const word = action === "shutdown" ? "MEMATIKAN" : "MERESTART";
  if (!confirm(`Yakin ${word} ${label}?\n\nSemua pekerjaan yang belum disimpan akan hilang.`)) return;
  try {
    const res = await api("/api/devices/power-all", {
      method: "POST",
      body: JSON.stringify({ action, ids }),
    });
    alert(`Perintah ${action} terkirim ke ${res.count} PC.`);
  } catch (err) {
    alert("Gagal: " + err.message);
  }
}

// "Semua" di halaman ini SENGAJA cuma menargetkan PC (bukan andalkan ids
// kosong = semua device di server, yang akan ikut menyasar HP Android).
function onlinePcIds() {
  return lastDevices.filter((d) => d.status === "online").map((d) => d.id);
}
function powerAll(action) {
  powerBulk(onlinePcIds(), action, "SEMUA PC online");
}
function powerGroup(ids, action) {
  powerBulk(ids, action, `grup ini (${ids.length} PC)`);
}

async function refresh() {
  try {
    const devices = await api("/api/devices");
    // HP Android punya halaman sendiri ("/hp") — jangan tampilkan di sini.
    const pcDevices = devices.filter((d) => d.os !== "Android");
    lastDevices = pcDevices;

    const online = pcDevices.filter((d) => d.status === "online").length;
    document.getElementById("stat-total").textContent = pcDevices.length;
    document.getElementById("stat-online").textContent = online;
    document.getElementById("stat-offline").textContent = pcDevices.length - online;

    const body = document.getElementById("device-rows");
    body.innerHTML = pcDevices.length
      ? buildRows(pcDevices)
      : '<tr><td colspan="9" class="soon">Belum ada PC.</td></tr>';

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
