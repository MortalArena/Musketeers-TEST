package api

// ReadySessionDashboard - Full multi-agent dashboard with model selection, workers, and task execution
const ReadySessionDashboard = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Musketeers AI - Multi-Agent Dashboard</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background: #1e1e1e; color: #d4d4d4; height: 100vh;
            display: flex; flex-direction: column;
        }
        .header {
            background: #252526; padding: 8px 20px;
            border-bottom: 1px solid #3c3c3c;
            display: flex; justify-content: space-between; align-items: center;
        }
        .header h1 { font-size: 15px; font-weight: 600; }
        .header .status { font-size: 11px; color: #858585; }
        .header .status .dot { display: inline-block; width: 8px; height: 8px; border-radius: 50%; margin-right: 4px; }
        .dot.green { background: #4ec9b0; }
        .dot.red { background: #f44747; }
        .main-container { display: flex; flex: 1; overflow: hidden; }
        .sidebar {
            width: 280px; background: #252526;
            border-right: 1px solid #3c3c3c; display: flex; flex-direction: column;
        }
        .agents-panel {
            width: 320px; background: #252526;
            border-left: 1px solid #3c3c3c; display: flex; flex-direction: column;
        }
        .sidebar .tabs { display: flex; border-bottom: 1px solid #3c3c3c; }
        .sidebar .tabs button {
            flex: 1; padding: 7px 4px; background: transparent; border: none;
            color: #858585; font-size: 10px; cursor: pointer; font-weight: 600;
            border-bottom: 2px solid transparent;
        }
        .sidebar .tabs button.active { color: #d4d4d4; border-bottom-color: #007acc; }
        .sidebar .tab-content { flex: 1; overflow-y: auto; padding: 8px; }
        .content { flex: 1; display: flex; flex-direction: column; }
        .chat-header {
            padding: 8px 16px; background: #2d2d2d;
            border-bottom: 1px solid #3c3c3c;
            display: flex; justify-content: space-between; align-items: center; gap: 8px;
        }
        .chat-header .info { font-size: 12px; color: #858585; }
        .chat-header select {
            background: #3c3c3c; color: #d4d4d4; border: 1px solid #3c3c3c;
            padding: 4px 8px; border-radius: 4px; font-size: 11px; max-width: 200px;
        }
        .chat-messages { flex: 1; padding: 12px 16px; overflow-y: auto; }
        .message { margin-bottom: 10px; padding: 8px 12px; border-radius: 6px; max-width: 85%; }
        .message.user { background: #0e639c; margin-left: auto; }
        .message.assistant { background: #3c3c3c; margin-right: auto; }
        .message.system { text-align: center; color: #858585; font-size: 11px; font-style: italic; margin: 8px auto; max-width: 100%; }
        .message .sender { font-size: 10px; font-weight: 600; margin-bottom: 3px; color: #9cdcfe; }
        .message .content { font-size: 13px; line-height: 1.5; white-space: pre-wrap; }
        .message .meta { font-size: 10px; color: #858585; margin-top: 4px; }
        .chat-input { display: flex; padding: 8px 12px; border-top: 1px solid #3c3c3c; gap: 6px; }
        .chat-input input[type="text"] {
            flex: 1; padding: 8px 12px; background: #3c3c3c; border: 1px solid #3c3c3c;
            border-radius: 4px; color: #d4d4d4; font-size: 13px;
        }
        .chat-input input[type="text"]:focus { outline: none; border-color: #007acc; }
        .chat-input button {
            background: #0e639c; color: white; border: none; padding: 8px 16px;
            border-radius: 4px; cursor: pointer; font-size: 12px; font-weight: 600;
        }
        .chat-input button:hover { background: #1177bb; }
        .chat-input button:disabled { background: #3c3c3c; cursor: not-allowed; opacity: 0.5; }
        .card {
            background: #2d2d2d; border: 1px solid #3c3c3c; border-radius: 4px;
            padding: 8px; margin-bottom: 6px; cursor: pointer; transition: all 0.15s;
        }
        .card:hover { border-color: #007acc; }
        .card.active { border-color: #007acc; background: #0e639c33; }
        .card .name { font-size: 12px; font-weight: 600; }
        .card .info { font-size: 10px; color: #858585; margin-top: 2px; }
        .card .tag {
            display: inline-block; font-size: 9px; padding: 1px 5px; border-radius: 3px;
            background: #3c3c3c; margin: 2px 2px 0 0;
        }
        .card .tag.manager { background: #0e639c; color: white; }
        .card .tag.worker { background: #3c3c3c; color: #d4d4d4; }
        .card .tag.online { background: #4ec9b0; color: white; }
        .card .tag.offline { background: #3c3c3c; color: #858585; }
        .card .tag.thinking { background: #ff6b35; color: white; }
        .btn { padding: 5px 10px; border-radius: 3px; border: none; cursor: pointer; font-size: 11px; font-weight: 600; }
        .btn.primary { background: #0e639c; color: white; }
        .btn.primary:hover { background: #1177bb; }
        .btn.danger { background: #c72e2e; color: white; }
        .btn.danger:hover { background: #d73e3e; }
        .btn.sm { padding: 3px 6px; font-size: 10px; }
        .section-title {
            font-size: 10px; font-weight: 600; color: #9cdcfe;
            text-transform: uppercase; letter-spacing: 0.5px; margin: 8px 0 5px;
        }
        .loading { color: #9cdcfe; font-size: 12px; font-style: italic; }
        .empty { color: #858585; font-size: 12px; padding: 12px; text-align: center; }
        .exec-flow { margin: 8px 0; }
        .exec-step {
            background: #2d2d2d; border-left: 3px solid #007acc;
            padding: 6px 10px; margin: 4px 0; border-radius: 0 4px 4px 0; font-size: 12px;
        }
        .exec-step .step-agent { font-size: 10px; font-weight: 600; color: #9cdcfe; }
        .exec-step .step-result { font-size: 11px; color: #d4d4d4; margin-top: 2px; white-space: pre-wrap; }
        .exec-step .step-status { font-size: 10px; color: #4ec9b0; margin-top: 2px; }
        .modal-overlay {
            display: none; position: fixed; top: 0; left: 0; width: 100%; height: 100%;
            background: rgba(0,0,0,0.6); z-index: 100; justify-content: center; align-items: center;
        }
        .modal-overlay.show { display: flex; }
        .modal {
            background: #2d2d2d; border: 1px solid #3c3c3c; border-radius: 8px;
            padding: 20px; width: 450px; max-height: 80vh; overflow-y: auto;
        }
        .modal h2 { font-size: 16px; margin-bottom: 12px; }
        .modal label { display: block; font-size: 12px; color: #858585; margin: 8px 0 3px; }
        .modal input, .modal select {
            width: 100%; padding: 6px 8px; background: #3c3c3c; border: 1px solid #3c3c3c;
            border-radius: 4px; color: #d4d4d4; font-size: 12px;
        }
        .modal input:focus, .modal select:focus { outline: none; border-color: #007acc; }
        .modal .btn-row { display: flex; gap: 8px; margin-top: 12px; justify-content: flex-end; }
        .provider-row {
            display: flex; justify-content: space-between; align-items: center;
            padding: 5px 8px; background: #2d2d2d; border: 1px solid #3c3c3c;
            border-radius: 4px; margin-bottom: 3px; font-size: 11px;
        }
        .provider-row .ok { color: #4ec9b0; }
        .provider-row .err { color: #f44747; }
        .provider-row .unknown { color: #858585; }
        .worker-card {
            background: #2d2d2d; border: 1px solid #3c3c3c; border-radius: 4px;
            padding: 6px 8px; margin-bottom: 4px; font-size: 11px;
        }
        .worker-card .wname { font-weight: 600; }
        .worker-card .winfo { color: #858585; font-size: 10px; }
        .agent-card {
            background: #2d2d2d; border: 1px solid #3c3c3c; border-radius: 4px;
            padding: 8px; margin-bottom: 6px; font-size: 11px;
        }
        .agent-card .agent-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 4px; }
        .agent-card .agent-name { font-weight: 600; color: #d4d4d4; }
        .agent-card .agent-model { font-size: 10px; color: #858585; }
        .agent-card .agent-status { font-size: 9px; padding: 2px 6px; border-radius: 3px; }
        .agent-card .agent-activity { font-size: 10px; color: #858585; margin-top: 4px; }
        .agent-card .agent-messages { max-height: 80px; overflow-y: auto; margin-top: 6px; font-size: 10px; }
        .agent-card .agent-message { background: #252526; padding: 3px 6px; border-radius: 3px; margin-bottom: 3px; }
        .operations-log { background: #1e1e1e; border-radius: 4px; padding: 8px; margin-top: 10px; max-height: 150px; overflow-y: auto; }
        .operations-log .log-entry { font-size: 9px; color: #858585; margin-bottom: 3px; padding: 3px; border-left: 2px solid #3c3c3c; }
        .operations-log .log-entry.success { border-left-color: #4ec9b0; color: #d4d4d4; }
        .operations-log .log-entry.error { border-left-color: #f44747; color: #ff6b6b; }
        .operations-log .log-entry.info { border-left-color: #007acc; color: #9cdcfe; }
    </style>
</head>
<body>
    <div class="header">
        <h1>Musketeers AI</h1>
        <div><span class="status"><span class="dot green"></span><span id="header-status">Operational</span></span></div>
    </div>
    <div class="main-container">
        <!-- Sidebar -->
        <div class="sidebar">
            <div class="tabs">
                <button id="tab-sessions-btn" class="active" onclick="switchTab('sessions')">Sessions</button>
                <button id="tab-providers-btn" onclick="switchTab('providers')">Providers</button>
            </div>
            <div class="tab-content">
                <div id="tab-sessions">
                    <button class="btn primary" style="width:100%;margin-bottom:6px;" onclick="showCreateModal()">+ New Session</button>
                    <div id="sessions-list"></div>
                </div>
                <div id="tab-providers" style="display:none;">
                    <div class="section-title">Providers</div>
                    <div id="providers-list"></div>
                    <div class="section-title" style="margin-top:10px;">Models</div>
                    <div id="models-list"></div>
                </div>
            </div>
        </div>
        <!-- Main Content -->
        <div class="content">
            <div class="chat-header">
                <div>
                    <span id="session-name" style="font-size:13px;font-weight:600;">No session selected</span>
                    <span id="session-info" class="info"></span>
                </div>
                <div style="display:flex;gap:6px;align-items:center;">
                    <span style="font-size:10px;color:#858585;">Model:</span>
                    <select id="model-select" onchange="updateModel()"><option value="">Auto</option></select>
                    <button class="btn primary sm" onclick="showAddWorkerModal()">+ Worker</button>
                </div>
            </div>
            <div class="chat-messages" id="chat-messages">
                <div class="message system">Welcome! Create a session and select a model to start.</div>
            </div>
            <div class="chat-input">
                <input type="text" id="chat-input" placeholder="Type a message or task..." autofocus>
                <button id="send-btn" onclick="sendMessage()">Send</button>
            </div>
        </div>
        <!-- Agents Panel -->
        <div class="agents-panel">
            <div class="tabs">
                <button id="tab-agents-btn" class="active" onclick="switchAgentsTab('agents')">Agents</button>
                <button id="tab-logs-btn" onclick="switchAgentsTab('logs')">Logs</button>
            </div>
            <div class="tab-content">
                <div id="tab-agents">
                    <div class="section-title">Session Agents</div>
                    <div id="agents-list"></div>
                </div>
                <div id="tab-logs" style="display:none;">
                    <div class="section-title">Operations Log</div>
                    <div id="operations-log" class="operations-log"></div>
                </div>
            </div>
        </div>
    </div>

    <!-- Create Session Modal -->
    <div class="modal-overlay" id="create-modal">
        <div class="modal">
            <h2>Create New Session</h2>
            <label>Session Name</label>
            <input type="text" id="cs-name" value="Session" placeholder="Session name">
            <label>Manager Agent (Model)</label>
            <select id="cs-model"><option value="">No manager model</option></select>
            <div class="btn-row">
                <button class="btn" style="background:#3c3c3c;color:white;" onclick="hideCreateModal()">Cancel</button>
                <button class="btn primary" onclick="createSession()">Create</button>
            </div>
        </div>
    </div>

    <!-- Add Worker Modal -->
    <div class="modal-overlay" id="worker-modal">
        <div class="modal">
            <h2>Add Worker Agent</h2>
            <label>Agent ID</label>
            <input type="text" id="aw-id" placeholder="worker-1">
            <label>Provider</label>
            <select id="aw-provider"></select>
            <label>Model</label>
            <select id="aw-model"></select>
            <label>Role</label>
            <select id="aw-role">
                <option value="assistant">Assistant</option>
                <option value="executor">Executor</option>
                <option value="reviewer">Reviewer</option>
                <option value="tester">Tester</option>
            </select>
            <div class="btn-row">
                <button class="btn" style="background:#3c3c3c;color:white;" onclick="hideWorkerModal()">Cancel</button>
                <button class="btn primary" onclick="addWorker()">Add</button>
            </div>
        </div>
    </div>

    <script>
        let sessions = []; let providers = []; let models = []; let allAgents = [];
        let selectedSession = null; let tab = 'sessions';
        let selectedModel = '';

        function switchTab(t) {
            tab = t;
            document.getElementById('tab-sessions-btn').className = t === 'sessions' ? 'active' : '';
            document.getElementById('tab-providers-btn').className = t === 'providers' ? 'active' : '';
            document.getElementById('tab-sessions').style.display = t === 'sessions' ? 'block' : 'none';
            document.getElementById('tab-providers').style.display = t === 'providers' ? 'block' : 'none';
            if (t === 'providers') { loadProviders(); loadModels(); }
        }

        function updateModel() {
            selectedModel = document.getElementById('model-select').value;
        }

        async function api(method, path, body) {
            const opts = { method, headers: { 'Content-Type': 'application/json' } };
            const token = localStorage.getItem('api_token');
            if (token) opts.headers['Authorization'] = 'Bearer ' + token;
            if (body) opts.body = JSON.stringify(body);
            const res = await fetch(path, opts);
            const text = await res.text();
            try { return { ok: res.ok, data: JSON.parse(text) }; }
            catch(e) { return { ok: res.ok, data: text }; }
        }

        // --- Sessions ---
        async function loadSessions() {
            const res = await api('GET', '/api/sessions');
            const d = res.data || {};
            sessions = d.sessions || [];
            allAgents = d.all_agents || [];
            renderSessions();
            if (sessions.length > 0 && !selectedSession) {
                selectSession(sessions[0].id);
            }
            // Auto-load agents into session if none exist
            if (sessions.length === 0 && allAgents.length > 0) {
                autoCreateSessionWithAgents();
            }
        }

        async function autoCreateSessionWithAgents() {
            const modelVal = document.getElementById('cs-model').value;
            let mgrProv = '', mgrModel = '';
            if (modelVal) {
                const parts = modelVal.split('|');
                mgrProv = parts[0] || '';
                mgrModel = parts[1] || '';
            }
            const res = await api('POST', '/api/sessions', {
                name: 'Auto Session',
                owner_did: 'local-user',
                manager_agent_id: 'manager',
                manager_provider: mgrProv,
                manager_model: mgrModel,
                assistant_agents: [],
                worker_agents: []
            });
            if (res.ok) { loadSessions(); }
        }

        // --- Agents Panel Functions ---
        function switchAgentsTab(tabName) {
            document.getElementById('tab-agents-btn').classList.remove('active');
            document.getElementById('tab-logs-btn').classList.remove('active');
            document.getElementById('tab-' + tabName + '-btn').classList.add('active');
            
            document.getElementById('tab-agents').style.display = 'none';
            document.getElementById('tab-logs').style.display = 'none';
            document.getElementById('tab-' + tabName).style.display = 'block';
        }

        function renderAgentsPanel() {
            const el = document.getElementById('agents-list');
            if (!selectedSession) {
                el.innerHTML = '<div class="empty">No session selected</div>';
                return;
            }
            
            const mgr = selectedSession.manager_info || {};
            const ws = selectedSession.workers || [];
            
            let html = '';
            
            // Manager Agent
            if (mgr.model) {
                html += '<div class="agent-card">';
                html += '<div class="agent-header">';
                html += '<span class="agent-name">👑 ' + esc(mgr.model) + '</span>';
                html += '<span class="agent-status online">Online</span>';
                html += '</div>';
                html += '<div class="agent-model">Manager Agent</div>';
                html += '<div class="agent-activity">Ready to coordinate</div>';
                html += '</div>';
            }
            
            // Worker Agents
            ws.forEach(function(w) {
                const status = Math.random() > 0.3 ? 'online' : 'thinking';
                const statusText = status === 'online' ? 'Online' : 'Thinking...';
                html += '<div class="agent-card">';
                html += '<div class="agent-header">';
                html += '<span class="agent-name">🤖 ' + esc(w.model || w.agent_id) + '</span>';
                html += '<span class="agent-status ' + status + '">' + statusText + '</span>';
                html += '</div>';
                html += '<div class="agent-model">Worker Agent</div>';
                html += '<div class="agent-activity">Waiting for tasks</div>';
                html += '</div>';
            });
            
            if (ws.length === 0 && !mgr.model) {
                html += '<div class="empty">No agents in session</div>';
            }
            
            el.innerHTML = html;
        }

        function addLogEntry(type, message) {
            const logEl = document.getElementById('operations-log');
            const time = new Date().toLocaleTimeString();
            const entry = document.createElement('div');
            entry.className = 'log-entry ' + type;
            entry.textContent = '[' + time + '] ' + message;
            logEl.insertBefore(entry, logEl.firstChild);
            
            // Keep only last 50 entries
            while (logEl.children.length > 50) {
                logEl.removeChild(logEl.lastChild);
            }
        }

        function renderSessions() {
            const el = document.getElementById('sessions-list');
            if (!sessions.length) {
                el.innerHTML = '<div class="empty">No sessions.</div>';
                return;
            }
            el.innerHTML = sessions.map(s => {
                const mgr = s.manager_info || {};
                const ws = s.workers || [];
                const totalAgents = (ws.length || 0) + (mgr.model ? 1 : 0);
                const mgrTag = mgr.model ? '<span class="tag manager">👑 ' + esc(mgr.model) + '</span>' : '<span class="tag">No Manager</span>';
                const agentTags = ws.slice(0, 5).map(w =>
                    '<span class="tag">🤖 ' + esc(w.model || w.agent_id) + '</span>'
                ).join('');
                return '<div class="card ' + (selectedSession && selectedSession.id === s.id ? 'active' : '') + '" onclick="selectSession(\'' + s.id + '\')">' +
                    '<div class="name">' + esc(s.name || s.id) + '</div>' +
                    '<div class="info">' + totalAgents + ' agents | ' + (s.status || 'active') + '</div>' +
                    '<div>' + mgrTag + agentTags + '</div>' +
                '</div>';
            }).join('');
        }

        async function createSession() {
            const name = document.getElementById('cs-name').value.trim() || 'Session';
            const modelVal = document.getElementById('cs-model').value;
            let managerProvider = '', managerModel = '';
            if (modelVal) {
                const parts = modelVal.split('|');
                managerProvider = parts[0] || '';
                managerModel = parts[1] || '';
            }
            await api('POST', '/api/sessions', {
                name, owner_did: 'local-user',
                manager_agent_id: 'manager',
                manager_provider: managerProvider,
                manager_model: managerModel,
                assistant_agents: [],
                worker_agents: []
            });
            hideCreateModal();
            loadSessions();
        }

        function selectSession(id) {
            selectedSession = sessions.find(s => s.id === id);
            renderSessions();
            if (!selectedSession) return;
            document.getElementById('session-name').textContent = selectedSession.name || selectedSession.id;
            const mgr = selectedSession.manager_info || {};
            const ws = selectedSession.workers || [];
            const totalAgents = (ws.length || 0) + (mgr.model ? 1 : 0);
            document.getElementById('session-info').textContent = ' | Team: ' + totalAgents + ' agents';
            loadMessages(id);
            renderAgentTeam();
            renderAgentsPanel();
            addLogEntry('info', 'Session selected: ' + selectedSession.name);
        }

        function renderAgentTeam() {
            const el = document.getElementById('chat-messages');
            if (!selectedSession) return;
            
            const mgr = selectedSession.manager_info || {};
            const ws = selectedSession.workers || [];
            
            // Team display as a floating badge area
            let teamHtml = '<div class="message system" style="margin-bottom:6px;">';
            teamHtml += '<strong>👑 Session Manager:</strong> ';
            if (mgr.model) {
                teamHtml += '<span class="tag manager" style="background:#0e639c;color:white;margin:1px;">' +
                    esc(mgr.model) + '</span> ';
            } else {
                teamHtml += '<span class="tag" style="background:#3c3c3c;color:#858585;margin:1px;">No Manager</span> ';
            }
            
            teamHtml += '<br><strong>🤖 Worker Agents (' + ws.length + '):</strong> ';
            if (ws.length > 0) {
                ws.forEach(function(w) {
                    teamHtml += '<span class="tag" style="background:#2d2d2d;color:#d4d4d4;margin:1px;">' +
                        esc(w.model || w.agent_id) + '</span> ';
                });
            } else {
                teamHtml += '<span class="tag" style="background:#3c3c3c;color:#858585;margin:1px;">No Workers</span> ';
            }
            teamHtml += '</div>';
            
            // Insert at the top of messages
            if (document.getElementById('team-banner')) {
                document.getElementById('team-banner').innerHTML = teamHtml;
            }
        }

        // --- Modals ---
        function showCreateModal() {
            const sel = document.getElementById('cs-model');
            sel.innerHTML = '<option value="">No manager model</option>';
            models.forEach(m => {
                sel.innerHTML += '<option value="' + (m.provider || '') + '|' + m.id + '">' + esc(m.name || m.id) + ' (' + esc(m.provider || '') + ')</option>';
            });
            document.getElementById('cs-name').value = 'Session ' + (sessions.length + 1);
            document.getElementById('create-modal').classList.add('show');
        }
        function hideCreateModal() { document.getElementById('create-modal').classList.remove('show'); }

        function showAddWorkerModal() {
            if (!selectedSession) { alert('Select a session first.'); return; }
            const psel = document.getElementById('aw-provider');
            const msel = document.getElementById('aw-model');
            psel.innerHTML = '<option value="">Select provider</option>';
            msel.innerHTML = '<option value="">Select model</option>';
            providers.forEach(p => {
                psel.innerHTML += '<option value="' + (p.type || p.id) + '">' + esc(p.name || p.id) + '</option>';
            });
            models.forEach(m => {
                msel.innerHTML += '<option value="' + m.id + '">' + esc(m.name || m.id) + '</option>';
            });
            document.getElementById('aw-id').value = 'worker-' + Date.now().toString(36);
            document.getElementById('worker-modal').classList.add('show');
        }
        function hideWorkerModal() { document.getElementById('worker-modal').classList.remove('show'); }

        async function addWorker() {
            const agentId = document.getElementById('aw-id').value.trim();
            const provider = document.getElementById('aw-provider').value;
            const model = document.getElementById('aw-model').value;
            const role = document.getElementById('aw-role').value;
            if (!agentId || !provider || !model) { alert('Fill all fields.'); return; }
            const res = await api('POST', '/api/agents', {
                session_id: selectedSession.id,
                agent_did: agentId, name: agentId, role: role,
                metadata: { provider, model }
            });
            hideWorkerModal();
            if (res.ok) {
                loadSessions();
                const msg = 'Worker agent "' + agentId + '" added with ' + provider + '/' + model;
                document.getElementById('chat-messages').innerHTML += '<div class="message system">' + msg + '</div>';
            } else { alert('Failed: ' + JSON.stringify(res.data)); }
        }

        // --- Messages ---
        async function loadMessages(sessionId) {
            const res = await api('GET', '/api/messages/' + sessionId);
            const msgs = Array.isArray(res.data) ? res.data : (res.data && res.data.messages) ? res.data.messages : [];
            renderMessages(msgs);
        }

        function renderMessages(msgs) {
            const el = document.getElementById('chat-messages');
            let html = '<div id="team-banner"></div>';
            if (!msgs || !msgs.length) {
                const mgr = selectedSession ? (selectedSession.manager_info || {}) : {};
                const ws = selectedSession ? (selectedSession.workers || []) : [];
                const totalAgents = (ws.length || 0) + (mgr.model ? 1 : 0);
                html += '<div class="message system">Send a message to start. Session has ' + totalAgents + ' agents (1 manager + ' + ws.length + ' workers).</div>';
                el.innerHTML = html;
                renderAgentTeam();
                return;
            }
            html += msgs.map(m => {
                const role = (m.source === 'user' || m.source === 'You') ? 'user' :
                            (m.source === 'assistant') ? 'assistant' : 'system';
                // Show model/provider in sender name if available
                let sender = esc(m.source || m.role || '');
                if (m.metadata && m.metadata.model) {
                    sender += ' [' + esc(m.metadata.model) + ']';
                } else if (m.meta && m.meta.indexOf('model:') >= 0) {
                    sender += ' ' + esc(m.meta);
                }
                return '<div class="message ' + role + '">' +
                    '<div class="sender">' + sender + '</div>' +
                    '<div class="content">' + esc(m.content || '') + '</div>' +
                    (m.meta ? '<div class="meta">' + esc(m.meta) + '</div>' : '') +
                '</div>';
            }).join('');
            el.innerHTML = html;
            el.scrollTop = el.scrollHeight;
            renderAgentTeam();
        }

        async function sendMessage() {
            const input = document.getElementById('chat-input');
            const text = input.value.trim();
            if (!text) return;
            if (!selectedSession) { alert('Select or create a session first.'); return; }
            
            addLogEntry('info', 'Sending message to session: ' + selectedSession.name);
            
            const btn = document.getElementById('send-btn');
            const msgs = document.getElementById('chat-messages');
            msgs.innerHTML += '<div class="message user"><div class="sender">You</div><div class="content">' + esc(text) + '</div></div>';
            input.value = ''; btn.disabled = true;
            const lid = 'ld-' + Date.now();
            msgs.innerHTML += '<div id="' + lid + '" class="loading">Processing across ' + allAgents.length + ' agents...</div>';
            msgs.scrollTop = msgs.scrollHeight;

            try {
                addLogEntry('info', 'Processing message through manager agent...');
                const modelToUse = selectedModel || (selectedSession.manager_info && selectedSession.manager_info.model) || '';
                const res = await api('POST', '/api/messages/' + selectedSession.id, {
                    content: text, role: 'user', sender: 'user', model: modelToUse
                });
                document.getElementById(lid).remove();
                
                if (res.ok) {
                    addLogEntry('success', 'Message processed successfully');
                } else {
                    addLogEntry('error', 'Request failed: ' + (res.data || 'Unknown error'));
                }
                
                const resp = res.data && res.data.response ? res.data.response : 'No response';
                const mgrModel = selectedSession.manager_info && selectedSession.manager_info.model ? ' via ' + selectedSession.manager_info.model : '';
                msgs.innerHTML += '<div class="message assistant"><div class="sender">Assistant' + esc(mgrModel) + '</div><div class="content">' + esc(resp) + '</div></div>';

                // Show multi-agent responses if available
                if (res.data && res.data.agent_responses) {
                    const agents = res.data.agent_responses;
                    addLogEntry('info', 'Received responses from ' + Object.keys(agents).length + ' agents');
                    for (var aid in agents) {
                        if (agents.hasOwnProperty(aid)) {
                            addLogEntry('info', 'Agent ' + aid + ' responded');
                            msgs.innerHTML += '<div class="message assistant"><div class="sender">' + esc(aid) + '</div><div class="content">' + esc(agents[aid]) + '</div></div>';
                        }
                    }
                }
            } catch(e) {
                addLogEntry('error', 'Network error: ' + e.message);
                document.getElementById(lid).remove();
                msgs.innerHTML += '<div class="message assistant"><div class="sender">Error</div><div class="content">' + esc(e.message) + '</div></div>';
            }
            msgs.scrollTop = msgs.scrollHeight;
            btn.disabled = false;
        }

        // --- Providers & Models ---
        async function loadProviders() {
            const res = await api('GET', '/api/providers');
            providers = Array.isArray(res.data) ? res.data : (res.data && res.data.value) ? res.data.value : [];
            const el = document.getElementById('providers-list');
            if (!providers.length) { el.innerHTML = '<div class="empty">No providers.</div>'; return; }
            el.innerHTML = providers.map(p =>
                '<div class="provider-row">' +
                    '<span>' + esc(p.name || p.id || p.type) + '</span>' +
                    '<span class="' + (p.health || 'unknown') + '">' + (p.status || '?') + '</span>' +
                '</div>'
            ).join('');
        }

        async function loadModels() {
            const res = await api('GET', '/api/models');
            models = Array.isArray(res.data) ? res.data : [];
            const el = document.getElementById('models-list');
            const sel = document.getElementById('model-select');
            if (!models.length) {
                el.innerHTML = '<div class="empty">No models.</div>';
                sel.innerHTML = '<option value="">No models</option>';
                return;
            }
            el.innerHTML = models.map(m =>
                '<div class="card" style="cursor:default;">' +
                    '<div class="name">' + esc(m.name || m.id) + '</div>' +
                    '<div class="info">' + esc(m.provider || '') + ' | ' + (m.max_context || '?') + ' ctx</div>' +
                '</div>'
            ).join('');
            const current = sel.value;
            sel.innerHTML = '<option value="">Auto-select model</option>' +
                models.map(m => '<option value="' + m.id + '" ' + (m.id === current ? 'selected' : '') + '>' + esc(m.name || m.id) + ' (' + esc(m.provider || '') + ')</option>').join('');
        }

        function esc(s) { if (!s) return ''; return String(s).replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;'); }

        // --- Init ---
        loadSessions(); loadProviders(); loadModels();
        document.getElementById('chat-input').addEventListener('keypress', function(e) { if (e.key === 'Enter') sendMessage(); });
        setInterval(() => { loadSessions(); loadProviders(); loadModels(); }, 15000);
    </script>
</body>
</html>`
