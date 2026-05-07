const API = '/echolox/api';

function toggleNav() {
  document.getElementById('navLinks').classList.toggle('open');
}

// ── About ──────────────────────────────────────────────────────────────────

async function loadAbout() {
  // Bridge info from /api/config (Hue endpoint)
  try {
    const res = await fetch('/api/nouser/config');
    if (res.ok) {
      const cfg = await res.json();
      const el = document.getElementById('bridgeInfo');
      if (el) {
        el.innerHTML = `
          <dt>Bridge ID</dt><dd style="font-family:monospace">${cfg.bridgeid || '—'}</dd>
          <dt>IP-Adresse</dt><dd>${cfg.ipaddress || '—'}</dd>
          <dt>API-Version</dt><dd>${cfg.apiversion || '—'}</dd>
          <dt>Modell</dt><dd>${cfg.modelid || '—'}</dd>`;
      }
    }
  } catch(_) {}

  // Version from plugin.cfg via status endpoint or hardcoded
  const verEl = document.getElementById('version');
  if (verEl) verEl.textContent = '1.0.0';
}

// ── Devices ────────────────────────────────────────────────────────────────

async function loadDevices() {
  const res = await fetch(`${API}/devices`);
  const devices = await res.json();
  const tbody = document.getElementById('deviceBody');
  if (!tbody) return;
  tbody.innerHTML = '';
  (devices || []).forEach(d => {
    const vis = Object.entries(d.virtual_inputs || {})
      .map(([k,v]) => `<span style="font-family:monospace;font-size:0.85em">${v}</span>`)
      .join('<br>');
    const tr = document.createElement('tr');
    tr.innerHTML = `
      <td>${d.name}</td>
      <td><span class="type-badge">${d.type}</span></td>
      <td>${d.transport || 'http'}</td>
      <td>${vis}</td>
      <td>
        <a href="device.html?id=${d.id}" class="btn action-btn">Bearbeiten</a>
        <button onclick="deleteDevice('${d.id}')" class="btn action-btn">Löschen</button>
        <button onclick="testDevice('${d.id}')" class="btn action-btn">Testen</button>
      </td>`;
    tbody.appendChild(tr);
  });
}

async function deleteDevice(id) {
  if (!confirm('Gerät wirklich löschen?')) return;
  await fetch(`${API}/devices/${id}`, { method: 'DELETE' });
  loadDevices();
}

async function testDevice(id) {
  const res = await fetch(`${API}/devices/${id}/test`, { method: 'POST' });
  const data = await res.json();
  alert(data.status === 'ok' ? 'Test erfolgreich!' : 'Fehler: ' + JSON.stringify(data.errors));
}

// ── Device Form ────────────────────────────────────────────────────────────

let currentDeviceId = null;

function initDeviceForm() {
  const params = new URLSearchParams(window.location.search);
  const id = params.get('id');
  const nameEl = document.getElementById('name');
  const typeEl = document.getElementById('type');
  const transportEl = document.getElementById('transport');

  function updatePreview() {
    const name = nameEl.value;
    const type = typeEl.value;
    const preview = document.getElementById('viPreview');
    if (!name) { preview.innerHTML = '<em>Bitte Name eingeben</em>'; return; }
    const vis = generateVIs(name, type);
    preview.innerHTML = vis.map(v => `<div>${v}</div>`).join('');
  }

  nameEl.addEventListener('input', updatePreview);
  typeEl.addEventListener('change', updatePreview);

  if (id) {
    currentDeviceId = id;
    document.getElementById('pageTitle').textContent = 'Gerät bearbeiten';
    const testBtn = document.getElementById('testDeviceBtn');
    if (testBtn) testBtn.style.display = '';
    fetch(`${API}/devices/${id}`).then(r => r.json()).then(d => {
      nameEl.value = d.name;
      typeEl.value = d.type;
      transportEl.value = d.transport || 'http';
      updatePreview();
    });
  }

  document.getElementById('deviceForm').addEventListener('submit', async e => {
    e.preventDefault();
    const body = {
      name: nameEl.value,
      type: typeEl.value,
      transport: transportEl.value,
    };
    if (id) {
      body.id = id;
      await fetch(`${API}/devices/${id}`, { method: 'PUT', headers: {'Content-Type':'application/json'}, body: JSON.stringify(body) });
    } else {
      await fetch(`${API}/devices`, { method: 'POST', headers: {'Content-Type':'application/json'}, body: JSON.stringify(body) });
    }
    window.location.href = 'index.html';
  });
}

async function testMiniserver() {
  const result = document.getElementById('msTestResult');
  result.className = 'test-result info';
  result.textContent = 'Verbinde mit Miniserver…';
  try {
    const res = await fetch(`${API}/verify`, { method: 'POST' });
    const data = await res.json();
    if (data.status === 'ok') {
      result.className = 'test-result ok';
      result.textContent = '✓ Miniserver erreichbar';
    } else {
      result.className = 'test-result error';
      result.textContent = '✗ ' + (data.error || 'Verbindung fehlgeschlagen');
    }
  } catch(e) {
    result.className = 'test-result error';
    result.textContent = '✗ ' + e.message;
  }
}

async function testCurrentDevice() {
  if (!currentDeviceId) return;
  const result = document.getElementById('msTestResult');
  result.className = 'test-result info';
  result.textContent = 'Sende Testwert an Loxone…';
  try {
    const res = await fetch(`${API}/devices/${currentDeviceId}/test`, { method: 'POST' });
    const data = await res.json();
    if (data.status === 'ok') {
      result.className = 'test-result ok';
      result.textContent = '✓ Gerät erfolgreich getestet';
    } else {
      result.className = 'test-result error';
      result.textContent = '✗ Fehler: ' + JSON.stringify(data.errors || data);
    }
  } catch(e) {
    result.className = 'test-result error';
    result.textContent = '✗ ' + e.message;
  }
}

function normalizeName(name) {
  const map = {'ä':'ae','ö':'oe','ü':'ue','Ä':'ae','Ö':'oe','Ü':'ue','ß':'ss'};
  let r = name.replace(/[äöüÄÖÜß]/g, c => map[c] || c);
  r = r.toLowerCase().replace(/[^a-z0-9]+/g, '_').replace(/^_|_$/g, '');
  return r;
}

function generateVIs(name, type) {
  const n = normalizeName(name);
  const p = 'echolox_' + n;
  switch (type) {
    case 'switch': return [`${p}_on`, `${p}_off`];
    case 'dimmer': return [`${p}_on`, `${p}_off`, `${p}_brightness`];
    case 'color':  return [`${p}_on`, `${p}_off`, `${p}_brightness`, `${p}_hue`, `${p}_saturation`];
    case 'scene':  return [`${p}_activate`];
    default: return [];
  }
}

// ── Status ─────────────────────────────────────────────────────────────────

let allRows = [];
let currentFilter = 'all';

async function loadStatus() {
  const res = await fetch(`${API}/status`);
  allRows = await res.json() || [];
  renderStatus();
}

async function refreshStatus() {
  await fetch(`${API}/status`, { method: 'POST' });
  await loadStatus();
}

function filterStatus(filter) {
  currentFilter = filter;
  document.querySelectorAll('.filter-row .btn').forEach(b => b.classList.remove('active'));
  const el = document.getElementById('f-' + filter);
  if (el) el.classList.add('active');
  renderStatus();
}

function renderStatus() {
  const search = (document.getElementById('searchInput')?.value || '').toLowerCase();
  const tbody = document.getElementById('statusBody');
  if (!tbody) return;
  const icons = { ok: '✅', not_found: '🟠', access_denied: '🔴', not_sent: '⬜' };
  tbody.innerHTML = '';
  (allRows || [])
    .filter(r => currentFilter === 'all' || r.status === currentFilter)
    .filter(r => !search || r.name.toLowerCase().includes(search) || r.device_name.toLowerCase().includes(search))
    .forEach(r => {
      const tr = document.createElement('tr');
      tr.innerHTML = `
        <td><span class="status-${r.status}">${icons[r.status] || '?'} ${r.status}</span></td>
        <td style="font-family:monospace">${r.name}</td>
        <td>${r.device_name}</td>
        <td>${r.last_value || '—'}</td>
        <td>${r.last_sent || 'nie'}</td>`;
      tbody.appendChild(tr);
    });
}

// ── Settings ───────────────────────────────────────────────────────────────

async function loadSettings() {
  // Load miniserver list from LoxBerry
  const msRes = await fetch(`${API}/miniservers`);
  const miniservers = await msRes.json() || [];
  const sel = document.getElementById('miniserver');
  if (sel) {
    sel.innerHTML = miniservers.length
      ? miniservers.map(ms => `<option value="${ms.id}">${ms.name} (${ms.ip})</option>`).join('')
      : '<option value="">— kein Miniserver gefunden —</option>';
  }

  // Load current config values
  try {
    const cfgRes = await fetch(`${API}/settings`);
    if (cfgRes.ok) {
      const cfg = await cfgRes.json();
      if (cfg.miniserver && sel) {
        const opt = sel.querySelector(`option[value="${cfg.miniserver}"]`);
        if (opt) opt.selected = true;
      }
      if (cfg.transport) {
        const t = document.getElementById('loxTransport');
        if (t) t.value = cfg.transport;
      }
      if (cfg.udp_port) {
        const u = document.getElementById('udpPort');
        if (u) u.value = cfg.udp_port;
      }
      if (cfg.port) {
        const p = document.getElementById('serverPort');
        if (p) p.value = cfg.port;
      }
      if (cfg.discovery_port !== undefined) {
        const dp = document.getElementById('discoveryPort');
        if (dp) dp.value = cfg.discovery_port;
      }
      if (cfg.mqtt_broker) {
        const m = document.getElementById('mqttBroker');
        if (m) m.value = cfg.mqtt_broker;
      }
      const mu = document.getElementById('mqttUsername');
      if (mu && cfg.mqtt_username) mu.value = cfg.mqtt_username;
    }
  } catch(_) {}

  // Settings form submit
  const form = document.getElementById('settingsForm');
  if (form) {
    form.addEventListener('submit', async e => {
      e.preventDefault();
      const pwEl = document.getElementById('mqttPassword');
      const body = {
        miniserver: document.getElementById('miniserver')?.value || '',
        transport:  document.getElementById('loxTransport')?.value || 'http',
        udp_port:   parseInt(document.getElementById('udpPort')?.value) || 7777,
        port:           parseInt(document.getElementById('serverPort')?.value) || 8079,
        discovery_port: parseInt(document.getElementById('discoveryPort')?.value) || 0,
        mqtt_broker:    document.getElementById('mqttBroker')?.value || '',
        mqtt_username:  document.getElementById('mqttUsername')?.value || '',
        mqtt_password:  pwEl ? pwEl.value : undefined,
      };
      const res = await fetch(`${API}/settings`, {
        method: 'POST',
        headers: {'Content-Type':'application/json'},
        body: JSON.stringify(body),
      });
      const data = await res.json();
      const notice = document.getElementById('saveNotice');
      if (notice) notice.textContent = data.message || (res.ok ? '✓ Gespeichert' : '✗ Fehler');
      setTimeout(() => { if (notice) notice.textContent = ''; }, 4000);
    });
  }
}

async function testConnection() {
  const result = document.getElementById('testResult');
  result.className = 'test-result info';
  result.textContent = 'Verbinde…';
  try {
    const res = await fetch(`${API}/verify`, { method: 'POST' });
    const data = await res.json();
    if (data.status === 'ok') {
      result.className = 'test-result ok';
      result.textContent = '✓ Miniserver erreichbar';
    } else {
      result.className = 'test-result error';
      result.textContent = '✗ ' + (data.error || 'Verbindung fehlgeschlagen');
    }
  } catch(e) {
    result.className = 'test-result error';
    result.textContent = '✗ ' + e.message;
  }
}

// ── Restart ────────────────────────────────────────────────────────────────

async function restartService() {
  const btn = document.getElementById('restartBtn');
  const result = document.getElementById('restartResult');
  if (btn) { btn.disabled = true; btn.textContent = 'Neustart läuft…'; }
  if (result) { result.className = 'test-result info'; result.textContent = 'EchoLox wird neu gestartet…'; }
  try {
    await fetch(`${API}/restart`, { method: 'POST' });
  } catch(_) {}
  // Poll until service responds again (max 30s)
  const start = Date.now();
  while (Date.now() - start < 30000) {
    await new Promise(r => setTimeout(r, 1500));
    try {
      const r = await fetch(`${API}/status`, { signal: AbortSignal.timeout(2000) });
      if (r.ok) {
        if (result) { result.className = 'test-result ok'; result.textContent = '✓ EchoLox erfolgreich neu gestartet'; }
        if (btn) { btn.disabled = false; btn.textContent = 'Dienst neu starten'; }
        return;
      }
    } catch(_) {}
  }
  if (result) { result.className = 'test-result error'; result.textContent = '✗ Neustart dauert länger als erwartet — Seite neu laden'; }
  if (btn) { btn.disabled = false; btn.textContent = 'Dienst neu starten'; }
}

// ── Backup ─────────────────────────────────────────────────────────────────

async function createLocalBackup() {
  const result = document.getElementById('backupResult');
  if (result) { result.className = 'test-result info'; result.textContent = 'Backup wird erstellt…'; }
  try {
    const res = await fetch(`${API}/backup`, { method: 'POST' });
    const data = await res.json();
    if (res.ok) {
      if (result) { result.className = 'test-result ok'; result.textContent = `✓ Backup gespeichert: ${data.file}`; }
      loadBackupList();
    } else {
      if (result) { result.className = 'test-result error'; result.textContent = '✗ ' + (data.error || 'Fehler'); }
    }
  } catch(e) {
    if (result) { result.className = 'test-result error'; result.textContent = '✗ ' + e.message; }
  }
}

async function loadBackupList() {
  const container = document.getElementById('backupList');
  if (!container) return;
  try {
    const res = await fetch(`${API}/backup`);
    const list = await res.json() || [];
    if (!list.length) {
      container.innerHTML = '<em style="color:#999">Keine lokalen Backups vorhanden</em>';
      return;
    }
    container.innerHTML = list.map(b => `
      <div class="backup-item">
        <span class="ts">${b.display}</span>
        <div style="display:flex;gap:8px">
          <button onclick="restoreLocalBackup('${b.file}')" class="btn action-btn">Wiederherstellen</button>
          <button onclick="deleteLocalBackup('${b.file}')" class="btn action-btn btn-danger">Löschen</button>
        </div>
      </div>`).join('');
  } catch(e) {
    container.innerHTML = '<em style="color:#c00">Fehler: ' + e.message + '</em>';
  }
}

async function restoreLocalBackup(file) {
  if (!confirm(`Backup "${file}" wiederherstellen? Die aktuellen Daten werden überschrieben.`)) return;
  const result = document.getElementById('restoreResult');
  if (result) { result.className = 'test-result info'; result.textContent = 'Wiederherstellung läuft…'; }
  try {
    const res = await fetch(`${API}/backup/restore-local`, {
      method: 'POST',
      headers: {'Content-Type': 'application/json'},
      body: JSON.stringify({ file })
    });
    const data = await res.json();
    if (res.ok) {
      if (result) { result.className = 'test-result ok'; result.textContent = `✓ Wiederhergestellt: ${(data.restored||[]).join(', ')} — bitte EchoLox neu starten`; }
    } else {
      if (result) { result.className = 'test-result error'; result.textContent = '✗ ' + (data || 'Fehler'); }
    }
  } catch(e) {
    if (result) { result.className = 'test-result error'; result.textContent = '✗ ' + e.message; }
  }
}

async function deleteLocalBackup(file) {
  if (!confirm(`Backup "${file}" löschen?`)) return;
  try {
    await fetch(`${API}/backup/delete`, {
      method: 'POST',
      headers: {'Content-Type': 'application/json'},
      body: JSON.stringify({ file })
    });
    loadBackupList();
  } catch(_) {}
}

function handleRestoreDrop(event) {
  event.preventDefault();
  document.getElementById('restoreDropZone').classList.remove('dragover');
  const file = event.dataTransfer.files[0];
  if (file) handleRestoreFile(file);
}

async function handleRestoreFile(file) {
  if (!file) return;
  if (!confirm(`Backup "${file.name}" wiederherstellen? Die aktuellen Daten werden überschrieben.`)) return;
  const result = document.getElementById('restoreResult');
  if (result) { result.className = 'test-result info'; result.textContent = 'Wird hochgeladen und wiederhergestellt…'; }
  const form = new FormData();
  form.append('backup', file);
  try {
    const res = await fetch(`${API}/backup/restore`, { method: 'POST', body: form });
    const data = await res.json();
    if (res.ok) {
      if (result) { result.className = 'test-result ok'; result.textContent = `✓ Wiederhergestellt: ${(data.restored||[]).join(', ')} — bitte EchoLox neu starten`; }
    } else {
      if (result) { result.className = 'test-result error'; result.textContent = '✗ ' + (data || 'Fehler'); }
    }
  } catch(e) {
    if (result) { result.className = 'test-result error'; result.textContent = '✗ ' + e.message; }
  }
}

// ── Import ─────────────────────────────────────────────────────────────────

let importData = [];

function initImport() {
  const fileInput = document.getElementById('fileInput');
  const dropZone = document.getElementById('dropZone');

  fileInput.addEventListener('change', e => handleFile(e.target.files[0]));

  dropZone.addEventListener('dragover', e => { e.preventDefault(); dropZone.classList.add('dragover'); });
  dropZone.addEventListener('dragleave', () => dropZone.classList.remove('dragover'));
  dropZone.addEventListener('drop', e => {
    e.preventDefault();
    dropZone.classList.remove('dragover');
    handleFile(e.dataTransfer.files[0]);
  });
}

async function handleFile(file) {
  if (!file) return;
  document.getElementById('fileName').textContent = file.name;
  const formData = new FormData();
  formData.append('file', file);
  const res = await fetch(`${API}/import/preview`, { method: 'POST', body: formData });
  const result = await res.json();

  importData = result.imported || [];
  const tbody = document.getElementById('previewBody');
  tbody.innerHTML = '';

  importData.forEach(d => {
    const vis = Object.values(d.virtual_inputs || {}).join(', ');
    const tr = document.createElement('tr');
    tr.innerHTML = `<td>${d.name}</td><td style="font-family:monospace;font-size:0.85em">${vis}</td><td>✅ OK</td>`;
    tbody.appendChild(tr);
  });

  (result.skipped || []).forEach(s => {
    const tr = document.createElement('tr');
    tr.innerHTML = `<td>${s.name}</td><td>—</td><td>⚠️ ${s.reason}</td>`;
    tbody.appendChild(tr);
  });

  document.getElementById('previewSection').style.display = 'block';
}

// ── Logs ───────────────────────────────────────────────────────────────────

let logAutoRefreshTimer = null;
let logAutoRefresh = false;
let currentLogLevel = 'info';

async function loadLogs() {
  try {
    const res = await fetch(`${API}/logs`);
    const data = await res.json();
    currentLogLevel = data.level || 'info';
    updateLevelUI(currentLogLevel);
    renderLogs(data.entries || []);
  } catch(e) {
    const tbody = document.getElementById('logBody');
    if (tbody) tbody.innerHTML = `<tr><td colspan="3" style="color:#c00">Fehler: ${e.message}</td></tr>`;
  }
}

function renderLogs(entries) {
  const tbody = document.getElementById('logBody');
  if (!tbody) return;
  if (!entries.length) {
    tbody.innerHTML = '<tr><td colspan="3" style="color:#999;text-align:center">Keine Einträge</td></tr>';
    return;
  }
  tbody.innerHTML = entries.map(e => {
    const color = e.level === 'DEBUG' ? '#999' : e.level === 'INFO' ? '#333' : '#b00';
    return `<tr>
      <td style="white-space:nowrap;color:#888">${e.t}</td>
      <td style="color:${color};font-weight:${e.level==='DEBUG'?'normal':'bold'}">${e.level}</td>
      <td style="word-break:break-word">${escapeHtml(e.msg)}</td>
    </tr>`;
  }).join('');
  const meta = document.getElementById('logMeta');
  if (meta) meta.textContent = `${entries.length} Einträge`;
  // Auto-scroll to bottom
  const container = document.getElementById('logContainer');
  if (container) container.scrollTop = container.scrollHeight;
}

function updateLevelUI(level) {
  const badge = document.getElementById('levelBadge');
  const btn = document.getElementById('toggleLevelBtn');
  if (badge) {
    badge.textContent = level.toUpperCase();
    badge.style.background = level === 'debug' ? '#fff3cd' : '#e0ecd0';
    badge.style.color = level === 'debug' ? '#856404' : '#3a6010';
  }
  if (btn) btn.textContent = level === 'debug' ? 'Debug deaktivieren' : 'Debug aktivieren';
}

async function toggleLogLevel() {
  const newLevel = currentLogLevel === 'debug' ? 'info' : 'debug';
  try {
    const res = await fetch(`${API}/logs/level`, {
      method: 'POST',
      headers: {'Content-Type': 'application/json'},
      body: JSON.stringify({level: newLevel})
    });
    const data = await res.json();
    currentLogLevel = data.level || newLevel;
    updateLevelUI(currentLogLevel);
    loadLogs();
  } catch(e) { console.error(e); }
}

function toggleAutoRefresh() {
  logAutoRefresh = !logAutoRefresh;
  const btn = document.getElementById('autoRefreshBtn');
  if (logAutoRefresh) {
    if (btn) { btn.textContent = 'Auto-Refresh stoppen'; btn.classList.add('active'); }
    logAutoRefreshTimer = setInterval(loadLogs, 5000);
  } else {
    if (btn) { btn.textContent = 'Auto-Refresh'; btn.classList.remove('active'); }
    clearInterval(logAutoRefreshTimer);
    logAutoRefreshTimer = null;
  }
}

function escapeHtml(str) {
  return String(str).replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;');
}

async function confirmImport() {
  const res = await fetch(`${API}/import/confirm`, {
    method: 'POST',
    headers: {'Content-Type': 'application/json'},
    body: JSON.stringify(importData)
  });
  const result = await res.json();
  document.getElementById('importResult').textContent = `✅ ${result.imported} Geräte importiert`;
  setTimeout(() => window.location.href = 'index.html', 1500);
}

function resetImport() {
  importData = [];
  document.getElementById('previewSection').style.display = 'none';
  document.getElementById('fileName').textContent = '';
  document.getElementById('fileInput').value = '';
}

// ── MQTT Subscriptions ─────────────────────────────────────────────────────

async function loadMQTTSubscriptions() {
  const tbody = document.getElementById('mqttSubBody');
  if (!tbody) return;
  try {
    const res = await fetch(`${API}/mqtt/subscriptions`);
    const subs = await res.json() || [];
    if (!subs.length) {
      tbody.innerHTML = '<tr><td colspan="3" style="color:#999;text-align:center">Keine Subscriptions konfiguriert</td></tr>';
      return;
    }
    tbody.innerHTML = subs.map(s => `
      <tr>
        <td style="font-family:monospace">${escapeHtml(s.topic)}</td>
        <td style="font-family:monospace">${escapeHtml(s.target)}</td>
        <td><button onclick="deleteMQTTSubscription('${escapeHtml(s.topic)}')" class="btn action-btn">Löschen</button></td>
      </tr>`).join('');
  } catch(e) {
    tbody.innerHTML = `<tr><td colspan="3" style="color:#c00">Fehler: ${e.message}</td></tr>`;
  }
}

async function addMQTTSubscription() {
  const topic  = document.getElementById('newSubTopic')?.value?.trim();
  const target = document.getElementById('newSubTarget')?.value?.trim();
  const result = document.getElementById('mqttSubResult');
  if (!topic || !target) {
    if (result) { result.className = 'test-result error'; result.textContent = 'Topic und Ziel-VI sind erforderlich'; }
    return;
  }
  try {
    const res = await fetch(`${API}/mqtt/subscriptions`, {
      method: 'POST',
      headers: {'Content-Type':'application/json'},
      body: JSON.stringify({topic, target}),
    });
    const data = await res.json();
    if (res.ok) {
      if (result) { result.className = 'test-result ok'; result.textContent = '✓ Subscription hinzugefügt'; }
      document.getElementById('newSubTopic').value = '';
      document.getElementById('newSubTarget').value = '';
      loadMQTTSubscriptions();
    } else {
      if (result) { result.className = 'test-result error'; result.textContent = '✗ ' + (data.error || 'Fehler'); }
    }
  } catch(e) {
    if (result) { result.className = 'test-result error'; result.textContent = '✗ ' + e.message; }
  }
  setTimeout(() => { if (result) result.textContent = ''; }, 4000);
}

async function deleteMQTTSubscription(topic) {
  if (!confirm(`Subscription für "${topic}" löschen?`)) return;
  try {
    const res = await fetch(`${API}/mqtt/subscriptions`, {
      method: 'DELETE',
      headers: {'Content-Type':'application/json'},
      body: JSON.stringify({topic}),
    });
    if (res.ok) loadMQTTSubscriptions();
  } catch(_) {}
}
