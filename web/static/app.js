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
// WebSocket 连接 — 支持 chat/typing/read_receipt/status 四种消息类型
// ========================================

let ws = null;
let onMessage = null;       // 回调: function(msg)  — chat 消息（已解包）
let onTyping = null;        // 回调: function(payload)
let onReadReceipt = null;   // 回调: function(payload)
let onStatus = null;        // 回调: function(payload)
let onlineUsers = new Set(); // 在线用户 ID 集合

function connectWS() {
    var token = getToken();
    if (!token) return;

    var proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    var url = proto + '//' + window.location.host + '/ws?token=' + token;

    ws = new WebSocket(url);

    ws.onopen = function() {
        console.log('[ws] 已连接');
    };

    ws.onmessage = function(evt) {
        try {
            var raw = JSON.parse(evt.data);
            var type = raw.type || 'chat'; // 兼容旧协议：无 type 视为 chat
            var payload = raw.payload || raw; // 兼容旧协议：无 wrapper 时直接当 message

            switch (type) {
            case 'chat':
                var msg = raw.payload ? payload : raw;
                if (msg.from_user_id !== getUserId()) {
                    if (!window.__suppressSound || !window.__suppressSound(msg)) {
                        playMessageSound();
                    }
                }
                if (onMessage) onMessage(msg);
                break;
            case 'typing':
                if (onTyping) onTyping(payload);
                break;
            case 'read_receipt':
                if (onReadReceipt) onReadReceipt(payload);
                break;
            case 'status':
                if (payload.is_online) {
                    onlineUsers.add(payload.user_id);
                } else {
                    onlineUsers.delete(payload.user_id);
                }
                if (onStatus) onStatus(payload);
                break;
            }
        } catch(e) {
            console.error('[ws] 解析失败:', e);
        }
    };

    ws.onclose = function() {
        console.log('[ws] 已断开，5 秒后重连...');
        onlineUsers.clear();
        setTimeout(connectWS, 5000);
    };

    ws.onerror = function(err) {
        console.error('[ws] 错误:', err);
    };
}

// sendWS 统一封装：sendWS(msg) → chat；sendWS(type, payload) → 指定类型
function sendWS(type, payload) {
    if (!ws || ws.readyState !== WebSocket.OPEN) return;
    if (arguments.length === 1) {
        ws.send(JSON.stringify({ type: 'chat', payload: type }));
    } else {
        ws.send(JSON.stringify({ type: type, payload: payload }));
    }
}

function closeWS() {
    if (ws) ws.close();
}

// ========================================
// 输入状态发送工具（500ms 节流）
// ========================================
var _typingTimers = {};
function sendTyping(peerId, groupId, isTyping) {
    var key = groupId ? ('group:' + groupId) : ('user:' + peerId);
    if (isTyping) {
        if (_typingTimers[key]) return; // 已在节流期间
        _typingTimers[key] = setTimeout(function() {
            delete _typingTimers[key];
            sendWS('typing', { to_user_id: peerId || 0, group_id: groupId || 0, is_typing: false });
        }, 3000); // 3 秒后自动停止
        sendWS('typing', { to_user_id: peerId || 0, group_id: groupId || 0, is_typing: true });
    } else {
        if (_typingTimers[key]) { clearTimeout(_typingTimers[key]); delete _typingTimers[key]; }
        sendWS('typing', { to_user_id: peerId || 0, group_id: groupId || 0, is_typing: false });
    }
}

// ========================================
// 已读回执发送
// ========================================
function sendReadReceipt(peerUserId, groupId, messageIds) {
    if (!messageIds || messageIds.length === 0) return;
    sendWS('read_receipt', {
        peer_user_id: peerUserId || 0,
        group_id: groupId || 0,
        message_ids: messageIds
    });
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
