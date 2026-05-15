// AIM 聊天增强：DOM 注入 + 输入状态/已读回执/在线状态
// 依赖 app.js（sendWS / sendTyping / sendReadReceipt / onlineUsers）

// ========================================
// DOM 注入：为新创建的 Tab 添加在线绿点 + 输入状态指示 + 输入监听
// ========================================
function augmentTab(key, tab) {
    // 在线绿点（仅单聊）
    if (tab.type === 'user' && tab.headerEl) {
        var dot = document.createElement('span');
        dot.className = 'online-dot';
        dot.style.cssText = 'display:none;width:8px;height:8px;background:#4caf50;border-radius:50%;margin-right:6px;vertical-align:middle';
        tab.headerEl.insertBefore(dot, tab.headerEl.firstChild);
        tab.onlineDotEl = dot;
        dot.style.display = onlineUsers.has(tab.targetId) ? 'inline-block' : 'none';
    }
    // 输入状态指示条
    var typingDiv = document.createElement('div');
    typingDiv.className = 'typing-indicator';
    typingDiv.style.cssText = 'display:none;padding:2px 12px;font-size:12px;color:#999;font-style:italic';
    // 插在 chat-input 之前
    var inputEl = tab.panelEl.querySelector('.chat-input');
    if (inputEl) inputEl.parentNode.insertBefore(typingDiv, inputEl);
    tab.typingEl = typingDiv;
    // 输入监听：发 typing 事件
    tab.inputEl.addEventListener('input', function() {
        if (tab.type === 'user') sendTyping(tab.targetId, 0, true);
        else if (tab.type === 'group') sendTyping(0, tab.targetId, true);
    });
}

// ========================================
// 增强 renderOneMessage：追加已读/未读标记
// ========================================
var _origRenderOneMessage = null;
function enhancedRenderOneMessage(msg) {
    var html = _origRenderOneMessage(msg);
    var isMe = msg.from_user_id === getUserId();
    if (isMe) {
        var mark = msg._read
            ? '<span class="read-mark" data-msg-id="' + msg.id + '" style="font-size:10px;color:#999;margin-left:4px">已读</span>'
            : '<span class="read-mark" data-msg-id="' + msg.id + '" style="font-size:10px;color:#ccc;margin-left:4px">未读</span>';
        html = html.replace('</div>', mark + '</div>');
    }
    return html;
}

// 延迟劫持（等 chat.html 内联脚本执行完）
setTimeout(function() {
    if (typeof renderOneMessage === 'function') {
        _origRenderOneMessage = renderOneMessage;
        renderOneMessage = enhancedRenderOneMessage;
    }
}, 100);

// ========================================
// 输入状态回调
// ========================================
onTyping = function(p) {
    var key;
    if (p.group_id > 0) {
        key = 'group:' + p.group_id;
    } else {
        key = 'user:' + p.from_user_id;
    }
    var tab = chatTabs.get(key);
    if (!tab || !tab.typingEl) return;
    if (p.is_typing) {
        tab.typingEl.textContent = (tab.name || '对方') + ' 正在输入...';
        tab.typingEl.style.display = 'block';
        clearTimeout(tab._typingTimeout);
        tab._typingTimeout = setTimeout(function() {
            if (tab.typingEl) tab.typingEl.style.display = 'none';
        }, 4000);
    } else {
        tab.typingEl.style.display = 'none';
    }
};

// ========================================
// 已读回执回调
// ========================================
onReadReceipt = function(p) {
    var key = p.group_id > 0 ? ('group:' + p.group_id) : ('user:' + p.from_user_id);
    var tab = chatTabs.get(key);
    if (!tab) return;
    var ids = p.message_ids || [];
    for (var i = 0; i < tab.messages.length; i++) {
        if (ids.indexOf(tab.messages[i].id) !== -1) tab.messages[i]._read = true;
    }
    if (key === activeTabKey) refreshReadMarks(tab);
};

function refreshReadMarks(tab) {
    var marks = tab.messagesEl.querySelectorAll('.read-mark');
    marks.forEach(function(el) {
        var mid = parseInt(el.dataset.msgId);
        var msg = tab.messages.find(function(m) { return m.id === mid; });
        if (msg && msg._read) el.textContent = '已读';
    });
}

// ========================================
// 在线状态回调 & 辅助
// ========================================
onStatus = function(p) {
    updateOnlineDots();
};

function updateOnlineDots() {
    // 侧边栏在线绿点
    document.querySelectorAll('.sidebar-item[data-type="user"]').forEach(function(el) {
        var uid = parseInt(el.dataset.target);
        var dot = el.querySelector('.online-dot-sidebar');
        if (!dot) {
            dot = document.createElement('span');
            dot.className = 'online-dot-sidebar';
            dot.style.cssText = 'display:inline-block;width:6px;height:6px;background:#4caf50;border-radius:50%;margin-left:4px;vertical-align:middle';
            var infoEl = el.querySelector('.item-name');
            if (infoEl) infoEl.appendChild(dot);
        }
        dot.style.display = onlineUsers.has(uid) ? 'inline-block' : 'none';
    });
    // 聊天头部在线绿点
    chatTabs.forEach(function(tab) {
        if (tab.onlineDotEl && tab.type === 'user') {
            tab.onlineDotEl.style.display = onlineUsers.has(tab.targetId) ? 'inline-block' : 'none';
        }
    });
}

// ========================================
// 发送已读回执
// ========================================
function sendReadReceiptForTab(key) {
    var tab = chatTabs.get(key);
    if (!tab) return;
    var unreadIds = [];
    for (var i = 0; i < tab.messages.length; i++) {
        var m = tab.messages[i];
        if (m.from_user_id !== getUserId() && !m._read) {
            unreadIds.push(m.id);
            m._read = true;
        }
    }
    if (unreadIds.length > 0) {
        if (tab.type === 'user') sendReadReceipt(tab.targetId, 0, unreadIds);
        else if (tab.type === 'group') sendReadReceipt(0, tab.targetId, unreadIds);
    }
}
