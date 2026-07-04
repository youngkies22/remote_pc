// Tab File Explorer (Tahap 6).
(function () {
  const rows = () => document.getElementById("fx-rows");
  const pathInput = () => document.getElementById("fx-path");
  let cwd = "";

  function join(base, name) {
    if (!base) return name;
    return base.endsWith("\\") ? base + name : base + "\\" + name;
  }
  function parent(p) {
    if (!p) return "";
    const t = p.replace(/\\+$/, "");
    const i = t.lastIndexOf("\\");
    if (i <= 2) return i === 2 ? t.slice(0, 3) : ""; // "C:\" atau daftar drive
    return t.slice(0, i);
  }

  async function load(path) {
    cwd = path || "";
    pathInput().value = cwd;
    rows().innerHTML = '<tr><td colspan="4" class="soon">Memuat…</td></tr>';
    let d;
    try {
      d = await RP.api("/fs/list?path=" + encodeURIComponent(cwd));
    } catch (err) {
      rows().innerHTML = `<tr><td colspan="4" class="soon">Error: ${escapeHtml(err.message)}</td></tr>`;
      return;
    }
    const entries = d.entries || [];
    rows().innerHTML = entries.length ? entries.map((e) => {
      const full = cwd ? join(cwd, e.name) : e.name;
      const icon = e.is_dir ? "📁" : "📄";
      const size = e.is_dir ? "" : RP.fmtBytes(e.size);
      const when = e.mod_time ? new Date(e.mod_time * 1000).toLocaleString("id-ID") : "";
      const dl = e.is_dir ? "" :
        `<a class="btn" href="/api/devices/${encodeURIComponent(RP.deviceId)}/fs/download?path=${encodeURIComponent(full)}">⬇</a>`;
      return `<tr>
        <td><span data-open="${e.is_dir ? full : ""}" style="cursor:${e.is_dir ? "pointer" : "default"}">${icon} ${escapeHtml(e.name)}</span></td>
        <td class="muted">${size}</td>
        <td class="muted">${when}</td>
        <td class="text-end">
          ${dl}
          <button class="btn" data-ren="${escapeHtml(full)}" data-name="${escapeHtml(e.name)}">✎</button>
          <button class="btn btn-danger" data-del="${escapeHtml(full)}">🗑</button>
        </td>
      </tr>`;
    }).join("") : '<tr><td colspan="4" class="soon">Folder kosong.</td></tr>';
  }

  async function post(path, body) {
    return RP.api(path, { method: "POST", body: JSON.stringify(body) });
  }

  async function mkdir() {
    const name = prompt("Nama folder baru:");
    if (!name) return;
    try { await post("/fs/mkdir", { path: join(cwd, name) }); load(cwd); }
    catch (err) { alert("Gagal: " + err.message); }
  }

  async function del(path) {
    if (!confirm("Hapus " + path + " ?")) return;
    try { await post("/fs/delete", { path }); load(cwd); }
    catch (err) { alert("Gagal: " + err.message); }
  }

  async function rename(full, name) {
    const nn = prompt("Nama baru:", name);
    if (!nn || nn === name) return;
    try { await post("/fs/rename", { src: full, dst: join(parent(full), nn) }); load(cwd); }
    catch (err) { alert("Gagal: " + err.message); }
  }

  function upload() {
    const inp = document.getElementById("fx-file");
    inp.onchange = () => {
      const f = inp.files[0];
      if (!f) return;
      const r = new FileReader();
      r.onload = async () => {
        const b64 = String(r.result).split(",")[1] || "";
        try { await post("/fs/upload", { path: join(cwd, f.name), data: b64 }); load(cwd); }
        catch (err) { alert("Gagal unggah: " + err.message); }
      };
      r.readAsDataURL(f);
    };
    inp.click();
  }

  document.getElementById("fx-up").addEventListener("click", () => load(parent(cwd)));
  document.getElementById("fx-refresh").addEventListener("click", () => load(cwd));
  document.getElementById("fx-mkdir").addEventListener("click", mkdir);
  document.getElementById("fx-upload").addEventListener("click", upload);
  pathInput().addEventListener("keydown", (e) => { if (e.key === "Enter") load(pathInput().value.trim()); });
  rows().addEventListener("click", (e) => {
    const open = e.target.closest("[data-open]");
    if (open && open.dataset.open) return load(open.dataset.open);
    const del1 = e.target.closest("[data-del]");
    if (del1) return del(del1.dataset.del);
    const ren = e.target.closest("[data-ren]");
    if (ren) return rename(ren.dataset.ren, ren.dataset.name);
  });

  window.Tabs.files = { activate() { load(cwd); }, deactivate() {} };
})();
