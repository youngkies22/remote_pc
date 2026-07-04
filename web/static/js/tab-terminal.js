// Tab Terminal (Tahap 7).
(function () {
  let ws = null;
  const out = () => document.getElementById("term-output");
  const input = () => document.getElementById("term-input");

  function setConnected(on) {
    document.getElementById("term-connect").disabled = on;
    document.getElementById("term-disconnect").disabled = !on;
    input().disabled = !on;
    if (on) input().focus();
  }

  function append(text) {
    const el = out();
    el.textContent += text;
    el.scrollTop = el.scrollHeight;
  }

  function connect() {
    if (ws) return;
    out().textContent = "";
    ws = RP.ws("terminal");
    ws.onopen = () => {
      setConnected(true);
      RP.sendEnv(ws, "term.start", { shell: document.getElementById("term-shell").value });
    };
    ws.onmessage = (ev) => {
      let m;
      try { m = JSON.parse(ev.data); } catch { return; }
      if (m.type === "term.output" && m.payload) append(m.payload.data || "");
    };
    ws.onclose = () => { ws = null; setConnected(false); append("\n[koneksi terminal ditutup]\n"); };
    ws.onerror = () => append("\n[gagal terhubung]\n");
  }

  function disconnect() { if (ws) ws.close(); }

  input().addEventListener("keydown", (e) => {
    if (e.key !== "Enter") return;
    const line = input().value;
    input().value = "";
    RP.sendEnv(ws, "term.input", { data: line + "\r\n" });
  });

  document.getElementById("term-connect").addEventListener("click", connect);
  document.getElementById("term-disconnect").addEventListener("click", disconnect);

  window.Tabs.terminal = { activate() {}, deactivate() { disconnect(); } };
})();
