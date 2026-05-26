// LanLine 聊天增强：DOM 注入 + 输入状态/已读回执/在线状态
// 依赖 app.js（sendWS / sendTyping / sendReadReceipt / onlineUsers）

// 全局：消息已读用户追踪（群聊用）msgId → Set<userId>
var messageReadBy = new Map();

// ========================================
// DOM 注入：为新创建的 Tab 添加在线绿点 + 输入状态指示 + 输入监听
// ========================================
function augmentTab(key, tab) {
    if (tab.type === 'user' && tab.headerEl) {
        var dot = document.createElement('span');
        dot.className = 'online-dot';
        dot.style.cssText = 'display:none;width:8px;height:8px;background:#4caf50;border-radius:50%;margin-right:6px;vertical-align:middle';
        tab.headerEl.insertBefore(dot, tab.headerEl.firstChild);
        tab.onlineDotEl = dot;
        dot.style.display = onlineUsers.has(tab.targetId) ? 'inline-block' : 'none';
    }
    var typingDiv = document.createElement('div');
    typingDiv.className = 'typing-indicator';
    typingDiv.style.cssText = 'display:none;padding:2px 12px;font-size:12px;color:#999;font-style:italic';
    var inputEl = tab.panelEl.querySelector('.chat-input');
    if (inputEl) inputEl.parentNode.insertBefore(typingDiv, inputEl);
    tab.typingEl = typingDiv;
    tab.inputEl.addEventListener('input', function() {
        if (tab.type === 'user') sendTyping(tab.targetId, 0, true);
        else if (tab.type === 'group') sendTyping(0, tab.targetId, true);
    });
}

// ========================================
// 增强 renderOneMessage：本端发出的消息追加已读标记
// 单聊：已读/未读文字
// 群聊：扇形图 + 已读人数
// ========================================
// 由 chat.html 的 renderOneMessage 主动调用，将已读标记注入已生成的 HTML
function injectReadMark(html, msg) {
    var isMe = msg.from_user_id === getUserId();

    var tab = chatTabs.get(activeTabKey);
    var isGroup = tab && tab.type === 'group';

    // 群聊：只在自己发送的消息上显示已读人数
    if (isGroup && isMe) {
        return html.replace('</div>', renderReadPie(msg, tab) + '</div>');
    }
    // 单聊：仅自己发的消息显示已读/未读
    if (!isMe) return html;
    var mark = msg._read
        ? '<span class="read-mark" data-msg-id="' + msg.id + '" style="font-size:10px;color:#999;margin-left:4px">已读</span>'
        : '<span class="read-mark" data-msg-id="' + msg.id + '" style="font-size:10px;color:#ccc;margin-left:4px">未读</span>';
    return html.replace('</div>', mark + '</div>');
}

// renderReadPie 生成扇形图 HTML（仅群聊）
function renderReadPie(msg, tab) {
    var readSet = messageReadBy.get(msg.id);
    var readCount = readSet ? readSet.size : 0;
    var members = groupMembersMap.get(tab ? tab.targetId : 0) || [];
    // 排除消息发送者（发送者自己不能已读自己的消息）
    var total = Number.isInteger(msg.recipient_count) ? msg.recipient_count : (members.length - 1);
    if (total < 0) total = 0;

    var pct = total > 0 ? Math.round(readCount / total * 100) : 0;
    var green = '#4caf50', gray = '#e0e0e0';
    var pieStyle = 'width:14px;height:14px;border-radius:50%;display:inline-block;vertical-align:middle;margin-left:4px;' +
        'background:conic-gradient(' + green + ' 0% ' + pct + '%, ' + gray + ' ' + pct + '% 100%)';

    return '<span class="read-pie" data-msg-id="' + msg.id + '" style="' + pieStyle + '"></span>' +
        '<span class="read-count" style="font-size:10px;color:#666;margin-left:2px">' + readCount + '/' + total + '</span>';
}

// injectReadMark 由 chat.html 的 renderOneMessage 主动调用，不再劫持函数引用

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
// 单聊：标记 _read = true
// 群聊：填充 messageReadBy Set，刷新扇形图
// ========================================
onReadReceipt = function(p) {
    var isGroup = p.group_id > 0;
    var key = isGroup ? ('group:' + p.group_id) : ('user:' + p.from_user_id);
    var tab = chatTabs.get(key);
    if (!tab) return;

    var ids = p.message_ids || [];
    if (isGroup) {
        var lastReadMsgId = p.last_read_msg_id || 0;
        if (lastReadMsgId > 0) {
            for (var i = 0; i < tab.messages.length; i++) {
                var msg = tab.messages[i];
                if (msg.id <= lastReadMsgId && msg.from_user_id !== p.from_user_id) {
                    if (!messageReadBy.has(msg.id)) messageReadBy.set(msg.id, new Set());
                    messageReadBy.get(msg.id).add(p.from_user_id);
                }
            }
        } else {
            // 群聊：兼容旧协议，累计每个消息的已读用户
            for (var i = 0; i < ids.length; i++) {
                var mid = ids[i];
                if (!messageReadBy.has(mid)) messageReadBy.set(mid, new Set());
                messageReadBy.get(mid).add(p.from_user_id);
            }
        }
    } else {
        // 单聊：标记 _read 并持久化到本地 IndexedDB
        var readIds = [];
        for (var i = 0; i < tab.messages.length; i++) {
            if (ids.indexOf(tab.messages[i].id) !== -1) {
                tab.messages[i]._read = true;
                readIds.push(tab.messages[i].id);
            }
        }
        if (readIds.length > 0 && typeof msgStore !== 'undefined') {
            msgStore.markRead(key, readIds);
        }
    }

    if (key === activeTabKey) refreshReadMarks(tab);
};

function refreshReadMarks(tab) {
    if (tab.type === 'group') {
        // 刷新扇形图：删掉旧元素让 renderReadPie 重新生成
        var pies = tab.messagesEl.querySelectorAll('.read-pie');
        pies.forEach(function(pie) {
            var mid = parseInt(pie.dataset.msgId);
            var msg = tab.messages.find(function(m) { return m.id === mid; });
            if (msg) {
                var countEl = pie.nextElementSibling;
                var readSet = messageReadBy.get(mid);
                var readCount = readSet ? readSet.size : 0;
                var members = groupMembersMap.get(tab.targetId) || [];
                var total = Number.isInteger(msg.recipient_count) ? msg.recipient_count : (members.length - 1);
                if (total < 0) total = 0;
                var pct = total > 0 ? Math.round(readCount / total * 100) : 0;
                pie.style.background = 'conic-gradient(#4caf50 0% ' + pct + '%, #e0e0e0 ' + pct + '% 100%)';
                if (countEl && countEl.classList.contains('read-count')) {
                    countEl.textContent = readCount + '/' + total;
                }
            }
        });
    } else {
        var marks = tab.messagesEl.querySelectorAll('.read-mark');
        marks.forEach(function(el) {
            var mid = parseInt(el.dataset.msgId);
            var msg = tab.messages.find(function(m) { return m.id === mid; });
            if (msg && msg._read) el.textContent = '已读';
        });
    }
}

// ========================================
// 在线状态回调 & 辅助
// ========================================
onStatus = function(p) {
    updateOnlineDots();
};

function updateOnlineDots() {
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
    var groupLastReadMsgId = 0;
    for (var i = 0; i < tab.messages.length; i++) {
        var m = tab.messages[i];
        if (m.from_user_id !== getUserId()) {
            if (tab.type === 'group') {
                // 群聊使用已读高水位，只发送最后一条已读消息 ID
                var readSet = messageReadBy.get(m.id);
                if (!readSet || !readSet.has(getUserId())) {
                    if (m.id > groupLastReadMsgId) groupLastReadMsgId = m.id;
                    if (!readSet) { readSet = new Set(); messageReadBy.set(m.id, readSet); }
                    readSet.add(getUserId());
                }
            } else {
                // 单聊：标记已读
                if (!m._read) { unreadIds.push(m.id); m._read = true; }
            }
        }
    }
    if (unreadIds.length > 0) {
        if (tab.type === 'user') {
            sendReadReceipt(tab.targetId, 0, unreadIds);
            // 本端标记持久化到 IndexedDB
            if (typeof msgStore !== 'undefined') msgStore.markRead(key, unreadIds);
        }
    } else if (tab.type === 'group' && groupLastReadMsgId > 0) {
        sendReadReceipt(0, tab.targetId, [], groupLastReadMsgId);
    }
}

// 从服务端恢复群聊已读状态，重建 messageReadBy Map（页面刷新/导航后调用）
async function loadGroupReads(groupId, tab) {
    if (!groupId) return;
    try {
        // 确保成员列表已加载（createTab 中的异步加载可能尚未完成）
        var members = groupMembersMap.get(groupId);
        if (!members || members.length === 0) {
            var mData = await api('GET', '/groups/' + groupId + '/members');
            members = mData.members || [];
            groupMembersMap.set(groupId, members);
        }
        var data = await api('GET', '/groups/' + groupId + '/reads');
        var reads = data.reads || {}; // {user_id: last_read_msg_id}
        // 遍历所有已加载的消息，重建已读用户集合
        for (var i = 0; i < tab.messages.length; i++) {
            var msg = tab.messages[i];
            if (!messageReadBy.has(msg.id)) messageReadBy.set(msg.id, new Set());
            var readSet = messageReadBy.get(msg.id);
            for (var j = 0; j < members.length; j++) {
                var mid = members[j].user_id;
                if (mid === msg.from_user_id) continue; // 发送者不读自己的消息
                if (reads[mid] && msg.id <= reads[mid]) readSet.add(mid);
            }
        }
        refreshReadMarks(tab); // 更新 DOM 中的扇形图
    } catch(e) { console.error('loadGroupReads:', e); }
}
