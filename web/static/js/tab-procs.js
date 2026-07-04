// Tab Process Manager (Tahap 11).
(function () {
  const rows = () => document.getElementById("proc-rows");

  async function load() {
    rows().innerHTML = '<tr><td colspan="6" class="soon">Memuat…</td></tr>';
    let d;
    try {
      d = await RP.api("/processes");
    } catch (err) {
      rows().innerHTML = `<tr><td colspan="6" class="soon">Error: ${escapeHtml(err.message)}</td></tr>`;
      return;
    }
    const list = d.processes || [];
    rows().innerHTML = list.length ? list.map((p) => `
      <tr>
        <td>${p.pid}</td>
        <td>${escapeHtml(p.name)}</td>
        <td>${p.cpu}</td>
        <td>${p.mem_mb}</td>
        <td class="muted">${escapeHtml(p.status || "-")}</td>
        <td class="text-end"><button class="btn btn-danger" data-kill="${p.pid}" data-name="${escapeHtml(p.name)}">Kill</button></td>
      </tr>`).join("") : '<tr><td colspan="6" class="soon">Tidak ada proses.</td></tr>';
  }

  async function kill(pid, name) {
    if (!confirm(`Matikan proses ${name} (PID ${pid})?`)) return;
    try {
      await RP.api("/processes/kill", { method: "POST", body: JSON.stringify({ pid: Number(pid) }) });
      load();
    } catch (err) { alert("Gagal: " + err.message); }
  }

  document.getElementById("proc-refresh").addEventListener("click", load);
  rows().addEventListener("click", (e) => {
    const b = e.target.closest("[data-kill]");
    if (b) kill(b.dataset.kill, b.dataset.name);
  });
  window.Tabs.processes = { activate() { load(); }, deactivate() {} };
})();
