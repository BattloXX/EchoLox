const API = '/echolox/api';

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
  const p = 'ha_' + n;
  switch (type) {
    case 'switch': return [`${p}_on`];
    case 'dimmer': return [`${p}_on`, `${p}_brightness`];
    case 'color':  return [`${p}_on`, `${p}_brightness`, `${p}_hue`, `${p}_saturation`];
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
      if (cfg.mqtt_broker) {
        const m = document.getElementById('mqttBroker');
        if (m) m.value = cfg.mqtt_broker;
      }
    }
  } catch(_) {}

  // Settings form submit
  const form = document.getElementById('settingsForm');
  if (form) {
    form.addEventListener('submit', async e => {
      e.preventDefault();
      const body = {
        miniserver: document.getElementById('miniserver')?.value || '',
        transport:  document.getElementById('loxTransport')?.value || 'http',
        udp_port:   parseInt(document.getElementById('udpPort')?.value) || 7777,
        port:       parseInt(document.getElementById('serverPort')?.value) || 8079,
        mqtt_broker: document.getElementById('mqttBroker')?.value || '',
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
