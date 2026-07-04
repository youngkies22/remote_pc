// Tab Windows Services (Tahap 12).
(function () {
  const rows = () => document.getElementById("svc-rows");
  let all = [];

  function render() {
    const q = document.getElementById("svc-filter").value.toLowerCase();
    const list = all.filter((s) =>
      s.name.toLowerCase().includes(q) || (s.display || "").toLowerCase().includes(q));
    rows().innerHTML = list.length ? list.slice(0, 500).map((s) => {
      const running = s.status === "running";
      const badge = running ? '<span class="badge online">running</span>'
        : `<span class="badge offline">${escapeHtml(s.status)}</span>`;
      return `<tr>
        <td>${escapeHtml(s.display || s.name)}</td>
        <td class="muted">${escapeHtml(s.name)}</td>
        <td>${badge}</td>
        <td class="text-end">
          <button class="btn" data-act="start" data-name="${escapeHtml(s.name)}">Start</button>
          <button class="btn" data-act="stop" data-name="${escapeHtml(s.name)}">Stop</button>
          <button class="btn" data-act="restart" data-name="${escapeHtml(s.name)}">Restart</button>
        </td>
      </tr>`;
    }).join("") : '<tr><td colspan="4" class="soon">Tidak ada service cocok.</td></tr>';
  }

  async function load() {
    rows().innerHTML = '<tr><td colspan="4" class="soon">Memuat…</td></tr>';
    try {
      const d = await RP.api("/services");
      all = d.services || [];
      render();
    } catch (err) {
      rows().innerHTML = `<tr><td colspan="4" class="soon">Error: ${escapeHtml(err.message)}</td></tr>`;
    }
  }

  async function control(name, action) {
    try {
      await RP.api("/services/control", { method: "POST", body: JSON.stringify({ name, action }) });
      setTimeout(load, 600);
    } catch (err) { alert("Gagal (perlu admin?): " + err.message); }
  }

  document.getElementById("svc-refresh").addEventListener("click", load);
  document.getElementById("svc-filter").addEventListener("input", render);
  rows().addEventListener("click", (e) => {
    const b = e.target.closest("[data-act]");
    if (b) control(b.dataset.name, b.dataset.act);
  });
  window.Tabs.services = { activate() { if (!all.length) load(); }, deactivate() {} };
})();
