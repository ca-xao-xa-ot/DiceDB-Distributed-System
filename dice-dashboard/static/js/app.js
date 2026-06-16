const navItems = document.querySelectorAll('.nav-item');
const views = document.querySelectorAll('.view');

const overviewEls = {
  totalNodes: document.getElementById('totalNodes'),
  nodeOnline: document.getElementById('nodeOnline'),
  nodeOffline: document.getElementById('nodeOffline'),
  totalKeys: document.getElementById('totalKeys'),
  replicatedCommands: document.getElementById('replicatedCommands'),
  averageLatency: document.getElementById('averageLatency'),
  replicationSuccessRate: document.getElementById('replicationSuccessRate'),
  storageMode: document.getElementById('storageMode'),
  lastRefreshLabel: document.getElementById('lastRefreshLabel')
};

const statsEls = {
  heartbeatChecks: document.getElementById('heartbeatChecks'),
  failedReplications: document.getElementById('failedReplications'),
  maxReplicationDelay: document.getElementById('maxReplicationDelay'),
  lastReplicationAt: document.getElementById('lastReplicationAt')
};

const replicationEls = {
  replica1Delay: document.getElementById('replica1Delay'),
  replica2Delay: document.getElementById('replica2Delay'),
  replicationMaxDelay: document.getElementById('replicationMaxDelay'),
  replicationCommandCount: document.getElementById('replicationCommandCount')
};

const healthBanner = {
  container: document.getElementById('healthBanner'),
  title: document.getElementById('healthBannerTitle'),
  text: document.getElementById('healthBannerText')
};

const nodeCards = document.getElementById('nodeCards');
const heartbeatTable = document.getElementById('heartbeatTable');
const activityLog = document.getElementById('activityLog');
const replicationLog = document.getElementById('replicationLog');
const kvResult = document.getElementById('kvResult');
const replicationForm = document.getElementById('replicationForm');
const keyInput = document.getElementById('keyInput');
const valueInput = document.getElementById('valueInput');
const clearLogBtn = document.getElementById('clearLogBtn');
const faultPanel = document.getElementById('faultPanel');

let latencyChart;
let replicationChart;

navItems.forEach((item) => {
  item.addEventListener('click', () => {
    navItems.forEach((btn) => btn.classList.remove('active'));
    views.forEach((view) => view.classList.remove('active'));
    item.classList.add('active');
    document.getElementById(item.dataset.view).classList.add('active');
  });
});

async function fetchJSON(url, options = {}) {
  const res = await fetch(url, options);
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new Error(body.error || `Request failed: ${res.status}`);
  }
  return res.json();
}

function statusClass(status) {
  return String(status).toLowerCase() === 'online' ? 'online' : 'offline';
}

function healthClass(status) {
  const value = String(status).toLowerCase();
  if (value === 'healthy') return 'healthy';
  if (value === 'warning') return 'warning';
  return 'critical';
}

function healthCardClass(status) {
  return `health-card-${healthClass(status)}`;
}

function severityClass(severity) {
  return `log-item-${String(severity || 'INFO').toLowerCase()}`;
}

function formatPercent(value) {
  return `${Number(value || 0).toFixed(1)}%`;
}

function updateHealthBanner(nodes) {
  const hasCritical = nodes.some((node) => node.health_status === 'Critical');
  const hasWarning = nodes.some((node) => node.health_status === 'Warning');
  healthBanner.container.className = 'health-banner';
  if (hasCritical) {
    healthBanner.container.classList.add('critical');
    healthBanner.title.textContent = 'Cluster critical';
    healthBanner.text.textContent = 'One or more replicas are unavailable or replication is unstable.';
    return;
  }
  if (hasWarning) {
    healthBanner.container.classList.add('warning');
    healthBanner.title.textContent = 'Cluster warning';
    healthBanner.text.textContent = 'Latency or replication delay is elevated. Monitor the replicas.';
    return;
  }
  healthBanner.container.classList.add('healthy');
  healthBanner.title.textContent = 'Cluster stable';
  healthBanner.text.textContent = 'Heartbeat, replication and node health are currently normal.';
}

function renderOverview(data) {
  overviewEls.totalNodes.textContent = data.total_nodes;
  overviewEls.nodeOnline.textContent = data.node_online;
  overviewEls.nodeOffline.textContent = data.node_offline;
  overviewEls.totalKeys.textContent = data.total_keys;
  overviewEls.replicatedCommands.textContent = data.replicated_commands;
  overviewEls.averageLatency.textContent = `${data.average_latency_ms} ms`;
  overviewEls.storageMode.textContent = data.storage_mode;
  overviewEls.lastRefreshLabel.textContent = data.last_updated;
}

function renderClusterStats(data) {
  statsEls.heartbeatChecks.textContent = data.total_heartbeat_checks;
  statsEls.failedReplications.textContent = data.failed_replications;
  statsEls.maxReplicationDelay.textContent = `${data.max_replication_delay_ms} ms`;
  statsEls.lastReplicationAt.textContent = data.last_replication_at;
  overviewEls.replicationSuccessRate.textContent = formatPercent(data.replication_success_rate);
}

function renderNodes(nodes) {
  nodeCards.innerHTML = nodes.map((node) => `
    <article class="node-card ${healthCardClass(node.health_status)}">
      <div class="node-top">
        <div>
          <h3>${node.name}</h3>
          <span class="role-tag">${node.role}</span>
        </div>
        <div class="node-status-group">
          <span class="status-pill ${statusClass(node.status)}">${node.status}</span>
          <span class="health-pill ${healthClass(node.health_status)}">${node.health_status}</span>
        </div>
      </div>
      <div class="node-meta">
        <div class="meta-box"><span>Port</span><strong>${node.port}</strong></div>
        <div class="meta-box"><span>Keys</span><strong>${node.key_count}</strong></div>
        <div class="meta-box"><span>Latency</span><strong>${node.latency_ms} ms</strong></div>
        <div class="meta-box"><span>Repl. Delay</span><strong>${node.replication_delay_ms} ms</strong></div>
        <div class="meta-box"><span>Uptime</span><strong>${node.uptime_seconds}s</strong></div>
        <div class="meta-box"><span>Health Score</span><strong>${node.health_score}</strong></div>
        <div class="meta-box"><span>Repl. Cmds</span><strong>${node.replicated_commands}</strong></div>
        <div class="meta-box"><span>Last Seen</span><strong>${node.last_seen}</strong></div>
        <div class="meta-box"><span>Memory</span><strong>${node.memory_usage}</strong></div>
      </div>
    </article>
  `).join('');

  faultPanel.innerHTML = nodes.filter((node) => node.role === 'replica').map((node) => `
    <article class="fault-card">
      <h3>${node.name}</h3>
      <p>Status: <strong>${node.status}</strong></p>
      <div class="fault-actions">
        <button onclick="failNode('${node.name}')">Fail</button>
        <button onclick="recoverNode('${node.name}')">Recover</button>
      </div>
      <div class="latency-actions">
        <input id="latency-${node.name}" type="number" min="0" placeholder="ms" />
        <button onclick="injectLatency('${node.name}')">Inject Latency</button>
        <button class="secondary-btn" onclick="clearLatency('${node.name}')">Clear</button>
      </div>
    </article>
  `).join('');

  updateHealthBanner(nodes);

  const replica1 = nodes.find((node) => node.name === 'Replica1');
  const replica2 = nodes.find((node) => node.name === 'Replica2');
  replicationEls.replica1Delay.textContent = `${replica1?.replication_delay_ms || 0} ms`;
  replicationEls.replica2Delay.textContent = `${replica2?.replication_delay_ms || 0} ms`;
  replicationEls.replicationCommandCount.textContent = nodes.reduce((sum, node) => sum + (node.replicated_commands || 0), 0);
}

function renderHeartbeat(rows) {
  heartbeatTable.innerHTML = rows.map((row) => `
    <tr>
      <td>${row.node}</td>
      <td><span class="status-pill ${statusClass(row.status)}">${row.status}</span></td>
      <td>${row.latency_ms} ms</td>
      <td>${row.uptime}</td>
      <td><span class="health-pill ${healthClass(row.health_status)}">${row.health_status} (${row.health_score})</span></td>
      <td>${row.consecutive_failures}</td>
      <td>${row.last_heartbeat}</td>
      <td>${row.last_seen}</td>
    </tr>
  `).join('');
}

function renderActivity(logs) {
  activityLog.innerHTML = logs.map((log) => `
    <article class="log-item ${severityClass(log.severity)}">
      <div class="log-top"><strong>${log.type}</strong><span>${log.timestamp}</span></div>
      <p>${log.message}</p>
    </article>
  `).join('');
}

function renderReplication(logs) {
  replicationLog.innerHTML = logs.map((log) => `
    <article class="log-item ${log.success ? 'log-item-success' : 'log-item-error'}">
      <div class="log-top"><strong>${log.action} → ${log.target}</strong><span>${log.timestamp}</span></div>
      <p>${log.message}</p>
      <small>Delay: ${log.delay_ms} ms</small>
    </article>
  `).join('');
  const maxDelay = logs.reduce((max, log) => Math.max(max, log.delay_ms || 0), 0);
  replicationEls.replicationMaxDelay.textContent = `${maxDelay} ms`;
}

function renderTrendCharts(history) {
  const latencyLabels = history.latency_trend.map((point) => point.timestamp);
  const latencyData = history.latency_trend.map((point) => point.value);
  const replicationLabels = history.replication_delay_trend.map((point) => point.timestamp);
  const replicationData = history.replication_delay_trend.map((point) => point.value);

  if (latencyChart) latencyChart.destroy();
  if (replicationChart) replicationChart.destroy();

  latencyChart = new Chart(document.getElementById('latencyTrendChart'), {
    type: 'line',
    data: { labels: latencyLabels, datasets: [{ label: 'Latency (ms)', data: latencyData, borderColor: '#2962ff', backgroundColor: 'rgba(41,98,255,0.12)', tension: 0.35, fill: true }] },
    options: { responsive: true, plugins: { legend: { display: true } } }
  });

  replicationChart = new Chart(document.getElementById('replicationTrendChart'), {
    type: 'line',
    data: { labels: replicationLabels, datasets: [{ label: 'Replication Delay (ms)', data: replicationData, borderColor: '#ff6d00', backgroundColor: 'rgba(255,109,0,0.12)', tension: 0.35, fill: true }] },
    options: { responsive: true, plugins: { legend: { display: true } } }
  });
}

async function refreshDashboard() {
  try {
    const [overview, stats, history, nodesData, heartbeatData, logsData, replicationData] = await Promise.all([
      fetchJSON('/api/overview'),
      fetchJSON('/api/cluster-stats'),
      fetchJSON('/api/metrics-history'),
      fetchJSON('/api/nodes'),
      fetchJSON('/api/heartbeat'),
      fetchJSON('/api/logs'),
      fetchJSON('/api/replication')
    ]);
    renderOverview(overview);
    renderClusterStats(stats);
    renderTrendCharts(history);
    renderNodes(nodesData.nodes || []);
    renderHeartbeat(heartbeatData.rows || []);
    renderActivity(logsData.logs || []);
    renderReplication(replicationData.logs || []);
  } catch (error) {
    console.error(error);
  }
}

async function handleSet(event) {
  event.preventDefault();
  const key = keyInput.value.trim();
  const value = valueInput.value.trim();
  if (!key || !value) return;
  try {
    const res = await fetchJSON('/api/kv/set', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ key, value })
    });
    kvResult.textContent = `${res.message}: ${res.key}=${res.value}`;
    keyInput.value = '';
    valueInput.value = '';
    await refreshDashboard();
  } catch (error) {
    kvResult.textContent = error.message;
  }
}

async function handleClearLogs() {
  await fetchJSON('/api/logs/clear', { method: 'POST' });
  await refreshDashboard();
}

async function failNode(node) {
  await fetchJSON(`/api/demo/fail/${node}`, { method: 'POST' });
  await refreshDashboard();
}

async function recoverNode(node) {
  await fetchJSON(`/api/demo/recover/${node}`, { method: 'POST' });
  await refreshDashboard();
}

async function injectLatency(node) {
  const input = document.getElementById(`latency-${node}`);
  const ms = Number(input.value || 0);
  await fetchJSON(`/api/demo/latency/${node}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ milliseconds: ms })
  });
  await refreshDashboard();
}

async function clearLatency(node) {
  await fetchJSON(`/api/demo/latency-clear/${node}`, { method: 'POST' });
  await refreshDashboard();
}

replicationForm?.addEventListener('submit', handleSet);
clearLogBtn?.addEventListener('click', handleClearLogs);

window.failNode = failNode;
window.recoverNode = recoverNode;
window.injectLatency = injectLatency;
window.clearLatency = clearLatency;

refreshDashboard();
setInterval(refreshDashboard, 3000);
