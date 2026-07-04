// Tab System Information (Tahap 4).
(function () {
  function row(k, v) {
    return `<div class="k">${escapeHtml(k)}</div><div>${escapeHtml(v)}</div>`;
  }

  async function load() {
    const body = document.getElementById("sysinfo-body");
    body.innerHTML = row("Status", "Mengambil info sistem (WMI, mungkin beberapa detik)…");
    let d;
    try {
      d = await RP.api("/sysinfo");
    } catch (err) {
      body.innerHTML = row("Error", err.message);
      return;
    }
    const gpus = (d.gpus || []).map((g) => g.name + (g.ram_mb ? ` (${g.ram_mb} MB)` : "")).join("; ") || "-";
    const disks = (d.disks || []).map((x) => `${x.name} (${x.size_gb} GB)`).join("; ") || "-";
    const nets = (d.adapters || []).map((a) => `${a.name} — ${a.ip || "-"} / ${a.mac}`).join("<br>") || "-";
    body.innerHTML = [
      row("Hostname", d.hostname || "-"),
      row("Username", d.username || "-"),
      row("Sistem Operasi", d.os || "-"),
      row("Build", d.build || "-"),
      row("Manufaktur", d.manufacturer || "-"),
      row("Model", d.model || "-"),
      row("Serial Number", d.serial || "-"),
      row("Motherboard", d.motherboard || "-"),
      row("BIOS", d.bios || "-"),
      row("CPU", `${d.cpu || "-"} (${d.cpu_cores || 0} core)`),
      row("RAM Total", `${d.ram_total_mb || 0} MB`),
      row("GPU", gpus),
      row("Disk", disks),
    ].join("") + `<div class="k">Adapter Jaringan</div><div>${nets}</div>`;
  }

  document.getElementById("si-refresh").addEventListener("click", load);
  window.Tabs.sysinfo = { activate() {}, deactivate() {} };
})();
