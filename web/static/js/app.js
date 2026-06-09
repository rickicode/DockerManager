// =============================================
// DockerManager - Main Application
// =============================================

const API_BASE = '/api';
let currentView = 'dashboard';

// =============================================
// Navigation
// =============================================

document.addEventListener('DOMContentLoaded', () => {
    // Setup navigation
    document.querySelectorAll('.nav-item').forEach(item => {
        item.addEventListener('click', (e) => {
            e.preventDefault();
            const view = item.dataset.view;
            navigateTo(view);
        });
    });

    // Mobile menu toggle
    const menuToggle = document.getElementById('menuToggle');
    const sidebar = document.querySelector('.sidebar');
    
    // Create backdrop element
    const backdrop = document.createElement('div');
    backdrop.className = 'sidebar-backdrop';
    document.body.appendChild(backdrop);
    
    menuToggle.addEventListener('click', () => {
        sidebar.classList.toggle('open');
        backdrop.classList.toggle('visible');
    });
    
    backdrop.addEventListener('click', () => {
        sidebar.classList.remove('open');
        backdrop.classList.remove('visible');
    });
    
    // Close sidebar on nav click (mobile)
    document.querySelectorAll('.nav-item').forEach(item => {
        item.addEventListener('click', () => {
            if (window.innerWidth <= 768) {
                sidebar.classList.remove('open');
                backdrop.classList.remove('visible');
            }
        });
    });

    // Handle hash routing
    window.addEventListener('hashchange', () => {
        const view = window.location.hash.slice(1) || 'dashboard';
        showView(view);
    });

    // Initial load
    const initialView = window.location.hash.slice(1) || 'dashboard';
    showView(initialView);
    loadDashboard();
});

function navigateTo(view) {
    window.location.hash = view;
    showView(view);
}

function showView(view) {
    currentView = view;

    // Update nav
    document.querySelectorAll('.nav-item').forEach(item => {
        item.classList.toggle('active', item.dataset.view === view);
    });

    // Update title
    const titles = {
        dashboard: 'Dashboard',
        containers: 'Containers',
        images: 'Images',
        networks: 'Networks',
        compose: 'Compose',
        tools: 'Tools',
        tailscale: 'Tailscale'
    };
    document.getElementById('pageTitle').textContent = titles[view] || 'Dashboard';

    // Show view
    document.querySelectorAll('.view').forEach(v => v.classList.remove('active'));
    const viewEl = document.getElementById(`view-${view}`);
    if (viewEl) {
        viewEl.classList.add('active');
    }

    // Load data
    switch (view) {
        case 'dashboard':
            loadDashboard();
            break;
        case 'containers':
            loadContainers();
            break;
        case 'images':
            loadImages();
            break;
        case 'networks':
            loadNetworks();
            break;
        case 'tailscale':
            loadTailscaleStatus();
            loadTSDProxyServices();
            loadTSDProxyConfig();
            break;
    }
}

function refreshCurrentView() {
    showView(currentView);
}

// =============================================
// API Helpers
// =============================================

async function apiGet(path) {
    const res = await fetch(`${API_BASE}${path}`);
    if (!res.ok) {
        const err = await res.json();
        throw new Error(err.error || `HTTP ${res.status}`);
    }
    return res.json();
}

async function apiPost(path, data) {
    const res = await fetch(`${API_BASE}${path}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(data),
    });
    if (!res.ok) {
        const err = await res.json();
        throw new Error(err.error || `HTTP ${res.status}`);
    }
    return res.json();
}

async function apiDelete(path) {
    const res = await fetch(`${API_BASE}${path}`, { method: 'DELETE' });
    if (!res.ok) {
        const err = await res.json();
        throw new Error(err.error || `HTTP ${res.status}`);
    }
    return res.json();
}

function showLoader() {
    document.getElementById('loader').style.display = 'flex';
}

function hideLoader() {
    document.getElementById('loader').style.display = 'none';
}

function showToast(message, type = 'success') {
    const toast = document.getElementById('toast');
    toast.textContent = message;
    toast.className = `toast ${type}`;
    toast.classList.add('show');
    setTimeout(() => toast.classList.remove('show'), 3000);
}

function closeModal() {
    document.getElementById('modalOverlay').style.display = 'none';
}

function showModal(title, content) {
    document.getElementById('modalTitle').textContent = title;
    document.getElementById('modalBody').innerHTML = content;
    document.getElementById('modalOverlay').style.display = 'flex';
}

// =============================================
// Dashboard
// =============================================

async function loadDashboard() {
    try {
        const data = await apiGet('/system/info');
        const info = data.info;

        // Update stats
        const statEls = document.querySelectorAll('.stat-card .stat-value');
        const statVals = [
            info.containers || 0,
            info.running || 0,
            info.paused || 0,
            info.stopped || 0,
            info.images || 0,
            info.serverVersion || '-'
        ];
        statEls.forEach((el, i) => {
            if (i === 5) {
                el.textContent = statVals[i];
                el.classList.add('stat-small');
            } else {
                el.textContent = statVals[i];
                el.classList.remove('stat-small');
            }
        });

        // Remove loading class
        document.querySelectorAll('.stat-card').forEach(c => c.classList.remove('loading'));

        // Update system info
        document.getElementById('siOS').textContent = info.os || '-';
        document.getElementById('siArch').textContent = info.architecture || '-';
        document.getElementById('siKernel').textContent = info.kernelVersion || '-';
        document.getElementById('siOSType').textContent = info.osType || '-';
        document.getElementById('siAPIVersion').textContent = info.version || '-';
    } catch (err) {
        showToast('Failed to load dashboard: ' + err.message, 'error');
    }
}

// =============================================
// Containers
// =============================================

async function loadContainers() {
    const tbody = document.getElementById('containersBody');
    const mobileDiv = document.getElementById('containersMobile');
    tbody.innerHTML = '<tr><td colspan="6" class="loading-row">Loading...</td></tr>';
    if (mobileDiv) mobileDiv.innerHTML = '<div class="loading-row">Loading...</div>';

    try {
        const all = document.getElementById('showAllContainers').checked;
        const data = await apiGet(`/containers?all=${all}`);
        const containers = data.containers || [];

        if (containers.length === 0) {
            tbody.innerHTML = '<tr><td colspan="6" class="loading-row">No containers found</td></tr>';
            if (mobileDiv) mobileDiv.innerHTML = '<div class="empty-state">No containers found</div>';
            return;
        }

        // Desktop table
        tbody.innerHTML = containers.map(c => `
            <tr>
                <td><strong>${escapeHtml(c.name)}</strong></td>
                <td>${escapeHtml(c.image)}</td>
                <td><span class="status-badge ${c.state}">${c.state}</span></td>
                <td>${formatPorts(c.ports)}</td>
                <td>${formatDate(c.created)}</td>
                <td>
                    <div class="action-btns">
                        ${c.state === 'running' ? `
                            <button class="btn-icon" onclick="stopContainer('${c.id}')" title="Stop">
                                <svg viewBox="0 0 24 24" fill="currentColor"><rect x="6" y="4" width="4" height="16"/><rect x="14" y="4" width="4" height="16"/></svg>
                            </button>
                            <button class="btn-icon" onclick="restartContainer('${c.id}')" title="Restart">
                                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="23 4 23 10 17 10"/><path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10"/></svg>
                            </button>
                        ` : `
                            <button class="btn-icon success" onclick="startContainer('${c.id}')" title="Start">
                                <svg viewBox="0 0 24 24" fill="currentColor"><polygon points="5 3 19 12 5 21 5 3"/></svg>
                            </button>
                        `}
                        <button class="btn-icon" onclick="viewContainerLogs('${c.id}')" title="Logs">
                            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M4 6h16M4 12h16M4 18h12"/></svg>
                        </button>
                        <button class="btn-icon danger" onclick="removeContainer('${c.id}')" title="Remove">
                            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg>
                        </button>
                    </div>
                </td>
            </tr>
        `).join('');

        // Mobile cards
        if (mobileDiv) {
            mobileDiv.innerHTML = containers.map(c => `
                <div class="container-card">
                    <div class="container-card-header">
                        <div class="container-card-name">${escapeHtml(c.name)}</div>
                        <span class="status-badge ${c.state}">${c.state}</span>
                    </div>
                    <div class="container-card-body">
                        <div class="container-card-row">
                            <span class="container-card-label">Image</span>
                            <span class="container-card-value">${escapeHtml(c.image)}</span>
                        </div>
                        ${c.ports && c.ports.length > 0 ? `
                        <div class="container-card-row">
                            <span class="container-card-label">Ports</span>
                            <span class="container-card-value">${formatPorts(c.ports)}</span>
                        </div>` : ''}
                        <div class="container-card-row">
                            <span class="container-card-label">Created</span>
                            <span class="container-card-value">${formatDate(c.created)}</span>
                        </div>
                    </div>
                    <div class="container-card-actions">
                        ${c.state === 'running' ? `
                            <button class="btn btn-sm btn-secondary" onclick="stopContainer('${c.id}')">Stop</button>
                            <button class="btn btn-sm btn-secondary" onclick="restartContainer('${c.id}')">Restart</button>
                        ` : `
                            <button class="btn btn-sm btn-primary" onclick="startContainer('${c.id}')">Start</button>
                        `}
                        <button class="btn btn-sm btn-ghost" onclick="viewContainerLogs('${c.id}')">Logs</button>
                        <button class="btn btn-sm btn-ghost btn-danger-text" onclick="removeContainer('${c.id}')">Remove</button>
                    </div>
                </div>
            `).join('');
        }
    } catch (err) {
        tbody.innerHTML = `<tr><td colspan="6" class="loading-row">Error: ${escapeHtml(err.message)}</td></tr>`;
        if (mobileDiv) mobileDiv.innerHTML = `<div class="empty-state">Error: ${escapeHtml(err.message)}</div>`;
    }
}

async function startContainer(id) {
    try {
        await apiPost(`/containers/${id}/start`);
        showToast('Container started');
        loadContainers();
    } catch (err) {
        showToast(err.message, 'error');
    }
}

async function stopContainer(id) {
    try {
        await apiPost(`/containers/${id}/stop`);
        showToast('Container stopped');
        loadContainers();
    } catch (err) {
        showToast(err.message, 'error');
    }
}

async function restartContainer(id) {
    try {
        await apiPost(`/containers/${id}/restart`);
        showToast('Container restarted');
        loadContainers();
    } catch (err) {
        showToast(err.message, 'error');
    }
}

async function removeContainer(id) {
    if (!confirm('Are you sure you want to remove this container?')) return;
    try {
        await apiDelete(`/containers/${id}?force=true`);
        showToast('Container removed');
        loadContainers();
    } catch (err) {
        showToast(err.message, 'error');
    }
}

async function viewContainerLogs(id) {
    try {
        const data = await apiGet(`/containers/${id}/logs?tail=50`);
        const logs = data.logs || 'No logs available';
        showModal('Container Logs', `<pre style="background:#0d1117;padding:16px;border-radius:6px;overflow:auto;max-height:400px;font-size:12px;line-height:1.5;color:#e6edf3;font-family:monospace;white-space:pre-wrap">${escapeHtml(logs)}</pre>`);
    } catch (err) {
        showToast(err.message, 'error');
    }
}

// =============================================
// Create Container Modal
// =============================================

function showCreateContainerModal() {
    const content = `
        <div class="form-grid">
            <div class="full-width">
                <div class="form-group">
                    <label>Container Name</label>
                    <input type="text" id="createName" class="input" placeholder="my-container">
                </div>
            </div>
            <div class="full-width">
                <div class="form-group">
                    <label>Image</label>
                    <input type="text" id="createImage" class="input" placeholder="nginx:latest">
                </div>
            </div>
            <div class="full-width">
                <div class="form-group">
                    <label>Command (optional)</label>
                    <input type="text" id="createCommand" class="input" placeholder="e.g., nginx -g 'daemon off;'">
                </div>
            </div>
            <div class="full-width">
                <div class="form-group">
                    <label>Network Mode</label>
                    <select id="createNetworkMode" class="select">
                        <option value="">Default (bridge)</option>
                        <option value="bridge">Bridge</option>
                        <option value="host">Host</option>
                        <option value="none">None</option>
                    </select>
                </div>
            </div>
            <div class="full-width">
                <div class="form-group">
                    <label>Restart Policy</label>
                    <select id="createRestartPolicy" class="select">
                        <option value="no">No</option>
                        <option value="always">Always</option>
                        <option value="unless-stopped">Unless Stopped</option>
                        <option value="on-failure">On Failure</option>
                    </select>
                </div>
            </div>
            <div class="full-width">
                <label>Port Mappings</label>
                <div id="portEntries">
                    <div class="entry-row">
                        <input type="number" class="input" placeholder="Host Port" style="width:120px">
                        <input type="number" class="input" placeholder="Container Port" style="width:120px">
                        <select class="select" style="width:100px">
                            <option value="tcp">TCP</option>
                            <option value="udp">UDP</option>
                        </select>
                        <button class="btn-remove" onclick="this.parentElement.remove()">&times;</button>
                    </div>
                </div>
                <button class="btn-add" onclick="addPortEntry()">+ Add Port</button>
            </div>
            <div class="full-width">
                <label>Environment Variables</label>
                <div id="envEntries">
                    <div class="entry-row">
                        <input type="text" class="input" placeholder="KEY" style="width:200px">
                        <input type="text" class="input" placeholder="value" style="flex:1">
                        <button class="btn-remove" onclick="this.parentElement.remove()">&times;</button>
                    </div>
                </div>
                <button class="btn-add" onclick="addEnvEntry()">+ Add Variable</button>
            </div>
            <div class="full-width">
                <label>Volume Mounts</label>
                <div id="volEntries">
                    <div class="entry-row">
                        <input type="text" class="input" placeholder="/host/path" style="flex:1">
                        <input type="text" class="input" placeholder="/container/path" style="flex:1">
                        <button class="btn-remove" onclick="this.parentElement.remove()">&times;</button>
                    </div>
                </div>
                <button class="btn-add" onclick="addVolumeEntry()">+ Add Volume</button>
            </div>
        </div>
        <div class="form-actions" style="margin-top:20px">
            <button class="btn btn-primary" onclick="createContainer()">Create Container</button>
            <button class="btn btn-secondary" onclick="closeModal()">Cancel</button>
        </div>
    `;
    showModal('Create Container', content);
}

function addPortEntry() {
    const div = document.createElement('div');
    div.className = 'entry-row';
    div.innerHTML = `
        <input type="number" class="input" placeholder="Host Port" style="width:120px">
        <input type="number" class="input" placeholder="Container Port" style="width:120px">
        <select class="select" style="width:100px">
            <option value="tcp">TCP</option>
            <option value="udp">UDP</option>
        </select>
        <button class="btn-remove" onclick="this.parentElement.remove()">&times;</button>
    `;
    document.getElementById('portEntries').appendChild(div);
}

function addEnvEntry() {
    const div = document.createElement('div');
    div.className = 'entry-row';
    div.innerHTML = `
        <input type="text" class="input" placeholder="KEY" style="width:200px">
        <input type="text" class="input" placeholder="value" style="flex:1">
        <button class="btn-remove" onclick="this.parentElement.remove()">&times;</button>
    `;
    document.getElementById('envEntries').appendChild(div);
}

function addVolumeEntry() {
    const div = document.createElement('div');
    div.className = 'entry-row';
    div.innerHTML = `
        <input type="text" class="input" placeholder="/host/path" style="flex:1">
        <input type="text" class="input" placeholder="/container/path" style="flex:1">
        <button class="btn-remove" onclick="this.parentElement.remove()">&times;</button>
    `;
    document.getElementById('volEntries').appendChild(div);
}

async function createContainer() {
    try {
        const name = document.getElementById('createName').value.trim();
        const image = document.getElementById('createImage').value.trim();
        const command = document.getElementById('createCommand').value.trim();
        const networkMode = document.getElementById('createNetworkMode').value;
        const restartPolicy = document.getElementById('createRestartPolicy').value;

        if (!name || !image) {
            showToast('Name and Image are required', 'error');
            return;
        }

        // Parse ports
        const portRows = document.querySelectorAll('#portEntries .entry-row');
        const ports = [];
        portRows.forEach(row => {
            const inputs = row.querySelectorAll('input');
            const select = row.querySelector('select');
            const hostPort = parseInt(inputs[0].value);
            const containerPort = parseInt(inputs[1].value);
            if (containerPort) {
                ports.push({
                    hostPort: hostPort || 0,
                    containerPort: containerPort,
                    protocol: select ? select.value : 'tcp'
                });
            }
        });

        // Parse env
        const envRows = document.querySelectorAll('#envEntries .entry-row');
        const env = {};
        envRows.forEach(row => {
            const inputs = row.querySelectorAll('input');
            const key = inputs[0].value.trim();
            const val = inputs[1].value.trim();
            if (key) env[key] = val;
        });

        // Parse volumes
        const volRows = document.querySelectorAll('#volEntries .entry-row');
        const volumes = [];
        volRows.forEach(row => {
            const inputs = row.querySelectorAll('input');
            const hostPath = inputs[0].value.trim();
            const containerPath = inputs[1].value.trim();
            if (hostPath && containerPath) {
                volumes.push({ hostPath, containerPath, readOnly: false });
            }
        });

        showLoader();
        await apiPost('/containers', {
            name,
            image,
            command: command ? splitCommandString(command) : [],
            env,
            ports,
            volumes,
            networkMode,
            restartPolicy
        });

        showToast('Container created successfully');
        closeModal();
        loadContainers();
    } catch (err) {
        showToast(err.message, 'error');
    } finally {
        hideLoader();
    }
}

// =============================================
// Images
// =============================================

async function loadImages() {
    const tbody = document.getElementById('imagesBody');
    tbody.innerHTML = '<tr><td colspan="6" class="loading-row">Loading...</td></tr>';

    try {
        const data = await apiGet('/images');
        const images = data.images || [];

        if (images.length === 0) {
            tbody.innerHTML = '<tr><td colspan="6" class="loading-row">No images found</td></tr>';
            return;
        }

        tbody.innerHTML = images.map(img => {
            const repoTag = img.repoTags && img.repoTags.length > 0 ? img.repoTags[0] : '<none>:<none>';
            const [repo, tag] = repoTag.split(':');
            return `
                <tr>
                    <td><code>${escapeHtml(img.id)}</code></td>
                    <td>${escapeHtml(repo)}</td>
                    <td>${escapeHtml(tag || 'latest')}</td>
                    <td>${formatSize(img.size)}</td>
                    <td>${formatDate(img.created)}</td>
                    <td>
                        <div class="action-btns">
                            <button class="btn-icon danger" onclick="removeImage('${escapeHtml(img.id)}')" title="Remove">
                                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg>
                            </button>
                        </div>
                    </td>
                </tr>
            `;
        }).join('');
    } catch (err) {
        tbody.innerHTML = `<tr><td colspan="6" class="loading-row">Error: ${escapeHtml(err.message)}</td></tr>`;
    }
}

function showPullImageModal() {
    const content = `
        <div class="form-group">
            <label>Image Name</label>
            <input type="text" id="pullImageName" class="input" placeholder="nginx:latest" value="nginx:latest">
        </div>
        <div class="form-actions">
            <button class="btn btn-primary" onclick="pullImage()">Pull Image</button>
            <button class="btn btn-secondary" onclick="closeModal()">Cancel</button>
        </div>
    `;
    showModal('Pull Image', content);
    setTimeout(() => document.getElementById('pullImageName')?.focus(), 100);
}

async function pullImage() {
    const imageName = document.getElementById('pullImageName').value.trim();
    if (!imageName) {
        showToast('Image name is required', 'error');
        return;
    }

    try {
        showLoader();
        await apiPost('/images/pull', { image: imageName });
        showToast('Image pulled successfully');
        closeModal();
        loadImages();
    } catch (err) {
        showToast(err.message, 'error');
    } finally {
        hideLoader();
    }
}

async function removeImage(id) {
    if (!confirm('Are you sure you want to remove this image?')) return;
    try {
        await apiDelete(`/images/${id}?force=true`);
        showToast('Image removed');
        loadImages();
    } catch (err) {
        showToast(err.message, 'error');
    }
}

// =============================================
// Networks
// =============================================

async function loadNetworks() {
    const tbody = document.getElementById('networksBody');
    tbody.innerHTML = '<tr><td colspan="7" class="loading-row">Loading...</td></tr>';

    try {
        const data = await apiGet('/networks');
        const networks = data.networks || [];

        if (networks.length === 0) {
            tbody.innerHTML = '<tr><td colspan="7" class="loading-row">No networks found</td></tr>';
            return;
        }

        tbody.innerHTML = networks.map(n => `
            <tr>
                <td><strong>${escapeHtml(n.name)}</strong></td>
                <td>${escapeHtml(n.driver)}</td>
                <td>${escapeHtml(n.scope)}</td>
                <td>${escapeHtml(n.subnet) || '-'}</td>
                <td>${escapeHtml(n.gateway) || '-'}</td>
                <td>${n.containers}</td>
                <td>
                    <div class="action-btns">
                        ${n.name !== 'bridge' && n.name !== 'host' && n.name !== 'none' ? `
                            <button class="btn-icon danger" onclick="removeNetwork('${escapeHtml(n.id)}')" title="Remove">
                                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg>
                            </button>
                        ` : '<span style="color:var(--text-muted);font-size:12px">default</span>'}
                    </div>
                </td>
            </tr>
        `).join('');
    } catch (err) {
        tbody.innerHTML = `<tr><td colspan="7" class="loading-row">Error: ${escapeHtml(err.message)}</td></tr>`;
    }
}

function showCreateNetworkModal() {
    const content = `
        <div class="form-grid">
            <div class="full-width">
                <div class="form-group">
                    <label>Network Name</label>
                    <input type="text" id="createNetName" class="input" placeholder="my-network">
                </div>
            </div>
            <div>
                <div class="form-group">
                    <label>Driver</label>
                    <select id="createNetDriver" class="select">
                        <option value="bridge">Bridge</option>
                        <option value="host">Host</option>
                        <option value="overlay">Overlay</option>
                        <option value="macvlan">Macvlan</option>
                    </select>
                </div>
            </div>
            <div>
                <div class="form-group">
                    <label>Subnet (optional)</label>
                    <input type="text" id="createNetSubnet" class="input" placeholder="172.20.0.0/16">
                </div>
            </div>
            <div class="full-width">
                <div class="form-group">
                    <label>Gateway (optional)</label>
                    <input type="text" id="createNetGateway" class="input" placeholder="172.20.0.1">
                </div>
            </div>
        </div>
        <div class="form-actions" style="margin-top:20px">
            <button class="btn btn-primary" onclick="createNetwork()">Create Network</button>
            <button class="btn btn-secondary" onclick="closeModal()">Cancel</button>
        </div>
    `;
    showModal('Create Network', content);
}

async function createNetwork() {
    try {
        const name = document.getElementById('createNetName').value.trim();
        const driver = document.getElementById('createNetDriver').value;
        const subnet = document.getElementById('createNetSubnet').value.trim();
        const gateway = document.getElementById('createNetGateway').value.trim();

        if (!name) {
            showToast('Network name is required', 'error');
            return;
        }

        showLoader();
        await apiPost('/networks', { name, driver, subnet, gateway });
        showToast('Network created');
        closeModal();
        loadNetworks();
    } catch (err) {
        showToast(err.message, 'error');
    } finally {
        hideLoader();
    }
}

async function removeNetwork(id) {
    if (!confirm('Are you sure you want to remove this network?')) return;
    try {
        await apiDelete(`/networks/${id}`);
        showToast('Network removed');
        loadNetworks();
    } catch (err) {
        showToast(err.message, 'error');
    }
}

// =============================================
// Compose
// =============================================

async function deployCompose() {
    const content = document.getElementById('composeYaml').value.trim();
    const projectName = document.getElementById('composeProjectName').value.trim();

    if (!content) {
        showToast('Please enter a docker-compose.yml content', 'error');
        return;
    }

    try {
        showLoader();
        const data = await apiPost('/compose/deploy', {
            content,
            projectName: projectName || undefined
        });

        const resultBox = document.getElementById('composeResult');
        resultBox.style.display = 'block';
        resultBox.className = 'result-box success';
        resultBox.innerHTML = `
            <strong>Deployed successfully!</strong><br>
            Containers created: ${(data.containers || []).join(', ') || 'None'}
        `;
        showToast('Compose deployment started');
    } catch (err) {
        const resultBox = document.getElementById('composeResult');
        resultBox.style.display = 'block';
        resultBox.className = 'result-box error';
        resultBox.innerHTML = `<strong>Error:</strong> ${escapeHtml(err.message)}`;
        showToast(err.message, 'error');
    } finally {
        hideLoader();
    }
}

async function parseCompose() {
    const content = document.getElementById('composeYaml').value.trim();
    if (!content) {
        showToast('Please enter a docker-compose.yml content', 'error');
        return;
    }

    try {
        const data = await apiPost('/compose/parse', { content });
        const services = data.services || [];

        const resultBox = document.getElementById('composeResult');
        resultBox.style.display = 'block';
        resultBox.className = 'result-box info';

        let html = '<strong>Compose File Valid!</strong><br><br>';
        html += `<strong>Version:</strong> ${data.version || '3'}<br>`;
        html += '<strong>Services:</strong><ul>';
        services.forEach(s => {
            html += `<li><strong>${escapeHtml(s.name)}</strong> (${escapeHtml(s.image)})`;
            if (s.ports && s.ports.length > 0) {
                html += `<br>Ports: ${s.ports.join(', ')}`;
            }
            if (s.environment && Object.keys(s.environment).length > 0) {
                const envCount = Object.keys(s.environment).length;
                html += `<br>Environment: ${envCount} variable(s)`;
            }
            if (s.volumes && s.volumes.length > 0) {
                html += `<br>Volumes: ${s.volumes.join(', ')}`;
            }
            html += '</li>';
        });
        html += '</ul>';

        resultBox.innerHTML = html;
    } catch (err) {
        const resultBox = document.getElementById('composeResult');
        resultBox.style.display = 'block';
        resultBox.className = 'result-box error';
        resultBox.innerHTML = `<strong>Invalid Compose File:</strong> ${escapeHtml(err.message)}`;
    }
}

// =============================================
// Tailscale / TSDProxy
// =============================================

async function loadTailscaleStatus() {
    try {
        const data = await apiGet('/tailscale/status');
        const status = data.status || {};

        document.getElementById('tsdStatus').textContent = status.running ? 'Running' : 'Stopped';
        document.getElementById('tsdStatus').style.color = status.running ? 'var(--success)' : 'var(--danger)';
        document.getElementById('tsdContainerId').textContent = status.containerId || '-';
        document.getElementById('tsdPort').textContent = status.port || '-';
    } catch (err) {
        showToast('Failed to load TSDProxy status: ' + err.message, 'error');
    }
}

async function loadTSDProxyServices() {
    const tbody = document.getElementById('tsdServicesBody');
    tbody.innerHTML = '<tr><td colspan="6" class="loading-row">Loading...</td></tr>';

    try {
        const data = await apiGet('/tailscale/services');
        const services = data.services || [];

        if (services.length === 0) {
            tbody.innerHTML = '<tr><td colspan="6" class="loading-row">No proxied services found. Add labels tsdproxy.enable=true to your containers.</td></tr>';
            return;
        }

        tbody.innerHTML = services.map(s => `
            <tr>
                <td><strong>${escapeHtml(s.containerName)}</strong></td>
                <td>${escapeHtml(s.image)}</td>
                <td><code>${escapeHtml(s.hostname)}.ts.net</code></td>
                <td>${s.funnel ? '<span class="status-badge running">Enabled</span>' : '<span class="status-badge created">Disabled</span>'}</td>
                <td><span class="status-badge ${s.state}">${s.state}</span></td>
                <td>
                    <div class="action-btns">
                        <button class="btn-icon" onclick="viewContainerLogs('${s.containerId}')" title="Logs">
                            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M4 6h16M4 12h16M4 18h12"/></svg>
                        </button>
                    </div>
                </td>
            </tr>
        `).join('');
    } catch (err) {
        tbody.innerHTML = `<tr><td colspan="6" class="loading-row">Error: ${escapeHtml(err.message)}</td></tr>`;
    }
}

async function loadTSDProxyConfig() {
    try {
        const data = await apiGet('/tailscale/config');
        const cfg = data.config || {};
        document.getElementById('tsdClientId').value = cfg.clientId || '';
        document.getElementById('tsdClientSecret').value = cfg.clientSecret || '';
        document.getElementById('tsdTags').value = cfg.tags || 'tag:tsdproxy';
        document.getElementById('tsdHostname').value = cfg.hostname || 'tsdproxy';
        document.getElementById('tsdDashPort').value = cfg.dashboardPort || 8080;
    } catch (err) {
        // Config not set yet, use defaults
    }
}

function showDeployTSDProxyModal() {
    const content = `
        <div class="form-grid">
            <div class="full-width">
                <div class="form-group">
                    <label>Tailscale OAuth Client ID</label>
                    <input type="text" id="deployClientId" class="input" placeholder="tsclient_...">
                </div>
            </div>
            <div class="full-width">
                <div class="form-group">
                    <label>Tailscale OAuth Client Secret</label>
                    <input type="password" id="deployClientSecret" class="input" placeholder="...">
                </div>
            </div>
            <div class="form-group flex-1">
                <label>Tags</label>
                <input type="text" id="deployTags" class="input" value="tag:tsdproxy" placeholder="tag:tsdproxy">
            </div>
            <div class="form-group flex-1">
                <label>Hostname</label>
                <input type="text" id="deployHostname" class="input" value="tsdproxy" placeholder="tsdproxy">
            </div>
            <div class="form-group flex-1">
                <label>Dashboard Port</label>
                <input type="number" id="deployDashPort" class="input" value="8080" min="1" max="65535">
            </div>
        </div>
        <div class="form-actions" style="margin-top:20px">
            <button class="btn btn-primary" onclick="deployTSDProxy()">Deploy</button>
            <button class="btn btn-secondary" onclick="closeModal()">Cancel</button>
        </div>
    `;
    showModal('Deploy TSDProxy', content);
}

async function deployTSDProxy() {
    const clientId = document.getElementById('deployClientId').value.trim();
    const clientSecret = document.getElementById('deployClientSecret').value.trim();
    const tags = document.getElementById('deployTags').value.trim();
    const hostname = document.getElementById('deployHostname').value.trim();
    const dashboardPort = parseInt(document.getElementById('deployDashPort').value);

    if (!clientId || !clientSecret) {
        showToast('Client ID and Secret are required', 'error');
        return;
    }

    try {
        showLoader();
        await apiPost('/tailscale/deploy', {
            clientId,
            clientSecret,
            tags,
            hostname,
            dashboardPort
        });
        showToast('TSDProxy deployed successfully');
        closeModal();
        loadTailscaleStatus();
    } catch (err) {
        showToast(err.message, 'error');
    } finally {
        hideLoader();
    }
}

async function tsdproxyAction(action) {
    try {
        showLoader();
        await apiPost(`/tailscale/${action}`);
        showToast(`TSDProxy ${action}ed`);
        loadTailscaleStatus();
    } catch (err) {
        showToast(err.message, 'error');
    } finally {
        hideLoader();
    }
}

async function tsdproxyRemove() {
    if (!confirm('Are you sure you want to remove TSDProxy? This will delete the container.')) return;
    try {
        showLoader();
        await apiDelete('/tailscale/remove');
        showToast('TSDProxy removed');
        loadTailscaleStatus();
    } catch (err) {
        showToast(err.message, 'error');
    } finally {
        hideLoader();
    }
}

async function saveTSDProxyConfig() {
    const clientId = document.getElementById('tsdClientId').value.trim();
    const clientSecret = document.getElementById('tsdClientSecret').value.trim();
    const tags = document.getElementById('tsdTags').value.trim();
    const hostname = document.getElementById('tsdHostname').value.trim();
    const dashboardPort = parseInt(document.getElementById('tsdDashPort').value);

    try {
        showLoader();
        await apiPost('/tailscale/config', {
            clientId,
            clientSecret,
            tags,
            hostname,
            dashboardPort
        });
        showToast('Config saved');
    } catch (err) {
        showToast(err.message, 'error');
    } finally {
        hideLoader();
    }
}

// =============================================
// Tools - Port Checker
// =============================================

async function checkPort() {
    const host = document.getElementById('portCheckHost').value.trim();
    const port = parseInt(document.getElementById('portCheckPort').value);

    if (!host || !port) {
        showToast('Host and Port are required', 'error');
        return;
    }

    try {
        showLoader();
        const data = await apiPost('/port-check', { host, port });
        const result = data.result;

        const resultBox = document.getElementById('portCheckResult');
        resultBox.style.display = 'block';
        resultBox.className = `result-box ${result.open ? 'success' : 'error'}`;

        resultBox.innerHTML = `
            <div class="port-status ${result.open ? 'open' : 'closed'}">
                <span>${result.open ? '✓' : '✗'}</span>
                <span>Port ${port} on ${host} is <strong>${result.open ? 'OPEN' : 'CLOSED'}</strong></span>
            </div>
            ${result.service ? `<div style="margin-top:8px">Service: ${escapeHtml(result.service)}</div>` : ''}
        `;
    } catch (err) {
        const resultBox = document.getElementById('portCheckResult');
        resultBox.style.display = 'block';
        resultBox.className = 'result-box error';
        resultBox.innerHTML = `<strong>Error:</strong> ${escapeHtml(err.message)}`;
    } finally {
        hideLoader();
    }
}

// =============================================
// Utility Functions
// =============================================

function splitCommandString(cmd) {
    if (!cmd) return [];
    const result = [];
    let current = '';
    let inQuote = false;
    let quoteChar = '';
    for (let i = 0; i < cmd.length; i++) {
        const ch = cmd[i];
        if ((ch === '"' || ch === "'") && !inQuote) {
            inQuote = true;
            quoteChar = ch;
            continue;
        }
        if (ch === quoteChar && inQuote) {
            inQuote = false;
            quoteChar = '';
            continue;
        }
        if (ch === ' ' && !inQuote) {
            if (current !== '') {
                result.push(current);
                current = '';
            }
            continue;
        }
        current += ch;
    }
    if (current !== '') {
        result.push(current);
    }
    return result;
}

function escapeHtml(str) {
    if (!str) return '';
    const div = document.createElement('div');
    div.textContent = String(str);
    return div.innerHTML;
}

function formatPorts(ports) {
    if (!ports || ports.length === 0) return '-';
    return ports.map(p => {
        if (p.hostPort) {
            return `${p.hostPort}:${p.containerPort}/${p.protocol || 'tcp'}`;
        }
        return `${p.containerPort}/${p.protocol || 'tcp'}`;
    }).join(', ');
}

function formatDate(dateStr) {
    if (!dateStr) return '-';
    try {
        const d = new Date(dateStr);
        const now = new Date();
        const diff = now - d;
        const days = Math.floor(diff / (1000*60*60*24));
        if (days === 0) return 'Today';
        if (days === 1) return 'Yesterday';
        if (days < 7) return `${days} days ago`;
        return d.toLocaleDateString();
    } catch {
        return dateStr;
    }
}

function formatSize(bytes) {
    if (!bytes) return '0 B';
    const units = ['B', 'KB', 'MB', 'GB', 'TB'];
    let i = 0;
    let size = bytes;
    while (size >= 1024 && i < units.length - 1) {
        size /= 1024;
        i++;
    }
    return `${size.toFixed(1)} ${units[i]}`;
}
