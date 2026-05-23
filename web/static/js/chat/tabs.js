// 标签管理
// ========================================
function openOrSwitchTab(key, type, targetId, name) {
    if (chatTabs.has(key)) { switchToTab(key); }
    else { createTab(key, type, targetId, name); switchToTab(key); loadHistory(key, 1); }
    unreadCounts.set(key, 0);
    updateSidebarBadge(key); updateTabBadge(key); renderSidebar();
}

function createTab(key, type, targetId, name) {
    var tabEl = document.createElement('span');
    tabEl.className = 'chat-tab';
    tabEl.innerHTML = '<span class="tab-name">' + escapeHtml(name) + '</span>' +
        '<span class="tab-badge" style="display:none"></span>' +
        '<span class="tab-close">&times;</span>';
    tabEl.querySelector('.tab-name').addEventListener('click', function(e) { e.stopPropagation(); switchToTab(key); });
    tabEl.querySelector('.tab-close').addEventListener('click', function(e) { e.stopPropagation(); closeTab(key); });
    document.getElementById('tabBar').appendChild(tabEl);

    var panelEl = document.createElement('div');
    panelEl.className = 'tab-panel hidden';
    if (type === 'group') {
        panelEl.innerHTML =
            '<div class="chat-header">' +
                '<span class="chat-title">' + escapeHtml(name) + '</span>' +
                '<div class="chat-header-actions">' +
                    '<input class="chat-search-input" type="search" placeholder="搜索消息">' +
                    '<input class="chat-search-start" type="datetime-local" title="开始时间">' +
                    '<input class="chat-search-end" type="datetime-local" title="结束时间">' +
                    '<button class="chat-search-btn" type="button">搜索</button>' +
                    '<button class="chat-search-clear hidden" type="button" title="清除搜索">&times;</button>' +
                    '<button class="btn btn-sm btn-primary group-detail-btn" type="button">群详情</button>' +
                '</div>' +
            '</div>' +
            '<div class="chat-search-results hidden"></div>' +
            '<div class="chat-messages"></div>' +
            '<div class="quote-compose hidden"></div>' +
            '<div class="chat-input">' +
                '<button class="attach-btn" title="添加附件">+</button>' +
                '<input type="text" placeholder="输入消息，@ 提及成员...">' +
                '<button class="send-btn">发送</button>' +
            '</div>';
    } else {
        panelEl.innerHTML =
            '<div class="chat-header">' +
                '<span class="chat-title">' + escapeHtml(name) + '</span>' +
                '<div class="chat-header-actions">' +
                    '<input class="chat-search-input" type="search" placeholder="搜索消息">' +
                    '<input class="chat-search-start" type="datetime-local" title="开始时间">' +
                    '<input class="chat-search-end" type="datetime-local" title="结束时间">' +
                    '<button class="chat-search-btn" type="button">搜索</button>' +
                    '<button class="chat-search-clear hidden" type="button" title="清除搜索">&times;</button>' +
                '</div>' +
            '</div>' +
            '<div class="chat-search-results hidden"></div>' +
            '<div class="chat-messages"></div>' +
            '<div class="quote-compose hidden"></div>' +
            '<div class="chat-input">' +
                '<button class="attach-btn" title="添加附件">+</button>' +
                '<input type="text" placeholder="输入消息...">' +
                '<button class="send-btn">发送</button>' +
            '</div>';
    }
    var inputEl = panelEl.querySelector('.chat-input input');
    var btnEl = panelEl.querySelector('.send-btn');
    var attachBtn = panelEl.querySelector('.attach-btn');
    var searchInputEl = panelEl.querySelector('.chat-search-input');
    var searchStartEl = panelEl.querySelector('.chat-search-start');
    var searchEndEl = panelEl.querySelector('.chat-search-end');
    var searchBtnEl = panelEl.querySelector('.chat-search-btn');
    var searchClearEl = panelEl.querySelector('.chat-search-clear');
    var searchResultsEl = panelEl.querySelector('.chat-search-results');
    var quoteComposeEl = panelEl.querySelector('.quote-compose');
    inputEl.addEventListener('keydown', function(e) {
        if (e.key === 'Enter') sendMessage(key);
    });
    btnEl.addEventListener('click', function() { sendMessage(key); });
    attachBtn.addEventListener('click', function(e) {
        e.stopPropagation();
        toggleAttachMenu(key, this);
    });
    searchBtnEl.addEventListener('click', function() { searchMessagesInTab(key); });
    searchClearEl.addEventListener('click', function() { clearSearchResults(key); });
    searchInputEl.addEventListener('keydown', function(e) {
        if (e.key === 'Enter') {
            e.preventDefault();
            searchMessagesInTab(key);
        } else if (e.key === 'Escape') {
            clearSearchResults(key);
        }
    });
    [searchStartEl, searchEndEl].forEach(function(el) {
        el.addEventListener('keydown', function(e) {
            if (e.key === 'Enter') {
                e.preventDefault();
                searchMessagesInTab(key);
            }
        });
    });

    // 群聊 @ 提及支持
    if (type === 'group') {
        inputEl.addEventListener('input', function(e) { onMentionInput(key, this); });
        inputEl.addEventListener('blur', function() { setTimeout(hideMentionDropdown, 200); });
        // 预加载成员列表
        if (!groupMembersMap.has(targetId)) {
            api('GET', '/groups/' + targetId + '/members').then(function(data) {
                groupMembersMap.set(targetId, data.members || []);
            }).catch(function() {});
        }
    }

    document.getElementById('tabPanels').appendChild(panelEl);

    var panelMessagesEl = panelEl.querySelector('.chat-messages');
    panelMessagesEl.addEventListener('scroll', function() {
        if (this.scrollTop < 40 && chatTabs.has(key)) {
            var tab = chatTabs.get(key);
            if (tab.hasMore && !tab._loading) { tab._loading = true; loadHistory(key, tab.historyPage); }
        }
    });
    panelMessagesEl.addEventListener('click', function(e) {
        var recallBtn = e.target.closest('.recall-message-btn');
        if (recallBtn) {
            e.preventDefault();
            recallMessage(key, parseInt(recallBtn.dataset.msgId || '0', 10), recallBtn);
            return;
        }
        var deleteBtn = e.target.closest('.delete-message-btn');
        if (deleteBtn) {
            e.preventDefault();
            deleteMessageForMe(key, parseInt(deleteBtn.dataset.msgId || '0', 10), deleteBtn);
            return;
        }
        var quoteBtn = e.target.closest('.quote-reply-btn');
        if (quoteBtn) {
            e.preventDefault();
            startQuoteReply(key, parseInt(quoteBtn.dataset.msgId || '0', 10));
            return;
        }
        var quoteRef = e.target.closest('.quote-ref');
        if (quoteRef) {
            e.preventDefault();
            focusMessageElement(key, parseInt(quoteRef.dataset.quoteId || '0', 10));
        }
    });

    var headerEl = panelEl.querySelector('.chat-header');

    chatTabs.set(key, {
        type: type, targetId: targetId, name: name,
        messages: [], historyPage: 1, hasMore: true, _loading: false,
        panelEl: panelEl, messagesEl: panelMessagesEl,
        inputEl: inputEl, headerEl: headerEl, tabEl: tabEl,
        searchInputEl: searchInputEl, searchStartEl: searchStartEl, searchEndEl: searchEndEl,
        searchResultsEl: searchResultsEl, searchClearEl: searchClearEl, quoteComposeEl: quoteComposeEl
    });
    augmentTab(key, chatTabs.get(key));

    if (type === 'group') {
        headerEl.querySelector('.group-detail-btn').addEventListener('click', function(e) {
            e.stopPropagation(); toggleGroupPanel(key);
        });
    }
}

function switchToTab(key) {
    if (activeTabKey && chatTabs.has(activeTabKey)) {
        var old = chatTabs.get(activeTabKey);
        old.panelEl.classList.add('hidden'); old.tabEl.classList.remove('active');
    }
    var tab = chatTabs.get(key);
    tab.panelEl.classList.remove('hidden'); tab.tabEl.classList.add('active');
    activeTabKey = key; tab.inputEl.focus();
    var placeholder = document.getElementById('noChatPlaceholder');
    if (placeholder) placeholder.style.display = 'none';
    document.querySelectorAll('.sidebar-item').forEach(function(el) {
        el.classList.toggle('active', el.dataset.key === key);
    });
    updateOnlineDots();
    sendReadReceiptForTab(key);
    var panel = document.getElementById('groupPanel');
    if (!panel.classList.contains('hidden')) {
        if (tab.type === 'group') { currentGroupPanelKey = key; loadGroupPanel(key); }
        else closeGroupPanel();
    }
}

function closeTab(key) {
    var tab = chatTabs.get(key);
    if (!tab) return;
    var wasActive = (key === activeTabKey);
    var keys = Array.from(chatTabs.keys());
    var idx = keys.indexOf(key);
    var fallbackKey = keys[idx + 1] || keys[idx - 1] || null;
    if (currentGroupPanelKey === key) closeGroupPanel();
    quotedMessageByTab.delete(key);
    tab.panelEl.remove(); tab.tabEl.remove(); chatTabs.delete(key);
    if (wasActive) {
        activeTabKey = null;
        if (fallbackKey) switchToTab(fallbackKey);
        else { var ph = document.getElementById('noChatPlaceholder'); if (ph) ph.style.display = ''; }
    }
}

function updateTabBadge(key) {
    var tab = chatTabs.get(key);
    if (!tab) return;
    var badgeEl = tab.tabEl.querySelector('.tab-badge');
    var count = unreadCounts.get(key) || 0;
    if (count > 0) { badgeEl.textContent = count; badgeEl.style.display = 'inline'; }
    else badgeEl.style.display = 'none';
}

// ========================================

