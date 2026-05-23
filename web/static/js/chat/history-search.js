// 历史消息加载
// ========================================
async function loadHistory(key, page) {
    var tab = chatTabs.get(key);
    if (!tab) return;
    page = page || 1;

    // 初始加载：先从本地 IndexedDB 瞬时渲染
    if (page === 1 && !tab._localLoaded) {
        tab._localLoaded = true;
        try {
            var cached = await msgStore.getMessages(key);
            if (cached.length > 0) {
                tab.messages = cached;
                renderAllMessages(tab);
                tab.messagesEl.scrollTop = tab.messagesEl.scrollHeight;
            }
        } catch(e) {}
    }

    // 状态更新会复用旧消息 ID，首屏固定拉最新页，避免本地缓存错过撤回态。
    var url, afterID = 0;
    if (tab.type === 'user') {
        url = '/history?peer_id=' + tab.targetId + '&page=' + page + '&page_size=30';
    } else if (tab.type === 'group') {
        url = '/history/group/' + tab.targetId + '?page=' + page + '&page_size=30';
    } else {
        url = '/history/broadcast?page=' + page + '&page_size=30';
    }

    try {
        var data = await api('GET', url);
        tab._loading = false;
        if (!data.messages || data.messages.length === 0) {
            if (data.read_info) {
                persistReadMarks(key, applyReadInfoToMessages(tab.messages, data.read_info));
                if (key === activeTabKey && typeof refreshReadMarks === 'function') refreshReadMarks(tab);
            }
            if (tab.type === 'group' && typeof loadGroupReads === 'function' && tab.targetId) loadGroupReads(tab.targetId, tab);
            if (!afterID) tab.hasMore = false;
            return;
        }

        // after_id 增量时服务端返回升序，全量时为降序需反转
        var newMsgs = afterID ? data.messages : data.messages.reverse();

        // 根据服务端 read_info 恢复已读标记
        if (data.read_info) {
            persistReadMarks(key, applyReadInfoToMessages(afterID ? tab.messages.concat(newMsgs) : newMsgs, data.read_info));
        }

        // 合并消息（按 ID 去重）
        var existingIds = new Set(tab.messages.map(function(m) { return m.id; }));
        var toAdd = newMsgs.filter(function(m) { return !existingIds.has(m.id); });

        if (page === 1 && afterID) {
            tab.messages = tab.messages.concat(toAdd);               // 增量追加
        } else if (page === 1) {
            tab.messages = mergeMessageLists(tab.messages, newMsgs); // 首次全量
        } else {
            tab.messages = toAdd.concat(tab.messages);               // 翻页前置
        }

        // 服务端返回的同 ID 状态也要回写本地缓存，例如撤回状态。
        if (newMsgs.length > 0) {
            msgStore.putMessages(key, newMsgs);
            var maxID = newMsgs.reduce(function(max, m) { return m.id > max ? m.id : max; }, 0);
            if (page === 1 && maxID > 0) msgStore.setLastMsgID(key, maxID);
        }

        var oldScrollHeight = tab.messagesEl.scrollHeight;
        renderAllMessages(tab);
        if (page === 1) tab.messagesEl.scrollTop = tab.messagesEl.scrollHeight;
        else { tab.messagesEl.scrollTop = tab.messagesEl.scrollHeight - oldScrollHeight; }

        // 消息加载完成后发送已读回执（switchToTab 时 tab.messages 尚为空）
        sendReadReceiptForTab(key);
        if (tab.type === 'group' && typeof loadGroupReads === 'function' && tab.targetId) loadGroupReads(tab.targetId, tab);
        tab.historyPage = page + 1;
        if (data.messages.length < (afterID ? 50 : 30)) tab.hasMore = false;
    } catch(e) { tab._loading = false; console.error(e); }
}

async function searchMessagesInTab(key) {
    var tab = chatTabs.get(key);
    if (!tab || !tab.searchInputEl) return;
    var q = (tab.searchInputEl.value || '').trim();
    var startTime = tab.searchStartEl ? tab.searchStartEl.value : '';
    var endTime = tab.searchEndEl ? tab.searchEndEl.value : '';
    if (!q && !startTime && !endTime) { clearSearchResults(key); return; }

    var url = '/search/messages?type=' + encodeURIComponent(tab.type) +
        '&q=' + encodeURIComponent(q) + '&page=1&page_size=30';
    if (tab.type === 'user' || tab.type === 'group') {
        url += '&target_id=' + encodeURIComponent(tab.targetId);
    }
    if (startTime) url += '&start_time=' + encodeURIComponent(startTime);
    if (endTime) url += '&end_time=' + encodeURIComponent(endTime);

    tab.searchResultsEl.classList.remove('hidden');
    tab.searchClearEl.classList.remove('hidden');
    tab.searchResultsEl.innerHTML = '<div class="search-state">搜索中...</div>';

    try {
        var data = await api('GET', url);
        renderSearchResults(key, data.messages || [], data.total || 0, q);
    } catch(e) {
        tab.searchResultsEl.innerHTML = '<div class="search-state">搜索失败: ' + escapeHtml(e.message) + '</div>';
    }
}

function renderSearchResults(key, messages, total, q) {
    var tab = chatTabs.get(key);
    if (!tab || !tab.searchResultsEl) return;
    if (!messages.length) {
        tab.searchResultsEl.innerHTML = '<div class="search-state">没有找到相关消息</div>';
        return;
    }

    var html = '<div class="search-summary">找到 ' + total + ' 条结果</div>';
    for (var i = 0; i < messages.length; i++) {
        var m = messages[i];
        var sender = m.from_user ? (m.from_user.nickname || m.from_user.username) : ('用户' + m.from_user_id);
        html += '<button class="search-result" type="button" data-msg-id="' + m.id + '">' +
            '<span class="search-result-meta">' + escapeHtml(sender) + ' · ' + escapeHtml(formatMsgTime(m.created_at)) + '</span>' +
            '<span class="search-result-text">' + highlightSearchText(getPreviewText(m), q) + '</span>' +
        '</button>';
    }

    tab.searchResultsEl.innerHTML = html;
    tab.searchResultsEl.querySelectorAll('.search-result').forEach(function(btn) {
        btn.addEventListener('click', function() {
            var id = parseInt(this.dataset.msgId || '0', 10);
            var msg = null;
            for (var i = 0; i < messages.length; i++) {
                if (messages[i].id === id) { msg = messages[i]; break; }
            }
            focusSearchResult(key, msg);
        });
    });
}

function clearSearchResults(key) {
    var tab = chatTabs.get(key);
    if (!tab) return;
    if (tab.searchInputEl) tab.searchInputEl.value = '';
    if (tab.searchStartEl) tab.searchStartEl.value = '';
    if (tab.searchEndEl) tab.searchEndEl.value = '';
    if (tab.searchResultsEl) {
        tab.searchResultsEl.classList.add('hidden');
        tab.searchResultsEl.innerHTML = '';
    }
    if (tab.searchClearEl) tab.searchClearEl.classList.add('hidden');
}

function focusSearchResult(key, msg) {
    if (!msg) return;
    var tab = chatTabs.get(key);
    if (!tab) return;

    if (upsertMessage(tab.messages, msg)) {
        renderAllMessages(tab);
    }

    focusMessageElement(key, msg.id);
}

function highlightSearchText(text, q) {
    text = text || '';
    q = q || '';
    if (!q) return escapeHtml(text);
    var lower = text.toLowerCase();
    var needle = q.toLowerCase();
    var idx = lower.indexOf(needle);
    if (idx < 0) return escapeHtml(text);
    return escapeHtml(text.slice(0, idx)) +
        '<mark>' + escapeHtml(text.slice(idx, idx + q.length)) + '</mark>' +
        escapeHtml(text.slice(idx + q.length));
}

async function globalSearchMessages() {
    var inputEl = document.getElementById('globalSearchInput');
    var startEl = document.getElementById('globalSearchStart');
    var endEl = document.getElementById('globalSearchEnd');
    var groupedEl = document.getElementById('globalSearchGrouped');
    var resultsEl = document.getElementById('globalSearchResults');
    var q = (inputEl.value || '').trim();
    var startTime = startEl.value || '';
    var endTime = endEl.value || '';
    var grouped = groupedEl.checked;
    if (!q && !startTime && !endTime) { clearGlobalSearch(); return; }

    var url = '/search/messages?type=all&page=1&page_size=50&q=' + encodeURIComponent(q);
    if (startTime) url += '&start_time=' + encodeURIComponent(startTime);
    if (endTime) url += '&end_time=' + encodeURIComponent(endTime);
    if (grouped) url += '&group_by=conversation';

    resultsEl.classList.remove('hidden');
    resultsEl.innerHTML = '<div class="search-state">查找中...</div>';
    try {
        var data = await api('GET', url);
        renderGlobalSearchResults(data, q, grouped);
    } catch(e) {
        resultsEl.innerHTML = '<div class="search-state">查找失败: ' + escapeHtml(e.message) + '</div>';
    }
}

function clearGlobalSearch() {
    document.getElementById('globalSearchInput').value = '';
    document.getElementById('globalSearchStart').value = '';
    document.getElementById('globalSearchEnd').value = '';
    var resultsEl = document.getElementById('globalSearchResults');
    resultsEl.classList.add('hidden');
    resultsEl.innerHTML = '';
}

function renderGlobalSearchResults(data, q, grouped) {
    var resultsEl = document.getElementById('globalSearchResults');
    var hits = [];
    var html = '<div class="search-summary">找到 ' + (data.total || 0) + ' 条结果</div>';

    if (grouped && data.conversations && data.conversations.length) {
        for (var i = 0; i < data.conversations.length; i++) {
            var group = data.conversations[i];
            html += '<div class="global-search-group">' + escapeHtml(conversationTitle(group.conversation_type, group.conversation_id)) +
                '<span>' + group.count + '</span></div>';
            var msgs = group.messages || [];
            for (var j = 0; j < msgs.length; j++) {
                hits.push(msgs[j]);
                html += globalSearchResultButton(hits.length - 1, msgs[j], q);
            }
        }
    } else {
        var messages = data.messages || [];
        for (var k = 0; k < messages.length; k++) {
            hits.push(messages[k]);
            html += globalSearchResultButton(hits.length - 1, messages[k], q);
        }
    }

    if (hits.length === 0) html += '<div class="search-state">没有找到相关消息</div>';
    resultsEl.innerHTML = html;
    resultsEl.querySelectorAll('.global-search-result').forEach(function(btn) {
        btn.addEventListener('click', function() {
            var idx = parseInt(this.dataset.index || '-1', 10);
            if (idx >= 0 && hits[idx]) openSearchResultMessage(hits[idx]);
        });
    });
}

function globalSearchResultButton(index, msg, q) {
    var sender = msg.from_user ? (msg.from_user.nickname || msg.from_user.username) : ('用户' + msg.from_user_id);
    return '<button class="search-result global-search-result" type="button" data-index="' + index + '">' +
        '<span class="search-result-meta">' + escapeHtml(conversationTitleForMessage(msg)) + ' · ' +
            escapeHtml(sender) + ' · ' + escapeHtml(formatMsgTime(msg.created_at)) + '</span>' +
        '<span class="search-result-text">' + highlightSearchText(getPreviewText(msg), q) + '</span>' +
    '</button>';
}

function conversationTitleForMessage(msg) {
    var conv = conversationFromMessage(msg);
    return conversationTitle(conv.type, conv.targetId);
}

function conversationTitle(type, targetId) {
    if (type === 'broadcast') return '广播消息';
    var key = type + ':' + targetId;
    var conv = conversations.find(function(c) { return c.key === key; });
    if (conv) return conv.name;
    if (type === 'group') return '群组' + targetId;
    return '用户' + targetId;
}

function conversationFromMessage(msg) {
    if (msg.group_id !== null && msg.group_id !== undefined && msg.group_id !== 0) {
        return { key: 'group:' + msg.group_id, type: 'group', targetId: msg.group_id };
    }
    if (msg.to_user_id !== null && msg.to_user_id !== undefined && msg.to_user_id !== 0) {
        var peerId = msg.from_user_id === getUserId() ? msg.to_user_id : msg.from_user_id;
        return { key: 'user:' + peerId, type: 'user', targetId: peerId };
    }
    return { key: 'broadcast', type: 'broadcast', targetId: 0 };
}

function openSearchResultMessage(msg) {
    var conv = conversationFromMessage(msg);
    ensureConversationForMessage(conv.key, msg);
    openOrSwitchTab(conv.key, conv.type, conv.targetId, conversationTitle(conv.type, conv.targetId));
    setTimeout(function() { focusSearchResult(conv.key, msg); }, 120);
}

// ========================================

