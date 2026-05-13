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
            // 收到他人消息播放提示音（所有页面生效，不依赖聊天页 onMessage）
            if (msg.from_user_id !== getUserId()) playMessageSound();
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

// 从服务端加载设置并启用 WebSocket（所有页面在初始化时调用）
async function loadAndApplySettings() {
    try {
        const data = await api('GET', '/settings');
        const s = data.settings;
        applyTheme(s.theme || 'light');
        applyFontSize(s.font_size || 'medium');
        window.__soundEnabled = s.sound_enabled !== false;
    } catch (e) {
        applyTheme('light');
        applyFontSize('medium');
        window.__soundEnabled = true;
    }
    // 各页面统一建立 WebSocket，保证任意页面都能收到消息提示音
    connectWS();
}

// ========================================
// 消息提示音（Web Audio API 生成，无需外部文件）
// ========================================

function playMessageSound() {
    if (window.__soundEnabled === false) return;
    try {
        var ctx = new (window.AudioContext || window.webkitAudioContext)();

        // 第一频：800Hz，短促 80ms
        var osc1 = ctx.createOscillator();
        var gain1 = ctx.createGain();
        osc1.type = 'sine';
        osc1.frequency.value = 800;
        gain1.gain.setValueAtTime(0.3, ctx.currentTime);
        gain1.gain.exponentialRampToValueAtTime(0.001, ctx.currentTime + 0.08);
        osc1.connect(gain1);
        gain1.connect(ctx.destination);
        osc1.start(ctx.currentTime);
        osc1.stop(ctx.currentTime + 0.08);

        // 第二频：1200Hz，延迟 40ms 叠加
        var osc2 = ctx.createOscillator();
        var gain2 = ctx.createGain();
        osc2.type = 'sine';
        osc2.frequency.value = 1200;
        gain2.gain.setValueAtTime(0.001, ctx.currentTime);
        gain2.gain.setValueAtTime(0.25, ctx.currentTime + 0.04);
        gain2.gain.exponentialRampToValueAtTime(0.001, ctx.currentTime + 0.15);
        osc2.connect(gain2);
        gain2.connect(ctx.destination);
        osc2.start(ctx.currentTime + 0.04);
        osc2.stop(ctx.currentTime + 0.15);

        // 播放完后关闭 AudioContext
        setTimeout(function() { ctx.close(); }, 200);
    } catch(e) {
        // 浏览器不支持或音频被阻止，静默处理
    }
}
