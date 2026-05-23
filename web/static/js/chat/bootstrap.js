// URL 参数处理
// ========================================
var pendingOpenKey = null, pendingOpenType = null, pendingOpenTarget = null;
var urlParams = new URLSearchParams(window.location.search);
var peerParam = urlParams.get('peer');
var typeParam = urlParams.get('type') || 'user';
if (peerParam) {
    var targetId = parseInt(peerParam);
    pendingOpenType = typeParam === 'group' ? 'group' : 'user';
    pendingOpenTarget = targetId;
    pendingOpenKey = pendingOpenType + ':' + targetId;
}

function escapeHtml(s) {
    var d = document.createElement('div'); d.textContent = s || ''; return d.innerHTML;
}

// ========================================
// 在线状态加载
// ========================================
async function loadOnlineStatus() {
    try {
        var data = await api('GET', '/friends/online');
        (data.friends || []).forEach(function(f) {
            if (f.is_online) onlineUsers.add(f.id);
        });
        updateOnlineDots();
    } catch(e) {}
}

async function syncAfterOnline() {
    if (!chatInitialized) {
        pendingOnlineSync = true;
        return;
    }
    if (onlineSyncRunning) {
        pendingOnlineSync = true;
        return;
    }

    onlineSyncRunning = true;
    pendingOnlineSync = false;
    try {
        await loadLastMessages();
        sortConversations();
        renderSidebar();
        updateOnlineDots();

        var keys = Array.from(chatTabs.keys());
        for (var i = 0; i < keys.length; i++) {
            var tab = chatTabs.get(keys[i]);
            if (!tab) continue;
            tab.hasMore = true;
            await loadHistory(keys[i], 1);
        }
    } finally {
        onlineSyncRunning = false;
        if (pendingOnlineSync) syncAfterOnline();
    }
}

onWSOpen = function() {
    syncAfterOnline();
};

onWSAck = function(payload) {
    var requestId = payload && payload.request_id;
    if (requestId) pendingSends.delete(requestId);
};

onWSError = function(payload) {
    payload = payload || {};
    var requestId = payload.request_id || '';
    var pending = requestId ? pendingSends.get(requestId) : null;
    if (requestId) pendingSends.delete(requestId);

    var message = payload.message || '发送失败';
    if (pending) {
        renderSendFailure(pending.key, pending.message, message);
        return;
    }
    renderSendFailure(activeTabKey, null, message);
};

function registerPendingSend(requestId, key, msg) {
    if (!requestId || !key || !msg) return;
    pendingSends.set(requestId, {
        key: key,
        message: Object.assign({}, msg),
        createdAt: Date.now()
    });
}

function renderSendFailure(key, msg, reason) {
    var tab = key ? chatTabs.get(key) : null;
    var text = msg ? getPreviewText(msg) : '';
    var html = '<div class="send-failure">' +
        '<strong>发送失败</strong>' +
        '<span>' + escapeHtml(reason || '请求处理失败') + '</span>' +
        (text ? '<em>' + escapeHtml(text) + '</em>' : '') +
    '</div>';
    if (tab && tab.messagesEl) {
        tab.messagesEl.innerHTML += html;
        tab.messagesEl.scrollTop = tab.messagesEl.scrollHeight;
        if (msg && msg.type === 'text' && tab.inputEl && !tab.inputEl.value) {
            tab.inputEl.value = msg.content || '';
        }
        return;
    }
    alert('发送失败: ' + (reason || '请求处理失败'));
}

function refreshRecallButtons() {
    chatTabs.forEach(function(tab) {
        if (!tab || !tab.messages || !tab.messagesEl) return;
        var hasVisibleRecallWindow = tab.messages.some(function(msg) {
            if (!msg || msg.from_user_id !== getUserId() || msg.is_recalled) return false;
            var createdAt = new Date(msg.created_at || 0).getTime();
            if (!createdAt) return false;
            return Date.now() - createdAt <= MESSAGE_RECALL_WINDOW_MS + 30000;
        });
        if (hasVisibleRecallWindow) renderAllMessages(tab);
    });
}
setInterval(refreshRecallButtons, 30000);

// ========================================
// 初始化
// ========================================
async function init() {
    ['globalSearchInput', 'globalSearchStart', 'globalSearchEnd'].forEach(function(id) {
        var el = document.getElementById(id);
        if (!el) return;
        el.addEventListener('keydown', function(e) {
            if (e.key === 'Enter') {
                e.preventDefault();
                globalSearchMessages();
            }
        });
    });
    await Promise.all([loadFriends(), loadGroups(), loadAIBots(), loadOnlineStatus()]);
    await loadGroupMetadata();
    buildConversations();
    await loadLastMessages();
    sortConversations();
    renderSidebar();
    updateOnlineDots(); // 侧边栏渲染完成后才能注入绿点
    if (pendingOpenKey) {
        var conv = conversations.find(function(c) { return c.key === pendingOpenKey; });
        var name = conv ? conv.name : ('用户' + pendingOpenTarget);
        openOrSwitchTab(pendingOpenKey, pendingOpenType, pendingOpenTarget, name);
    }
    chatInitialized = true;
    if (pendingOnlineSync) syncAfterOnline();
}

loadAndApplySettings();
init();

