const navItems = document.querySelectorAll('.nav-item');
const viewSections = document.querySelectorAll('.view-section');

const overviewEls = {
    totalNodes: document.getElementById('totalNodes'),
    nodeOnline: document.getElementById('nodeOnline'),
    nodeOffline: document.getElementById('nodeOffline'),
    totalKeys: document.getElementById('totalKeys'),
    lastUpdated: document.getElementById('lastUpdated'),
    storageMode: document.getElementById('storageMode'),
    lastRefreshLabel: document.getElementById('lastRefreshLabel')
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

function showView(target) {
    navItems.forEach((item) => {
        item.classList.toggle('active', item.dataset.target === target);
    });

    viewSections.forEach((section) => {
        section.classList.toggle('active', section.id === `view-${target}`);
    });
}

navItems.forEach((item) => {
    item.addEventListener('click', () => showView(item.dataset.target));
});

async function fetchJSON(url, options = {}) {
    const response = await fetch(url, options);
    const data = await response.json();
    if (!response.ok) {
        throw new Error(data.error || 'Có lỗi xảy ra');
    }
    return data;
}

function statusClass(status) {
    return status === 'Online' ? 'status-online' : 'status-offline';
}

function renderOverview(data) {
    overviewEls.totalNodes.textContent = data.total_nodes;
    overviewEls.nodeOnline.textContent = data.node_online;
    overviewEls.nodeOffline.textContent = data.node_offline;
    overviewEls.totalKeys.textContent = data.total_keys;
    overviewEls.lastUpdated.textContent = data.last_updated;
    overviewEls.storageMode.textContent = data.storage_mode;
    overviewEls.lastRefreshLabel.textContent = `Refresh lúc ${new Date().toLocaleTimeString('vi-VN')}`;
}

function renderNodes(nodes) {
    if (!nodes.length) {
        nodeCards.innerHTML = '<p class="empty-state">Chưa có node nào.</p>';
        return;
    }

    nodeCards.innerHTML = nodes.map((node) => `
        <article class="node-card">
            <div class="node-top">
                <div>
                    <div class="node-name">${node.name}</div>
                    <div class="role-tag">${node.role.toUpperCase()}</div>
                </div>
                <div class="status-pill ${statusClass(node.status)}">
                    <span class="status-dot"></span>${node.status}
                </div>
            </div>
            <div class="node-meta">
                <div class="meta-box">
                    <span>Port</span>
                    <strong>${node.port}</strong>
                </div>
                <div class="meta-box">
                    <span>Key Count</span>
                    <strong>${node.key_count}</strong>
                </div>
                <div class="meta-box">
                    <span>Memory Usage</span>
                    <strong>${node.memory_usage}</strong>
                </div>
                <div class="meta-box">
                    <span>Last Heartbeat</span>
                    <strong>${new Date(node.last_heartbeat).toLocaleTimeString('vi-VN')}</strong>
                </div>
            </div>
        </article>
    `).join('');
}

function renderHeartbeat(rows) {
    heartbeatTable.innerHTML = rows.map((row) => `
        <tr>
            <td>${row.node}</td>
            <td>${row.last_heartbeat}</td>
            <td>
                <span class="status-pill ${statusClass(row.status)}">
                    <span class="status-dot"></span>${row.status}
                </span>
            </td>
        </tr>
    `).join('');
}

function renderActivity(logs) {
    if (!logs.length) {
        activityLog.innerHTML = '<p class="empty-state">Chưa có activity log.</p>';
        return;
    }

    activityLog.innerHTML = logs.map((log) => `
        <div class="log-item">
            <div class="log-badge ${log.type.toLowerCase()}">${log.type}</div>
            <div>[${log.timestamp}] ${log.message}</div>
        </div>
    `).join('');
}

function renderReplication(logs) {
    if (!logs.length) {
        replicationLog.innerHTML = '<p class="empty-state">Chưa có replication log.</p>';
        return;
    }

    replicationLog.innerHTML = logs.map((log) => `
        <div class="log-item">
            <div class="log-badge replication">${log.target}</div>
            <div>[${log.timestamp}] ${log.message}</div>
        </div>
    `).join('');
}

async function refreshDashboard() {
    try {
        const [overview, nodes, heartbeat, logs, replication] = await Promise.all([
            fetchJSON('/api/overview'),
            fetchJSON('/api/nodes'),
            fetchJSON('/api/heartbeat'),
            fetchJSON('/api/logs?limit=30'),
            fetchJSON('/api/replication?limit=12')
        ]);

        renderOverview(overview);
        renderNodes(nodes.nodes);
        renderHeartbeat(heartbeat.rows);
        renderActivity(logs.logs);
        renderReplication(replication.logs);
    } catch (error) {
        kvResult.textContent = `Không thể tải dữ liệu dashboard: ${error.message}`;
    }
}

async function handleSet(event) {
    event.preventDefault();
    const key = keyInput.value.trim();
    const value = valueInput.value.trim();

    if (!key || !value) {
        kvResult.textContent = 'Vui lòng nhập đầy đủ Key và Value.';
        return;
    }

    try {
        const data = await fetchJSON('/api/kv/set', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ key, value })
        });

        kvResult.textContent = [
            `[${new Date().toLocaleTimeString('vi-VN')}] SET ${data.key}=${data.value}`,
            ...data.replications
        ].join('\n');

        await refreshDashboard();
    } catch (error) {
        kvResult.textContent = error.message;
    }
}

async function handleClearLogs() {
    try {
        const data = await fetchJSON('/api/logs/clear', { method: 'POST' });
        kvResult.textContent = data.message;
        await refreshDashboard();
        showView('activity');
    } catch (error) {
        kvResult.textContent = error.message;
    }
}

replicationForm.addEventListener('submit', handleSet);
clearLogBtn.addEventListener('click', handleClearLogs);

showView('dashboard');
refreshDashboard();
setInterval(refreshDashboard, 3000);
