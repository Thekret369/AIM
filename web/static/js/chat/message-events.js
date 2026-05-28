// 消息提示音控制（免打扰 + @ 提醒）
// ========================================
// 返回 true 表示应该抑制提示音
window.__suppressSound = function(msg) {
    if (!msg.group_id) return false; // 非群消息不抑制
    var groupId = msg.group_id;
    // 免打扰检查
    if (!groupDNDMap.get(groupId)) return false; // 未开启免打扰，正常播放
    // 开启免打扰，检查是否被 @
    if (!msg.mentions) return true; // 无 @ 信息，抑制
    var myId = getUserId();
    try {
        var ids = JSON.parse(msg.mentions);
        if (!Array.isArray(ids) || ids.indexOf(myId) === -1) return true; // 未被 @，抑制
    } catch(e) { return true; }
    return false; // 被 @ 了，不抑制
};

// ========================================
// 消息接收路由
// ========================================
onMessage = function(msg) {
    var myId = getUserId();
    var key;
    if (msg.group_id !== null && msg.group_id !== undefined && msg.group_id !== 0) {
        key = 'group:' + msg.group_id;
    } else if (msg.to_user_id !== null && msg.to_user_id !== undefined && msg.to_user_id !== 0) {
        var peerId = (msg.from_user_id === myId) ? msg.to_user_id : msg.from_user_id;
        key = 'user:' + peerId;
    } else {
        key = 'broadcast';
    }

    ensureConversationForMessage(key, msg);
    var updateLatest = shouldUseAsConversationLatest(key, msg);
    if (updateLatest) {
        lastMessages.set(key, msg);
        lastTimestamps.set(key, new Date(msg.created_at || Date.now()));
        updateSidebarPreview(key, msg);
    }

    // 新会话补充进 conversations
    if (!conversations.find(function(c) { return c.key === key; })) {
        if (key.startsWith('group:')) {
            var gid = parseInt(key.split(':')[1]);
            var g = sidebarGroups.find(function(g) { return g.id === gid; });
            if (g) conversations.push({ key: key, type: 'group', targetId: gid, name: g.name, avatarChar: g.name.charAt(0) });
        } else if (key.startsWith('user:')) {
            var uid = parseInt(key.split(':')[1]);
            var f = sidebarFriends.find(function(f) { return f.friend_id === uid; });
            if (f) {
                var name = f.remark || f.nickname || f.username || ('用户' + uid);
                conversations.push({ key: key, type: 'user', targetId: uid, name: name, avatarChar: name.charAt(0) });
            }
        }
    }

    resortSidebarItem(key);
    if (!document.querySelector('.sidebar-item[data-key="' + key + '"]')) {
        renderSidebar();
        if (activeTabKey) {
            document.querySelectorAll('.sidebar-item').forEach(function(el) { el.classList.toggle('active', el.dataset.key === activeTabKey); });
        }
    }

    if (chatTabs.has(key)) {
        var tab = chatTabs.get(key);
        var existing = findTabMessage(tab, msg.id);
        var wasStreaming = existing && existing._streaming;
        var added = upsertMessage(tab.messages, msg);
        if (!added) {
            if (key === activeTabKey) {
                if (!wasStreaming || !updateMessageElement(tab, findTabMessage(tab, msg.id))) {
                    renderAllMessages(tab);
                }
                tab.messagesEl.scrollTop = tab.messagesEl.scrollHeight;
            }
        } else if (key === activeTabKey) {
            tab.messagesEl.innerHTML += renderOneMessage(msg);
            tab.messagesEl.scrollTop = tab.messagesEl.scrollHeight;
        } else if (!msg.is_recalled && msg.from_user_id !== myId) { unreadCounts.set(key, (unreadCounts.get(key) || 0) + 1); updateTabBadge(key); }
    } else if (!msg.is_recalled && msg.from_user_id !== myId) { unreadCounts.set(key, (unreadCounts.get(key) || 0) + 1); }

    // 写入本地 IndexedDB
    msgStore.putMessage(key, msg);
    if (updateLatest) msgStore.setLastMsgID(key, msg.id);

    if (updateLatest) sortConversations();
    renderSidebar();
    updateOnlineDots();
    updateSidebarBadge(key);
    if (key === activeTabKey && !msg.is_recalled && msg.from_user_id !== getUserId()) {
        sendReadReceiptForTab(key);
    }
};

function findOpenMessageByID(messageID) {
    var found = null;
    chatTabs.forEach(function(tab, key) {
        if (found) return;
        var msg = findTabMessage(tab, messageID);
        if (msg) found = { key: key, tab: tab, msg: msg };
    });
    return found;
}

onAIStream = function(payload) {
    if (!payload || !payload.message_id) return;
    var found = findOpenMessageByID(payload.message_id);
    if (!found) return;

    var tab = found.tab;
    var msg = found.msg;
    if (payload.content !== undefined && payload.content !== null) {
        msg.content = payload.content;
    } else if (payload.delta) {
        msg.content = (msg.content || '') + payload.delta;
    }

    if (payload.error) {
        msg._streaming = false;
        msg._stream_error = payload.error;
    } else if (payload.done) {
        delete msg._streaming;
        delete msg._stream_error;
    } else {
        msg._streaming = true;
        delete msg._stream_error;
    }

    var latest = lastMessages.get(found.key);
    if (latest && latest.id === msg.id) {
        lastMessages.set(found.key, msg);
        updateSidebarPreview(found.key, msg);
    }
    msgStore.putMessage(found.key, msg);

    if (found.key === activeTabKey) {
        var nearBottom = tab.messagesEl.scrollHeight - tab.messagesEl.scrollTop - tab.messagesEl.clientHeight < 96;
        if (!updateMessageElement(tab, msg)) {
            renderAllMessages(tab);
        }
        if (nearBottom) tab.messagesEl.scrollTop = tab.messagesEl.scrollHeight;
    }
};

// ========================================

