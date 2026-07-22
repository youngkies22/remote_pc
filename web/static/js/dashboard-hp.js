// Logika halaman "/hp": daftar HP Android siswa (BYOD) saja, dipisah total
// dari dashboard PC ("/") — tetap dikelompokkan per subnet IP di antara
// sesama HP, tapi tak pernah tercampur baris dengan PC. Aksi yang didukung
// cuma Pesan & Hapus (Shutdown/Restart/Wake tak berlaku utk Android).

const REFRESH_MS = 3000;
const SMALL_BTN = "font-size:0.75rem;padding:0.25rem 0.5rem";

let lastDevices = [];

function meter(percent) {
  const p = Math.max(0, Math.min(100, Number(percent) || 0));
  return `<div class="meter" title="${p}%"><span style="width:${p}%"></span></div>`;
}

function subnetOf(ip) {
  const parts = String(ip || "").split(".");
  if (parts.length !== 4 || parts.some((p) => p === "" || isNaN(p))) return "Tanpa IP";
  return `${parts[0]}.${parts[1]}.${parts[2]}.0/24`;
}

function hpRow(d) {
  const badge = d.status === "online"
    ? '<span class="badge online">online</span>'
    : '<span class="badge offline">offline</span>';
  const online = d.status === "online";
  const id = encodeURIComponent(d.id);
  const m = d.metrics || {};

  const msgBtn = online
    ? `<button class="btn" style="${SMALL_BTN}"
         onclick="event.stopPropagation();messageOneHp('${d.id}')" title="Kirim pesan ke HP ini">💬</button>`
    : "";
  const delBtn = `<button class="btn btn-danger" style="${SMALL_BTN}"
         onclick="event.stopPropagation();deleteDevice('${d.id}','${escapeHtml(d.hostname || "")}')" title="Hapus HP ini dari daftar">🗑</button>`;

  return `<tr onclick="location.href='/device/${id}'">
    <td>${escapeHtml(d.hostname || "-")}</td>
    <td>${escapeHtml(d.username || "-")}</td>
    <td>${escapeHtml(d.ip || "-")}</td>
    <td>${escapeHtml(d.windows_version || "-")}</td>
    <td>${m.battery_percent ? meter(m.battery_percent) : '<span class="muted">-</span>'}</td>
    <td>${meter(m.ram_percent)}</td>
    <td>${badge}</td>
    <td class="muted">${timeAgo(d.last_seen)}</td>
    <td class="text-end" style="white-space:nowrap">${msgBtn}${delBtn}</td>
  </tr>`;
}

function groupHeaderRow(subnet, list) {
  const onlineIDs = list.filter((d) => d.status === "online").map((d) => d.id);
  const idsAttr = escapeHtml(JSON.stringify(onlineIDs));
  const onlineCount = onlineIDs.length;
  const actions = onlineCount
    ? `<span style="float:right;display:inline-flex;gap:0.35rem">
         <button class="btn" style="${SMALL_BTN}" onclick='messageGroupHp(${idsAttr})' title="Kirim pesan ke grup ini">💬 Pesan</button>
       </span>`
    : "";
  return `<tr class="group-row"><td colspan="9">
    <strong>${escapeHtml(subnet)}</strong>
    <span class="muted" style="font-size:0.8rem">· ${list.length} HP (${onlineCount} online)</span>
    ${actions}
  </td></tr>`;
}

function buildRows(devices) {
  const groups = new Map();
  for (const d of devices) {
    const key = subnetOf(d.ip);
    if (!groups.has(key)) groups.set(key, []);
    groups.get(key).push(d);
  }
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
    html += groupHeaderRow(key, list) + list.map(hpRow).join("");
  }
  return html;
}

async function deleteDevice(id, hostname) {
  if (!confirm(`Hapus "${hostname || id}" dari daftar?\n\nKalau HP ini connect lagi, ia akan terdaftar ulang sbg device baru.`)) return;
  try {
    await api(`/api/devices/${encodeURIComponent(id)}`, { method: "DELETE" });
    refresh();
  } catch (err) {
    alert("Gagal menghapus: " + err.message);
  }
}

async function messageOneHp(id) {
  const text = prompt("Tulis pesan yang akan muncul di HP ini:");
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

async function messageBulkHp(ids, label) {
  if (!ids.length) { alert("Tidak ada HP online di " + label + "."); return; }
  const text = prompt(`Tulis pesan yang akan muncul di ${label}:`);
  if (!text) return;
  try {
    const res = await api("/api/devices/message-all", {
      method: "POST",
      body: JSON.stringify({ title: "Pesan dari Guru", text, ids }),
    });
    alert(`Pesan terkirim ke ${res.count} HP.`);
  } catch (err) {
    alert("Gagal mengirim pesan: " + err.message);
  }
}

function onlineHpIds() {
  return lastDevices.filter((d) => d.status === "online").map((d) => d.id);
}
function messageAllHp() {
  messageBulkHp(onlineHpIds(), "SEMUA HP online");
}
function messageGroupHp(ids) {
  messageBulkHp(ids, `grup ini (${ids.length} HP)`);
}

async function refresh() {
  try {
    const devices = await api("/api/devices");
    const hpDevices = devices.filter((d) => d.os === "Android");
    lastDevices = hpDevices;

    const online = hpDevices.filter((d) => d.status === "online").length;
    document.getElementById("stat-total").textContent = hpDevices.length;
    document.getElementById("stat-online").textContent = online;
    document.getElementById("stat-offline").textContent = hpDevices.length - online;

    const body = document.getElementById("device-rows");
    body.innerHTML = hpDevices.length
      ? buildRows(hpDevices)
      : '<tr><td colspan="9" class="soon">Belum ada HP.</td></tr>';

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
