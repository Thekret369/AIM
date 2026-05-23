// 侧边栏渲染
// ========================================
function renderSidebar() {
    var list = document.getElementById('sidebarList');
    var html = '';
    conversations.forEach(function(c) {
        var unread = unreadCounts.get(c.key) || 0;
        var preview = lastMessages.get(c.key);
        var ts = lastTimestamps.get(c.key);
        var dnd = false;
        if (c.type === 'group') dnd = groupDNDMap.get(c.targetId) || false;
        html += sidebarItemHTML(c.key, c.type, c.targetId, c.avatarChar, c.name, preview, unread, ts, dnd);
    });
    list.innerHTML = html || '<div class="empty-state sidebar-empty">暂无会话</div>';
    list.querySelectorAll('.sidebar-item').forEach(function(el) {
        el.addEventListener('click', function() {
            var key = this.dataset.key, type = this.dataset.type;
            var targetId = parseInt(this.dataset.target) || 0;
            var name = this.querySelector('.item-name').textContent;
            openOrSwitchTab(key, type, targetId, name);
        });
    });
}

function sidebarItemHTML(key, type, targetId, avatarChar, name, previewMsg, unread, timestamp, dnd) {
    var activeClass = (key === activeTabKey) ? ' active' : '';
    var badgeHTML = unread > 0 ? '<span class="badge">' + unread + '</span>' : '';
    var previewHTML = previewMsg ? getPreviewText(previewMsg).substring(0, 20) : '';
    var timeHTML = timestamp ? '<span class="item-time">' + formatTime(timestamp) + '</span>' : '';
    var dndIcon = dnd ? '<span class="item-dnd">🔕</span>' : '';
    return '<div class="sidebar-item' + activeClass + '" data-key="' + key + '" data-type="' + type + '" data-target="' + targetId + '">' +
        '<div class="avatar">' + escapeHtml(avatarChar) + '</div>' +
        '<div class="item-info">' +
            '<div class="item-name">' + dndIcon + escapeHtml(name) + '</div>' +
            (previewHTML ? '<div class="item-preview">' + escapeHtml(previewHTML) + '</div>' : '') +
        '</div>' + timeHTML + badgeHTML +
    '</div>';
}

function ensureConversationForMessage(key, msg) {
    if (conversations.find(function(c) { return c.key === key; })) return;

    if (key.startsWith('group:')) {
        var gid = parseInt(key.split(':')[1]);
        var group = sidebarGroups.find(function(g) { return g.id === gid; });
        var groupName = group ? group.name : ('群组' + gid);
        conversations.push({
            key: key,
            type: 'group',
            targetId: gid,
            name: groupName,
            avatarChar: groupName.charAt(0)
        });
        return;
    }

    if (key.startsWith('user:')) {
        var uid = parseInt(key.split(':')[1]);
        var friend = sidebarFriends.find(function(f) { return f.friend_id === uid; });
        var bot = findAIBotByUserID(uid);
        var userName = '';
        if (friend) {
            userName = friend.remark || friend.nickname || friend.username || ('用户' + uid);
        } else if (bot) {
            userName = getAIBotName(bot);
        } else if (msg.from_user && msg.from_user_id === uid) {
            userName = msg.from_user.nickname || msg.from_user.username || ('用户' + uid);
        } else {
            userName = '用户' + uid;
        }
        conversations.push({
            key: key,
            type: 'user',
            targetId: uid,
            name: userName,
            avatarChar: userName.charAt(0)
        });
    }
}

function findAIBotByUserID(userID) {
    return sidebarAIBots.find(function(bot) { return bot.user_id === userID; });
}

function getAIBotName(bot) {
    return bot.nickname || bot.username || ('AI用户' + bot.user_id);
}

function updateSidebarBadge(key) {
    var el = document.querySelector('.sidebar-item[data-key="' + key + '"]');
    if (!el) return;
    var badgeEl = el.querySelector('.badge');
    var count = unreadCounts.get(key) || 0;
    if (count > 0) {
        if (!badgeEl) { badgeEl = document.createElement('span'); badgeEl.className = 'badge'; el.appendChild(badgeEl); }
        badgeEl.textContent = count; badgeEl.style.display = 'inline-block';
    } else if (badgeEl) { badgeEl.style.display = 'none'; }
}

function getPreviewText(msg) {
    if (msg && msg.is_recalled) return '消息已撤回';
    if (msg && msg._streaming) return msg.content || '正在生成';
    switch (msg.type) {
        case 'image': return '[图片]';
        case 'file': return '[文件] ' + (msg.file_name || '');
        case 'audio': return '[语音]';
        default: return msg.content || '';
    }
}

function shouldUseAsConversationLatest(key, msg) {
    var current = lastMessages.get(key);
    if (!current || !current.id || !msg || !msg.id) return true;
    if (current.id === msg.id) return true;
    return msg.id > current.id;
}

function updateSidebarPreview(key, msg) {
    var el = document.querySelector('.sidebar-item[data-key="' + key + '"] .item-preview');
    if (!el) return;
    el.textContent = getPreviewText(msg).substring(0, 20);
}

function resortSidebarItem(key) {
    var el = document.querySelector('.sidebar-item[data-key="' + key + '"]');
    if (!el || key === 'broadcast') return;
    sortConversations();
    var idx = -1;
    for (var i = 0; i < conversations.length; i++) { if (conversations[i].key === key) { idx = i; break; } }
    if (idx === -1) return;
    var list = document.getElementById('sidebarList');
    var nextKey = (idx + 1 < conversations.length) ? conversations[idx + 1].key : null;
    var nextEl = nextKey ? list.querySelector('.sidebar-item[data-key="' + nextKey + '"]') : null;
    if (nextEl) list.insertBefore(el, nextEl); else list.appendChild(el);
}

// ========================================

