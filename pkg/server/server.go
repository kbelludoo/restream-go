package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/kbelludoo/restream-go/pkg/relay"
)

type Server struct {
	port int
	mgr  *relay.Manager
}

func NewServer(port int, mgr *relay.Manager) *Server {
	return &Server{port: port, mgr: mgr}
}

func (s *Server) Start() {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		data := s.mgr.GetTelemetry()
		_ = json.NewEncoder(w).Encode(data)
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(htmlDashboard))
	})

	addr := fmt.Sprintf(":%d", s.port)
	log.Printf("[Web Server] Dashboard online em http://localhost:%d\n", s.port)

	go func() {
		if err := http.ListenAndServe(addr, mux); err != nil && err != http.ErrServerClosed {
			log.Printf("[Web Server] Erro no servidor: %v", err)
		}
	}()
}

const htmlDashboard = `<!DOCTYPE html>
<html lang="pt-BR">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>⚡ Restream Go — Dashboard</title>
  <link rel="icon" href="data:image/svg+xml,<svg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 100 100'><text y='.9em' font-size='90'>📡</text></svg>">
  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
  <link href="https://fonts.googleapis.com/css2?family=JetBrains+Mono:wght@400;600;700&family=Plus+Jakarta+Sans:wght@400;600;700;800&display=swap" rel="stylesheet">
  <style>
    :root {
      --bg: #06090e;
      --card-bg: rgba(13, 20, 31, 0.85);
      --card-border: rgba(83, 252, 24, 0.15);
      --accent: #53fc18;
      --accent-glow: rgba(83, 252, 24, 0.35);
      --text: #f3f4f6;
      --text-muted: #94a3b8;
      --text-dim: #64748b;
      --danger: #ff4d4d;
      --warn: #ffb300;
    }
    * { box-sizing: border-box; margin: 0; padding: 0; }
    body {
      background: var(--bg);
      background-image: radial-gradient(circle at 10% 10%, rgba(83, 252, 24, 0.05) 0%, transparent 40%), linear-gradient(to bottom, #06090e, #0a111a);
      color: var(--text);
      font-family: 'Plus Jakarta Sans', sans-serif;
      min-height: 100vh;
      padding: 24px;
    }
    .container { max-width: 1100px; margin: 0 auto; }
    .header {
      display: flex; justify-content: space-between; align-items: center;
      padding: 20px 24px; background: var(--card-bg);
      border: 1px solid var(--card-border); border-radius: 16px;
      margin-bottom: 24px; backdrop-filter: blur(12px);
      box-shadow: 0 10px 30px -10px rgba(0,0,0,0.6);
    }
    .logo-group { display: flex; align-items: center; gap: 14px; }
    .logo-icon {
      width: 44px; height: 44px; border-radius: 12px;
      background: rgba(83, 252, 24, 0.15); border: 1px solid var(--accent);
      display: flex; align-items: center; justify-content: center;
      font-size: 1.4rem; box-shadow: 0 0 16px var(--accent-glow);
    }
    .title h1 { font-size: 1.3rem; font-weight: 800; }
    .title p { font-size: 0.8rem; color: var(--text-muted); font-family: 'JetBrains Mono', monospace; }
    .live-pill {
      display: inline-flex; align-items: center; gap: 8px;
      padding: 6px 14px; border-radius: 999px; font-size: 0.75rem; font-weight: 700;
      background: rgba(83, 252, 24, 0.12); border: 1px solid var(--accent);
      color: var(--accent); font-family: 'JetBrains Mono', monospace;
    }
    .live-pill.off { background: rgba(255,77,77,0.12); border-color: var(--danger); color: var(--danger); }
    .pulse { width: 8px; height: 8px; background: var(--accent); border-radius: 50%; box-shadow: 0 0 8px var(--accent); animation: p 1.5s infinite; }
    .pulse.off { background: var(--danger); box-shadow: 0 0 8px var(--danger); }
    @keyframes p { 0%, 100% { opacity: 1; } 50% { opacity: 0.4; } }
    .stats-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(220px, 1fr)); gap: 16px; margin-bottom: 24px; }
    .stat-card { background: var(--card-bg); border: 1px solid var(--card-border); border-radius: 14px; padding: 18px 20px; }
    .stat-label { font-size: 0.75rem; color: var(--text-muted); text-transform: uppercase; letter-spacing: 0.05em; margin-bottom: 6px; }
    .stat-val { font-size: 1.6rem; font-weight: 800; font-family: 'JetBrains Mono', monospace; color: var(--accent); }
    .channels-card { background: var(--card-bg); border: 1px solid var(--card-border); border-radius: 16px; padding: 22px; margin-bottom: 24px; }
    .card-title { font-size: 1rem; font-weight: 700; margin-bottom: 16px; display: flex; align-items: center; justify-content: space-between; }
    .stream-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(300px, 1fr)); gap: 14px; }
    .stream-item { background: rgba(255,255,255,0.03); border: 1px solid rgba(255,255,255,0.06); border-radius: 12px; padding: 14px 16px; transition: all 0.2s; }
    .stream-item.active { border-color: var(--accent); background: rgba(83, 252, 24, 0.05); }
    .stream-head { display: flex; justify-content: space-between; align-items: center; margin-bottom: 8px; }
    .stream-name { font-weight: 700; font-size: 0.95rem; }
    .stream-status { font-size: 0.72rem; padding: 3px 8px; border-radius: 6px; font-weight: 600; font-family: 'JetBrains Mono', monospace; }
    .status-live { background: rgba(83, 252, 24, 0.2); color: var(--accent); border: 1px solid var(--accent); }
    .status-waiting { background: rgba(255, 160, 0, 0.15); color: var(--warn); border: 1px solid var(--warn); }
    .status-error { background: rgba(255,77,77,0.15); color: var(--danger); border: 1px solid var(--danger); }
    .status-offline { background: rgba(255,255,255,0.05); color: var(--text-dim); }
    .stream-meta { font-size: 0.8rem; color: var(--text-muted); font-family: 'JetBrains Mono', monospace; margin-top: 6px; }
    .log-box {
      background: #040608; border: 1px solid rgba(255,255,255,0.08);
      border-radius: 12px; padding: 16px; font-family: 'JetBrains Mono', monospace;
      font-size: 0.78rem; max-height: 220px; overflow-y: auto; color: #a1a1aa; line-height: 1.6;
    }
    .log-entry { margin-bottom: 4px; }
    .log-entry:last-child { color: var(--accent); }
  </style>
</head>
<body>
  <div class="container">
    <div class="header">
      <div class="logo-group">
        <div class="logo-icon">📡</div>
        <div class="title">
          <h1>⚡ Restream Go</h1>
          <p>Fan-out RTMP ultra-leve — OBS → Twitch/Kick/YouTube/Bilibili</p>
        </div>
      </div>
      <div class="live-pill off" id="livePill">
        <span class="pulse off" id="livePulse"></span>
        <span id="liveStatus">AGUARDANDO OBS</span>
      </div>
    </div>

    <div class="stats-grid">
      <div class="stat-card">
        <div class="stat-label">Memória RAM</div>
        <div class="stat-val" id="valRam">—</div>
      </div>
      <div class="stat-card">
        <div class="stat-label">Fonte (OBS)</div>
        <div class="stat-val" id="valSource">Offline</div>
      </div>
      <div class="stat-card">
        <div class="stat-label">Destinos Enviando</div>
        <div class="stat-val" id="valTargets">0 / 0</div>
      </div>
      <div class="stat-card">
        <div class="stat-label">Uptime</div>
        <div class="stat-val" id="valUptime">0s</div>
      </div>
    </div>

    <div class="channels-card">
      <div class="card-title">
        <span>Destinos</span>
        <span style="font-size: 0.75rem; color: var(--text-muted); font-family: 'JetBrains Mono', monospace;" id="sourceInfo">OBS: offline</span>
      </div>
      <div class="stream-grid" id="streamContainer">
        <div style="color: var(--text-muted); font-size: 0.85rem;">Carregando destinos...</div>
      </div>
    </div>

    <div class="channels-card">
      <div class="card-title">Console de Logs em Tempo Real</div>
      <div class="log-box" id="logBox">
        <div class="log-entry">Aguardando telemetria...</div>
      </div>
    </div>
  </div>

  <script>
    async function update() {
      try {
        const res = await fetch('/api/status');
        const data = await res.json();

        document.getElementById('valRam').textContent = (data.ramMb ? data.ramMb.toFixed(2) : '0') + ' MB';
        document.getElementById('valUptime').textContent = data.uptime || '0s';

        const src = data.source || {};
        const pill = document.getElementById('livePill');
        const pulse = document.getElementById('livePulse');
        const statusEl = document.getElementById('liveStatus');
        if (src.connected) {
          pill.classList.remove('off');
          pulse.classList.remove('off');
          statusEl.textContent = 'OBS CONECTADO';
          document.getElementById('valSource').textContent = 'Online (' + (src.tracks || 0) + ' tracks)';
          document.getElementById('sourceInfo').textContent = 'OBS: ' + (src.codec || 'H.264+AAC') + ' • ' + (src.frames || 0) + ' frames';
        } else {
          pill.classList.add('off');
          pulse.classList.add('off');
          statusEl.textContent = 'AGUARDANDO OBS';
          document.getElementById('valSource').textContent = 'Offline';
          document.getElementById('sourceInfo').textContent = 'OBS: offline';
        }

        const targets = data.targets || [];
        const live = targets.filter(t => t.status === 'enviando').length;
        document.getElementById('valTargets').textContent = live + ' / ' + targets.length;

        const container = document.getElementById('streamContainer');
        if (targets.length > 0) {
          container.innerHTML = targets.map(t => {
            let statusClass, statusText;
            switch (t.status) {
              case 'enviando': statusClass = 'status-live'; statusText = '● AO VIVO'; break;
              case 'conectando': statusClass = 'status-waiting'; statusText = 'CONECTANDO...'; break;
              case 'reconectando': statusClass = 'status-error'; statusText = 'RECONECTANDO'; break;
              case 'aguardando fonte': statusClass = 'status-waiting'; statusText = 'AGUARDANDO OBS'; break;
              case 'desativado': statusClass = 'status-offline'; statusText = 'DESATIVADO'; break;
              default: statusClass = 'status-offline'; statusText = t.status.toUpperCase();
            }
            const activeClass = t.status === 'enviando' ? 'active' : '';
            let meta = t.bitrateKbps + ' kbps • ' + (t.sentMb || 0).toFixed(1) + ' MB';
            if (t.uptime) meta += ' • ' + t.uptime;
            if (t.reconnects > 0) meta += ' • ' + t.reconnects + ' reconexões';
            if (t.dropped > 0) meta += ' • ' + t.dropped + ' descartes';
            let errLine = t.lastError ? '<div class="stream-meta" style="color: var(--danger);">' + t.lastError + '</div>' : '';
            return '<div class="stream-item ' + activeClass + '">' +
              '<div class="stream-head">' +
                '<span class="stream-name">' + t.name + '</span>' +
                '<span class="stream-status ' + statusClass + '">' + statusText + '</span>' +
              '</div>' +
              '<div class="stream-meta">' + meta + '</div>' + errLine +
            '</div>';
          }).join('');
        }

        if (data.logs && data.logs.length > 0) {
          const logBox = document.getElementById('logBox');
          logBox.innerHTML = data.logs.map(l => '<div class="log-entry">' + l + '</div>').join('');
          logBox.scrollTop = logBox.scrollHeight;
        }
      } catch (e) {
        console.error(e);
      }
    }

    update();
    setInterval(update, 3000);
  </script>
</body>
</html>`
