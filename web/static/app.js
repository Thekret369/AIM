// AIM 前端工具函数 — 封装 API 请求和 WebSocket

// ========================================
// Token 管理
// ========================================

function getToken() {
    return sessionStorage.getItem('aim_token');
}

function setToken(t) {
    sessionStorage.setItem('aim_token', t);
}

function getUsername() {
    return sessionStorage.getItem('aim_username');
}

function setUsername(u) {
    sessionStorage.setItem('aim_username', u);
}

function getUserId() {
    return parseInt(sessionStorage.getItem('aim_user_id') || '0');
}

function setUserId(id) {
    sessionStorage.setItem('aim_user_id', id);
}

function getUserNickname() {
    return sessionStorage.getItem('aim_nickname') || getUsername();
}

function setUserNickname(n) {
    sessionStorage.setItem('aim_nickname', n);
}

function checkAuth() {
    if (!getToken()) {
        window.location.href = '/login';
        return false;
    }
    return true;
}

async function logout() {
    try { await api('POST', '/logout', {}); } catch(e) {}
    sessionStorage.removeItem('aim_token');
    sessionStorage.removeItem('aim_username');
    sessionStorage.removeItem('aim_user_id');
    sessionStorage.removeItem('aim_nickname');
    window.location.href = '/login';
}

// ========================================
// API 请求封装
// ========================================

async function api(method, path, body) {
    const headers = { 'Content-Type': 'application/json' };
    const token = getToken();
    if (token) headers['Authorization'] = 'Bearer ' + token;

    const opts = { method, headers };
    if (body) opts.body = JSON.stringify(body);

    const resp = await fetch('/api' + path, opts);
    const data = await resp.json();
    if (!resp.ok) throw new Error(data.error || '请求失败');
    return data;
}

// ========================================
// WebSocket 连接
// ========================================

let ws = null;
let onMessage = null;  // 回调: function(msg)

function connectWS() {
    const token = getToken();
    if (!token) return;

    const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const url = proto + '//' + window.location.host + '/ws?token=' + token;

    ws = new WebSocket(url);

    ws.onopen = function() {
        console.log('[ws] 已连接');
    };

    ws.onmessage = function(evt) {
        try {
            const msg = JSON.parse(evt.data);
            if (onMessage) onMessage(msg);
        } catch(e) {
            console.error('[ws] 解析失败:', e);
        }
    };

    ws.onclose = function() {
        console.log('[ws] 已断开，5 秒后重连...');
        setTimeout(connectWS, 5000);
    };

    ws.onerror = function(err) {
        console.error('[ws] 错误:', err);
    };
}

function sendWS(msg) {
    if (ws && ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify(msg));
    }
}

function closeWS() {
    if (ws) ws.close();
}

// ========================================
// 用户设置：主题、字体
// ========================================

function applyTheme(theme) {
    if (theme === 'dark') {
        document.body.classList.add('theme-dark');
    } else {
        document.body.classList.remove('theme-dark');
    }
}

function applyFontSize(size) {
    document.body.classList.remove('font-small', 'font-large');
    if (size === 'small') document.body.classList.add('font-small');
    else if (size === 'large') document.body.classList.add('font-large');
}

// 从服务端加载设置并应用（所有页面在初始化时调用）
async function loadAndApplySettings() {
    try {
        const data = await api('GET', '/settings');
        const s = data.settings;
        applyTheme(s.theme || 'light');
        applyFontSize(s.font_size || 'medium');
        return s;
    } catch (e) {
        // 未配置或请求失败，使用默认值
        applyTheme('light');
        applyFontSize('medium');
        return null;
    }
}
