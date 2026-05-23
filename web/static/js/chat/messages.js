// 消息渲染
// ========================================
function formatMsgTime(createdAt) {
	    if (!createdAt) return '';
	    var d = new Date(createdAt);
	    var now = new Date();
	    var time = d.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' });
	    var sameDay = d.getFullYear() === now.getFullYear() &&
	                  d.getMonth() === now.getMonth() &&
	                  d.getDate() === now.getDate();
	    if (sameDay) return time;
	    return (d.getMonth() + 1) + '-' + d.getDate() + ' ' + time;
	}

function quoteMessageSummary(msg) {
    if (!msg) return '';
    if (msg.is_recalled) return '消息已撤回';
    return getPreviewText(msg).replace(/\s+/g, ' ').substring(0, 80);
}

function renderQuoteMessage(quote) {
    if (!quote) return '';
    var sender = quote.from_user ? (quote.from_user.nickname || quote.from_user.username) : ('用户' + quote.from_user_id);
    return '<button class="quote-ref" type="button" data-quote-id="' + quote.id + '">' +
        '<span class="quote-ref-sender">' + escapeHtml(sender) + '</span>' +
        '<span class="quote-ref-text">' + escapeHtml(quoteMessageSummary(quote)) + '</span>' +
    '</button>';
}

function canRecallMessage(msg) {
    if (!msg || !msg.id || msg.is_recalled) return false;
    if (msg.from_user_id !== getUserId()) return false;
    if (!msg.to_user_id && !msg.group_id) return false;
    var createdAt = new Date(msg.created_at || 0).getTime();
    if (!createdAt) return false;
    return Date.now() - createdAt <= MESSAGE_RECALL_WINDOW_MS;
}

function renderOneMessage(msg) {
    var isMe = msg.from_user_id === getUserId();
    var cls = isMe ? 'from-me' : '';
    if (msg.is_recalled) cls += ' recalled';
    var name = msg.from_user ? (msg.from_user.nickname || msg.from_user.username) : ('用户' + msg.from_user_id);
    var body = '';
    if (msg.is_recalled) {
        body = '<div class="msg-recalled">消息已撤回</div>';
    } else {
        switch (msg.type) {
            case 'image':
                var src = msg.thumbnail_url || msg.content;
                body = '<div class="msg-image"><img src="' + escapeHtml(src) + '" loading="lazy" onclick="viewImage(\'' + escapeHtml(msg.content) + '\')"></div>';
                break;
            case 'file':
                body = '<div class="msg-file">' +
                    '<div class="file-icon">📄</div>' +
                    '<div class="file-info">' +
                        '<div class="file-name" title="' + escapeHtml(msg.file_name || '') + '">' + escapeHtml(msg.file_name || '未知文件') + '</div>' +
                        '<div class="file-size">' + formatFileSize(msg.file_size || 0) + '</div>' +
                    '</div>' +
                    '<a class="file-download" href="' + escapeHtml(msg.content) + '" download="' + escapeHtml(msg.file_name || '') + '">下载</a>' +
                '</div>';
                break;
            case 'audio':
                body = '<div class="msg-audio"><audio controls src="' + escapeHtml(msg.content) + '"></audio></div>';
                break;
            default:
                if (msg._streaming) {
                    var streamText = msg.content ? escapeHtml(msg.content) : '<span class="msg-stream-placeholder">正在生成</span>';
                    body = '<span class="msg-stream-text">' + streamText + '</span><span class="msg-stream-cursor"></span>';
                } else {
                    body = escapeHtml(msg.content);
                }
                if (msg._stream_error) {
                    body += '<div class="msg-stream-error">生成失败</div>';
                }
        }
    }
    var timeStr = formatMsgTime(msg.created_at);
    var quoteHTML = msg.is_recalled ? '' : renderQuoteMessage(msg.quote_message);
    var quoteAction = (!msg.is_recalled && (msg.to_user_id || msg.group_id))
        ? '<button class="quote-reply-btn" type="button" data-msg-id="' + msg.id + '" title="引用回复">引用</button>'
        : '';
    var recallAction = canRecallMessage(msg)
        ? '<button class="recall-message-btn" type="button" data-msg-id="' + msg.id + '" title="2 分钟内可撤回">撤回</button>'
        : '';
    var deleteAction = msg.id
        ? '<button class="delete-message-btn" type="button" data-msg-id="' + msg.id + '" title="仅从我的聊天记录中删除">删除</button>'
        : '';
    var html = '<div class="msg ' + cls + '" data-msg-id="' + msg.id + '">' +
        '<div class="bubble">' +
            '<div class="meta"><a href="/profile?user_id=' + msg.from_user_id + '" class="text-link">' + escapeHtml(name) + '</a>' +
            (timeStr ? ' <span class="msg-time">' + timeStr + '</span>' : '') + quoteAction + recallAction + deleteAction + '</div>' +
            quoteHTML +
            body +
        '</div>' +
    '</div>';
    if (typeof injectReadMark === 'function') html = injectReadMark(html, msg);
    return html;
}

function formatFileSize(bytes) {
    if (bytes < 1024) return bytes + ' B';
    if (bytes < 1048576) return (bytes / 1024).toFixed(1) + ' KB';
    return (bytes / 1048576).toFixed(1) + ' MB';
}

function viewImage(url) {
    var viewer = document.createElement('div');
    viewer.className = 'image-viewer';
    viewer.innerHTML = '<img src="' + escapeHtml(url) + '">';
    viewer.addEventListener('click', function() { viewer.remove(); });
    document.body.appendChild(viewer);
}

function renderAllMessages(tab) {
    var html = '';
    for (var i = 0; i < tab.messages.length; i++) html += renderOneMessage(tab.messages[i]);
    tab.messagesEl.innerHTML = html;
}

function findTabMessage(tab, msgId) {
    if (!tab || !msgId) return null;
    for (var i = 0; i < tab.messages.length; i++) {
        if (tab.messages[i].id === msgId) return tab.messages[i];
    }
    return null;
}

function startQuoteReply(key, msgId) {
    var tab = chatTabs.get(key);
    var msg = findTabMessage(tab, msgId);
    if (!tab || !msg) return;
    quotedMessageByTab.set(key, msg);
    renderQuoteCompose(key);
    tab.inputEl.focus();
}

function clearQuoteReply(key) {
    quotedMessageByTab.delete(key);
    renderQuoteCompose(key);
}

function renderQuoteCompose(key) {
    var tab = chatTabs.get(key);
    if (!tab || !tab.quoteComposeEl) return;
    var msg = quotedMessageByTab.get(key);
    if (!msg) {
        tab.quoteComposeEl.classList.add('hidden');
        tab.quoteComposeEl.innerHTML = '';
        return;
    }
    var sender = msg.from_user ? (msg.from_user.nickname || msg.from_user.username) : ('用户' + msg.from_user_id);
    tab.quoteComposeEl.classList.remove('hidden');
    tab.quoteComposeEl.innerHTML =
        '<div class="quote-compose-content">' +
            '<span class="quote-compose-label">引用 ' + escapeHtml(sender) + '</span>' +
            '<span class="quote-compose-text">' + escapeHtml(quoteMessageSummary(msg)) + '</span>' +
        '</div>' +
        '<button type="button" class="quote-compose-clear" onclick="clearQuoteReply(\'' + escapeHtml(key) + '\')">&times;</button>';
}

function focusMessageElement(key, msgId) {
    var tab = chatTabs.get(key);
    if (!tab || !msgId) return;
    var el = tab.messagesEl.querySelector('.msg[data-msg-id="' + msgId + '"]');
    if (!el) return;
    el.scrollIntoView({ block: 'center' });
    el.classList.add('search-hit');
    setTimeout(function() { el.classList.remove('search-hit'); }, 1600);
}

async function recallMessage(key, msgId, buttonEl) {
    var tab = chatTabs.get(key);
    var msg = findTabMessage(tab, msgId);
    if (!tab || !msg || !canRecallMessage(msg)) return;
    if (buttonEl) buttonEl.disabled = true;
    try {
        var data = await api('POST', '/messages/' + msgId + '/recall', {});
        if (data.message && onMessage) onMessage(data.message);
    } catch(e) {
        renderSendFailure(key, null, e.message || '撤回失败');
        if (buttonEl) buttonEl.disabled = false;
    }
}

async function deleteMessageForMe(key, msgId, buttonEl) {
    var tab = chatTabs.get(key);
    var msg = findTabMessage(tab, msgId);
    if (!tab || !msg || !msgId) return;
    if (!confirm('确认删除这条聊天记录？删除后仅自己不可见。')) return;
    if (buttonEl) buttonEl.disabled = true;

    try {
        await api('DELETE', '/messages/' + msgId);
        if (typeof msgStore !== 'undefined' && typeof msgStore.deleteMessage === 'function') {
            await msgStore.deleteMessage(key, msgId);
        }
        removeMessageFromTab(key, msgId);
    } catch(e) {
        renderSendFailure(key, null, e.message || '删除失败');
        if (buttonEl) buttonEl.disabled = false;
    }
}

function removeMessageFromTab(key, msgId) {
    var tab = chatTabs.get(key);
    if (!tab || !msgId) return;

    var removed = null;
    tab.messages = tab.messages.filter(function(msg) {
        if (msg.id === msgId) {
            removed = msg;
            return false;
        }
        return true;
    });
    if (!removed) return;

    messageReadBy.delete(msgId);
    var quoted = quotedMessageByTab.get(key);
    if (quoted && quoted.id === msgId) clearQuoteReply(key);
    renderAllMessages(tab);

    var latest = lastMessages.get(key);
    if (latest && latest.id === msgId) {
        lastMessages.delete(key);
        lastTimestamps.delete(key);
        loadLastMessages().then(function() {
            sortConversations();
            renderSidebar();
            updateOnlineDots();
        });
    } else {
        renderSidebar();
        updateOnlineDots();
    }
}

function mergeMessageState(existing, incoming) {
    if (!existing) return incoming;
    if (existing._read && !incoming._read) incoming._read = true;
    for (var k in incoming) {
        if (Object.prototype.hasOwnProperty.call(incoming, k)) existing[k] = incoming[k];
    }
    if (!incoming._streaming) delete existing._streaming;
    if (!incoming._stream_error) delete existing._stream_error;
    return existing;
}

function upsertMessage(list, msg) {
    for (var i = 0; i < list.length; i++) {
        if (list[i].id === msg.id) {
            mergeMessageState(list[i], msg);
            return false;
        }
    }
    list.push(msg);
    list.sort(function(a, b) { return a.id - b.id; });
    return true;
}

function mergeMessageLists(existing, incoming) {
    for (var i = 0; i < incoming.length; i++) {
        upsertMessage(existing, incoming[i]);
    }
    return existing;
}

function applyReadInfoToMessages(messages, info) {
    if (!messages || !info) return [];
    var readIds = [];
    var myId = getUserId();
    for (var i = 0; i < messages.length; i++) {
        var m = messages[i];
        var shouldMark = false;
        if (m.from_user_id === myId && info.peer_last_read && m.id <= info.peer_last_read) {
            shouldMark = true;
        } else if (m.from_user_id !== myId && info.my_last_read && m.id <= info.my_last_read) {
            shouldMark = true;
        }
        if (shouldMark && !m._read) {
            m._read = true;
            readIds.push(m.id);
        }
    }
    return readIds;
}

function persistReadMarks(key, readIds) {
    if (readIds.length > 0 && typeof msgStore !== 'undefined') msgStore.markRead(key, readIds);
}

// ========================================

