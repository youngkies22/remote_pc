// Inti halaman detail: pergantian tab, Overview, dan helper bersama antar-tab.

const DEVICE_ID = decodeURIComponent(location.pathname.split("/").pop());
window.Tabs = {}; // tiap file tab mendaftarkan { activate, deactivate }

let currentTab = "overview";
let overviewTimer = null;

// --- Helper bersama (dipakai file tab-*.js) ---
window.RP = {
  deviceId: DEVICE_ID,

  // api relatif ke device ini.
  api(path, opts) {
    return api("/api/devices/" + encodeURIComponent(DEVICE_ID) + path, opts);
  },

  // buka WebSocket operator untuk mode "screen"/"terminal".
  ws(mode) {
    const proto = location.protocol === "https:" ? "wss:" : "ws:";
    return new WebSocket(
      `${proto}//${location.host}/ws/operator?device=${encodeURIComponent(DEVICE_ID)}&mode=${mode}`
    );
  },

  // bungkus payload menjadi envelope protokol dan kirim lewat ws.
  sendEnv(ws, type, payload) {
    if (ws && ws.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify({ id: RP.uuid(), type, payload, timestamp: Date.now() }));
    }
  },

  uuid() {
    return (crypto.randomUUID && crypto.randomUUID()) ||
      "xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx".replace(/[xy]/g, (c) => {
        const r = (Math.random() * 16) | 0;
        return (c === "x" ? r : (r & 0x3) | 0x8).toString(16);
      });
  },

  fmtBytes(n) {
    n = Number(n) || 0;
    if (n < 1024) return n + " B";
    const u = ["KB", "MB", "GB", "TB"];
    let i = -1;
    do { n /= 1024; i++; } while (n >= 1024 && i < u.length - 1);
    return n.toFixed(1) + " " + u[i];
  },

  fmtEpoch(sec) {
    if (!sec) return "-";
    return new Date(sec * 1000).toLocaleString("id-ID");
  },
};

// --- Pergantian tab ---
function showPanel(name) {
  if (name === currentTab) return;
  const prev = window.Tabs[currentTab];
  if (prev && prev.deactivate) prev.deactivate();
  currentTab = name;

  document.querySelectorAll("#tabs .tab").forEach((t) =>
    t.classList.toggle("active", t.dataset.tab === name));
  document.querySelectorAll("[data-panel]").forEach((p) =>
    (p.hidden = p.dataset.panel !== name));

  const mod = window.Tabs[name];
  if (mod && mod.activate) mod.activate();
}

// --- Overview ---
function ovRow(label, value) {
  return `<div class="k">${escapeHtml(label)}</div><div>${escapeHtml(value)}</div>`;
}
function fmtUptime(sec) {
  sec = Number(sec) || 0;
  const d = Math.floor(sec / 86400), h = Math.floor((sec % 86400) / 3600), m = Math.floor((sec % 3600) / 60);
  return `${d} hari ${h} jam ${m} mnt`;
}

async function overviewRefresh() {
  let d;
  try {
    d = await RP.api("");
  } catch (err) {
    document.getElementById("overview").innerHTML = ovRow("Error", err.message);
    return;
  }
  document.getElementById("title").textContent = d.hostname || "Detail";
  document.getElementById("status-badge").innerHTML = d.status === "online"
    ? '<span class="badge online">online</span>' : '<span class="badge offline">offline</span>';
  const m = d.metrics || {};
  document.getElementById("overview").innerHTML = [
    ovRow("Hostname", d.hostname || "-"),
    ovRow("Username", d.username || "-"),
    ovRow("IP Address", d.ip || "-"),
    ovRow("MAC Address", d.mac || "-"),
    ovRow("Versi Windows", d.windows_version || "-"),
    ovRow("Arsitektur", d.arch || "-"),
    ovRow("CPU", (m.cpu_percent ?? 0) + " %"),
    ovRow("RAM", `${m.ram_used_mb ?? 0} / ${m.ram_total_mb ?? 0} MB (${m.ram_percent ?? 0} %)`),
    ovRow("Disk", `${m.disk_used_gb ?? 0} / ${m.disk_total_gb ?? 0} GB (${m.disk_percent ?? 0} %)`),
    ovRow("Uptime", fmtUptime(m.uptime_sec)),
    ovRow("Terakhir terlihat", timeAgo(d.last_seen)),
  ].join("");
  document.getElementById("ov-updated").textContent = "diperbarui " + new Date().toLocaleTimeString("id-ID");
}

window.Tabs.overview = {
  activate() { overviewRefresh(); overviewTimer = setInterval(overviewRefresh, 3000); },
  deactivate() { clearInterval(overviewTimer); overviewTimer = null; },
};

async function powerAction(action) {
  const isShutdown = action === "shutdown";
  const word = isShutdown ? "MEMATIKAN" : "MERESTART";
  if (!confirm(`Yakin ${word} komputer ini dari jarak jauh?\n\nSemua aplikasi yang belum disimpan akan tertutup paksa.`)) return;
  try {
    await RP.api("/power", { method: "POST", body: JSON.stringify({ action }) });
    alert(`Perintah terkirim. Komputer akan segera ${isShutdown ? "mati" : "restart"}.`);
  } catch (err) {
    alert("Gagal mengirim perintah: " + err.message);
  }
}

async function removeDevice() {
  if (!confirm("Hapus device ini dari daftar?")) return;
  try {
    await RP.api("", { method: "DELETE" });
    location.href = "/";
  } catch (err) { alert("Gagal menghapus: " + err.message); }
}

function DeviceInit() {
  document.getElementById("tabs").addEventListener("click", (e) => {
    const tab = e.target.closest(".tab");
    if (tab) showPanel(tab.dataset.tab);
  });
  window.Tabs.overview.activate();
}
