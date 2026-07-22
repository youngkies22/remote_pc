// Halaman "/hp/live": grid thumbnail live screen SEMUA HP Android yang
// sedang online sekaligus (bukan cuma satu). Tiap thumbnail = satu koneksi
// /ws/operator?mode=screen sendiri (server otomatis kirim screen.start ke
// agent begitu koneksi dibuka, persis seperti tab Live Screen individual).
// Klik thumbnail -> buka tampilan penuh per-HP (/device/{id}#screen).

const RECONCILE_MS = 5000;
const tiles = new Map(); // deviceId -> { ws, tileEl, imgEl, hintEl }

function wsUrl(deviceId) {
  const proto = location.protocol === "https:" ? "wss:" : "ws:";
  return `${proto}//${location.host}/ws/operator?device=${encodeURIComponent(deviceId)}&mode=screen`;
}

function makeTile(d) {
  const tile = document.createElement("div");
  tile.className = "live-tile";
  tile.innerHTML = `
    <div class="thumb-wrap">
      <img hidden />
      <div class="hint">menyambungkan…</div>
    </div>
    <div class="label">
      <span>${escapeHtml(d.hostname || d.id)}</span>
      <span class="badge-dot" title="online"></span>
    </div>`;
  tile.addEventListener("click", () => { location.href = `/device/${encodeURIComponent(d.id)}#screen`; });
  document.getElementById("live-grid").appendChild(tile);
  return {
    tileEl: tile,
    imgEl: tile.querySelector("img"),
    hintEl: tile.querySelector(".hint"),
  };
}

function openStream(id, entry) {
  const ws = new WebSocket(wsUrl(id));
  entry.ws = ws;
  ws.onmessage = (ev) => {
    let m;
    try { m = JSON.parse(ev.data); } catch { return; }
    if (m.type === "screen.frame" && m.payload && m.payload.data) {
      entry.imgEl.src = "data:image/jpeg;base64," + m.payload.data;
      entry.imgEl.hidden = false;
      entry.hintEl.hidden = true;
    }
  };
  ws.onclose = () => { entry.hintEl.textContent = "terputus"; entry.hintEl.hidden = false; entry.imgEl.hidden = true; };
  ws.onerror = () => { entry.hintEl.textContent = "gagal terhubung"; };
}

function removeTile(id) {
  const entry = tiles.get(id);
  if (!entry) return;
  try { entry.ws && entry.ws.close(); } catch (_) {}
  entry.tileEl.remove();
  tiles.delete(id);
}

async function reconcile() {
  let devices;
  try {
    devices = await api("/api/devices");
  } catch (err) {
    document.getElementById("live-note").textContent = "gagal memuat: " + err.message;
    return;
  }
  const online = new Map(
    devices.filter((d) => d.os === "Android" && d.status === "online").map((d) => [d.id, d])
  );

  // Hapus tile utk HP yang sudah offline/hilang.
  for (const id of [...tiles.keys()]) {
    if (!online.has(id)) removeTile(id);
  }
  // Tambah tile utk HP online baru.
  for (const [id, d] of online) {
    if (tiles.has(id)) continue;
    const entry = makeTile(d);
    tiles.set(id, entry);
    openStream(id, entry);
  }

  const grid = document.getElementById("live-grid");
  if (tiles.size === 0) {
    grid.innerHTML = '<div class="soon">Tidak ada HP online saat ini.</div>';
  } else if (grid.querySelector(".soon")) {
    grid.querySelector(".soon").remove();
  }
  document.getElementById("live-note").textContent = `${tiles.size} HP online`;
}

async function init() {
  try {
    const me = await api("/api/me");
  } catch (_) { /* redirect ditangani api() */ }
  await reconcile();
  setInterval(reconcile, RECONCILE_MS);
}

init();
