// Tab Live Screen + Remote Control (Tahap 8/9/10).
(function () {
  let ws = null;
  let lastMove = 0;
  const img = () => document.getElementById("screen-img");
  const hint = () => document.getElementById("screen-hint");
  const stage = () => document.getElementById("screen-stage");
  const control = () => document.getElementById("screen-control").checked;
  const quality = () => document.getElementById("screen-quality").value;

  function setConnected(on) {
    document.getElementById("screen-connect").disabled = on;
    document.getElementById("screen-disconnect").disabled = !on;
  }

  function sendQuality() {
    RP.sendEnv(ws, "screen.quality", { quality: quality() });
  }

  function connect() {
    if (ws) return;
    ws = RP.ws("screen");
    ws.onopen = () => { setConnected(true); hint().textContent = "menyambungkan…"; sendQuality(); };
    ws.onmessage = (ev) => {
      let m;
      try { m = JSON.parse(ev.data); } catch { return; }
      if (m.type === "screen.frame" && m.payload && m.payload.data) {
        img().src = "data:image/jpeg;base64," + m.payload.data;
        hint().style.display = "none";
      }
    };
    ws.onclose = () => { ws = null; setConnected(false); hint().style.display = ""; hint().textContent = "Terputus."; };
    ws.onerror = () => { hint().textContent = "Gagal terhubung (agent online?)"; };
  }

  function disconnect() { if (ws) ws.close(); }

  function toggleFullscreen() {
    if (!document.fullscreenElement) {
      stage().requestFullscreen().catch(() => {});
    } else {
      document.exitFullscreen();
    }
  }
  document.addEventListener("fullscreenchange", () => {
    const on = document.fullscreenElement === stage();
    document.getElementById("screen-fullscreen").textContent = on ? "🗗 Keluar Fullscreen" : "⛶ Fullscreen";
  });

  // Koordinat relatif 0..1 terhadap gambar.
  function rel(e) {
    const r = img().getBoundingClientRect();
    return {
      x: Math.min(1, Math.max(0, (e.clientX - r.left) / r.width)),
      y: Math.min(1, Math.max(0, (e.clientY - r.top) / r.height)),
    };
  }
  function mouse(action, e, extra) {
    if (!control()) return;
    const p = rel(e);
    RP.sendEnv(ws, "input.mouse", Object.assign({ action, x: p.x, y: p.y }, extra || {}));
  }

  function bindInput() {
    const el = img();
    el.addEventListener("mousemove", (e) => {
      const now = Date.now();
      if (now - lastMove < 50) return;
      lastMove = now;
      mouse("move", e);
    });
    el.addEventListener("click", (e) => mouse("click", e));
    el.addEventListener("dblclick", (e) => mouse("dblclick", e));
    el.addEventListener("contextmenu", (e) => { e.preventDefault(); mouse("rclick", e); });
    el.addEventListener("wheel", (e) => {
      if (!control()) return;
      e.preventDefault();
      const p = rel(e);
      RP.sendEnv(ws, "input.mouse", { action: "scroll", x: p.x, y: p.y, scroll: -Math.sign(e.deltaY) });
    }, { passive: false });

    window.addEventListener("keydown", (e) => {
      if (!control() || document.querySelector('[data-panel="screen"]').hidden) return;
      const mods = [];
      if (e.ctrlKey) mods.push("ctrl");
      if (e.altKey) mods.push("alt");
      if (e.shiftKey) mods.push("shift");
      if (e.metaKey) mods.push("win");
      if (e.key.length === 1 && !e.ctrlKey && !e.altKey && !e.metaKey) {
        RP.sendEnv(ws, "input.key", { text: e.key });
      } else {
        RP.sendEnv(ws, "input.key", { key: e.key, modifiers: mods });
      }
      e.preventDefault();
    });
  }

  document.getElementById("screen-connect").addEventListener("click", connect);
  document.getElementById("screen-disconnect").addEventListener("click", disconnect);
  document.getElementById("screen-fullscreen").addEventListener("click", toggleFullscreen);
  document.getElementById("screen-quality").addEventListener("change", sendQuality);
  bindInput();

  window.Tabs.screen = { activate() {}, deactivate() { disconnect(); } };
})();
