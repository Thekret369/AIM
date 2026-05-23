// @ 提及下拉
// ========================================
function onMentionInput(key, inputEl) {
    var val = inputEl.value;
    var cursorPos = inputEl.selectionStart;
    // 找到光标前最近的 @
    var atIdx = val.lastIndexOf('@', cursorPos - 1);
    if (atIdx === -1 || (atIdx > 0 && val.charAt(atIdx - 1) !== ' ' && val.charAt(atIdx - 1) !== '')) {
        // @ 不在词首则隐藏（简单处理：@ 前面必须是空格或行首）
        if (atIdx > 0 && val.charAt(atIdx - 1) !== ' ') { hideMentionDropdown(); return; }
    }
    if (atIdx === -1) { hideMentionDropdown(); return; }

    var filter = val.substring(atIdx + 1, cursorPos).toLowerCase();
    var parts = key.split(':');
    var groupId = parseInt(parts[1]);
    var members = groupMembersMap.get(groupId) || [];
    if (members.length === 0) return;

    var myId = getUserId();
    var filtered = members.filter(function(m) {
        if (m.user_id === myId) return false; // 不 @ 自己
        var dn = (m.nickname || m.username || '').toLowerCase();
        return filter === '' || dn.indexOf(filter) !== -1;
    });

    if (filtered.length === 0) { hideMentionDropdown(); return; }

    mentionActiveInput = inputEl;
    mentionActiveKey = key;
    var dd = document.getElementById('mentionDropdown');
    dd.innerHTML = filtered.map(function(m) {
        var dn = m.nickname || m.username || ('用户' + m.user_id);
        return '<div class="mention-item" data-user-id="' + m.user_id + '" data-name="' + escapeHtml(dn) + '" onclick="insertMention(this)">' +
            '<span class="mention-avatar">' + escapeHtml(dn.charAt(0)) + '</span>' +
            '<span>' + escapeHtml(dn) + '</span>' +
        '</div>';
    }).join('');
    dd.classList.remove('hidden');
    // 定位到输入框上方
    var rect = inputEl.getBoundingClientRect();
    dd.style.left = rect.left + 'px';
    dd.style.top = (rect.top - dd.offsetHeight - 4) + 'px';
    dd.style.minWidth = '180px';
}

function insertMention(el) {
    if (!mentionActiveInput) return;
    var name = el.dataset.name;
    var inputEl = mentionActiveInput;
    var val = inputEl.value;
    var cursorPos = inputEl.selectionStart;
    var atIdx = val.lastIndexOf('@', cursorPos - 1);
    if (atIdx === -1) return;
    var before = val.substring(0, atIdx);
    var after = val.substring(cursorPos);
    inputEl.value = before + '@' + name + ' ' + after;
    var newPos = before.length + name.length + 2;
    inputEl.setSelectionRange(newPos, newPos);
    inputEl.focus();
    hideMentionDropdown();
}

function hideMentionDropdown() {
    document.getElementById('mentionDropdown').classList.add('hidden');
    mentionActiveInput = null;
    mentionActiveKey = null;
}

// ========================================
// 群详情面板
// ========================================
function toggleGroupPanel(key) {
    var panel = document.getElementById('groupPanel');
    if (panel.classList.contains('hidden')) {
        panel.classList.remove('hidden');
        currentGroupPanelKey = key; groupPanelActiveTab = 'info';
        document.querySelectorAll('.group-panel-tabs .gp-tab-btn').forEach(function(btn) { btn.classList.toggle('active', btn.dataset.gpTab === 'info'); });
        loadGroupPanel(key);
    } else if (currentGroupPanelKey === key) { closeGroupPanel(); }
    else { currentGroupPanelKey = key; groupPanelActiveTab = 'info';
        document.querySelectorAll('.group-panel-tabs .gp-tab-btn').forEach(function(btn) { btn.classList.toggle('active', btn.dataset.gpTab === 'info'); });
        loadGroupPanel(key); }
}

function closeGroupPanel() { document.getElementById('groupPanel').classList.add('hidden'); currentGroupPanelKey = null; groupPanelData = null; }

function switchGroupPanelTab(tab) {
    groupPanelActiveTab = tab;
    document.querySelectorAll('.group-panel-tabs .gp-tab-btn').forEach(function(btn) { btn.classList.toggle('active', btn.dataset.gpTab === tab); });
    renderGroupPanelBody();
}

async function loadGroupPanel(key) {
    var parts = key.split(':'), groupId = parseInt(parts[1]);
    if (!groupId) return;
    try {
        var detailData = await api('GET', '/groups/' + groupId);
        var membersData = await api('GET', '/groups/' + groupId + '/members');
        groupPanelData = {
            group: detailData.group, myRole: detailData.my_role || '',
            isMuted: detailData.is_muted || false,
            members: membersData.members || [],
            dnd: detailData.dnd || false
        };
        groupMembersMap.set(groupId, membersData.members || []);
        groupDNDMap.set(groupId, detailData.dnd || false);
        document.getElementById('gpTitle').textContent = groupPanelData.group.name || '群详情';
        updateDndBtn();
        renderGroupPanelBody();
    } catch(e) {
        document.getElementById('groupPanelBody').innerHTML = '<div class="gp-section gp-error">加载失败: ' + escapeHtml(e.message) + '</div>';
    }
}

function updateDndBtn() {
    var btn = document.getElementById('gpDndBtn');
    if (!groupPanelData) return;
    btn.textContent = groupPanelData.dnd ? '🔕' : '🔔';
    btn.title = groupPanelData.dnd ? '免打扰中，点击取消' : '消息提醒开启，点击免打扰';
}

function toggleGroupDND() {
    if (!currentGroupPanelKey) return;
    var parts = currentGroupPanelKey.split(':'), groupId = parseInt(parts[1]);
    api('POST', '/groups/' + groupId + '/dnd').then(function(data) {
        groupPanelData.dnd = data.dnd;
        groupDNDMap.set(groupId, data.dnd);
        updateDndBtn();
        renderSidebar(); // 更新侧边栏图标
    }).catch(function(e) { alert('操作失败: ' + e.message); });
}

function renderGroupPanelBody() {
    if (!groupPanelData) return;
    switch (groupPanelActiveTab) {
        case 'info': renderGroupInfo(); break;
        case 'announce': loadGroupAnnouncements(); break;
        case 'members': renderGroupMembers(); break;
    }
}

// ---- 群资料标签 ----
function renderGroupInfo() {
    var g = groupPanelData.group;
    var ownerName = '';
    var om = groupPanelData.members.find(function(m) { return m.user_id === g.owner_id; });
    if (om) ownerName = om.nickname || om.username || ('用户' + g.owner_id);
    else ownerName = '用户' + g.owner_id;
    var createdAt = g.created_at ? new Date(g.created_at).toLocaleString('zh-CN') : '-';

    var html = '<div class="gp-section"><h3>基本信息</h3>' +
        '<div class="gp-row"><span>群名称</span><span>' + escapeHtml(g.name) + '</span></div>' +
        '<div class="gp-row"><span>群主</span><span><a href="/profile?user_id=' + g.owner_id + '" class="text-link">' + escapeHtml(ownerName) + '</a></span></div>' +
        '<div class="gp-row"><span>创建时间</span><span>' + escapeHtml(createdAt) + '</span></div>' +
        '<div class="gp-row"><span>成员数</span><span>' + groupPanelData.members.length + '</span></div>' +
    '</div>';

    // 群简介
    html += '<div class="gp-section"><h3>群简介</h3>' +
        '<div class="gp-text">' + escapeHtml(g.description || '暂无简介') + '</div>' +
    '</div>';

    // 群主可编辑
    if (groupPanelData.myRole === 'owner') {
        html += '<div class="gp-section"><h3>编辑群资料</h3>' +
            '<label class="gp-label">群名称</label>' +
            '<input class="gp-input" id="gpEditName" value="' + escapeHtml(g.name) + '" placeholder="输入新群名称">' +
            '<label class="gp-label">群简介</label>' +
            '<input class="gp-input" id="gpEditDesc" value="' + escapeHtml(g.description || '') + '" placeholder="输入群简介">' +
            '<div class="gp-actions">' +
                '<button class="gp-btn gp-btn-primary" onclick="saveGroupInfo(' + g.id + ')">保存</button>' +
            '</div>' +
        '</div>';
    }

    document.getElementById('groupPanelBody').innerHTML = html;
}

function saveGroupInfo(groupId) {
    var newName = document.getElementById('gpEditName').value.trim();
    var newDesc = document.getElementById('gpEditDesc').value.trim();
    if (!newName) return;
    api('PUT', '/groups/' + groupId, { name: newName, description: newDesc }).then(function() {
        groupPanelData.group.name = newName;
        groupPanelData.group.description = newDesc;
        var key = 'group:' + groupId;
        var tab = chatTabs.get(key);
        if (tab) { tab.name = newName; tab.tabEl.querySelector('.tab-name').textContent = newName;
            var hSpan = tab.headerEl.querySelector('span'); if (hSpan) hSpan.textContent = newName; }
        var conv = conversations.find(function(c) { return c.key === key; });
        if (conv) conv.name = newName;
        var g = sidebarGroups.find(function(g) { return g.id === groupId; });
        if (g) { g.name = newName; g.description = newDesc; }
        renderSidebar(); renderGroupInfo();
    }).catch(function(e) { alert('保存失败: ' + e.message); });
}

// ---- 群公告标签 ----
async function loadGroupAnnouncements() {
    if (!currentGroupPanelKey) return;
    var parts = currentGroupPanelKey.split(':'), groupId = parseInt(parts[1]);
    try {
        var data = await api('GET', '/groups/' + groupId + '/announcements');
        renderGroupAnnounce(data.announcements || [], groupId);
    } catch(e) {
        document.getElementById('groupPanelBody').innerHTML = '<div class="gp-section gp-error">加载公告失败: ' + escapeHtml(e.message) + '</div>';
    }
}

function renderGroupAnnounce(announcements, groupId) {
    var html = '';
    // 发布公告表单
    if (groupPanelData.myRole === 'owner' || groupPanelData.myRole === 'admin') {
        html += '<div class="gp-section"><h3>发布公告</h3>' +
            '<textarea class="gp-textarea" id="gpNewAnnounce" placeholder="输入公告内容..."></textarea>' +
            '<div class="gp-actions">' +
                '<button class="gp-btn gp-btn-primary" onclick="publishAnnounce(' + groupId + ')">发布</button>' +
            '</div>' +
        '</div>';
    }

    // 公告历史
    html += '<div class="gp-section"><h3>公告历史 (' + announcements.length + ')</h3>';
    if (announcements.length === 0) {
        html += '<div class="empty-state compact">暂无公告</div>';
    } else {
        announcements.forEach(function(a) {
            var time = a.created_at ? new Date(a.created_at).toLocaleString('zh-CN') : '-';
            html += '<div class="announce-item">' +
                '<div class="announce-content">' + escapeHtml(a.content) + '</div>' +
                '<div class="announce-meta">' + escapeHtml(a.editor || '') + ' · ' + time + '</div>' +
            '</div>';
        });
    }
    html += '</div>';

    document.getElementById('groupPanelBody').innerHTML = html;
}

function publishAnnounce(groupId) {
    var text = document.getElementById('gpNewAnnounce').value.trim();
    if (!text) { alert('公告内容不能为空'); return; }
    api('POST', '/groups/' + groupId + '/announcements', { content: text }).then(function() {
        document.getElementById('gpNewAnnounce').value = '';
        // 同步更新 group.current announce
        groupPanelData.group.announce = text;
        loadGroupAnnouncements();
    }).catch(function(e) { alert('发布失败: ' + e.message); });
}

// ---- 群成员标签 ----
function renderGroupMembers() {
    var myRole = groupPanelData.myRole, myUserId = getUserId();
    var html = '<div class="gp-section"><h3>群成员 (' + groupPanelData.members.length + ')</h3>';

    if (myRole === 'owner' || myRole === 'admin') {
        html += '<div class="gp-row gp-add-member">' +
            '<input type="number" id="gpAddMemberInput" placeholder="输入用户 ID" class="gp-input">' +
            '<button class="gp-btn gp-btn-primary" onclick="addMemberAction(' + groupPanelData.group.id + ')">添加成员</button>' +
        '</div>';
    }

    groupPanelData.members.forEach(function(m) {
        var displayName = m.nickname || m.username || ('用户' + m.user_id);
        var roleLabel = '', roleCls = '';
        if (m.role === 'owner') { roleLabel = '群主'; roleCls = 'owner'; }
        else if (m.role === 'admin') { roleLabel = '管理员'; roleCls = 'admin'; }
        else { roleLabel = '成员'; roleCls = 'member'; }
        var isMuted = m.muted_until && new Date(m.muted_until) > new Date();
        var dndIcon = m.dnd ? ' 🔕' : '';

        html += '<div class="gp-member-item" data-user-id="' + m.user_id + '">' +
            '<div class="member-info">' +
                '<span>' + escapeHtml(displayName.charAt(0)) + '</span>' +
                '<a href="/profile?user_id=' + m.user_id + '" class="text-link" title="查看资料">' + escapeHtml(displayName) + '</a>' +
                '<span class="role-tag ' + roleCls + '">' + roleLabel + '</span>' +
                (isMuted ? '<span class="muted-tag">禁言中</span>' : '') +
                dndIcon +
            '</div><div class="gp-actions">';

        if (m.user_id !== myUserId) {
            if (myRole === 'owner' && m.role !== 'owner') {
                if (isMuted) html += '<button class="gp-btn gp-btn-warn" onclick="unmuteMemberAction(' + groupPanelData.group.id + ',' + m.user_id + ')">解禁</button>';
                else html += '<button class="gp-btn gp-btn-warn" onclick="muteMemberAction(' + groupPanelData.group.id + ',' + m.user_id + ')">禁言</button>';
                html += '<button class="gp-btn gp-btn-danger" onclick="kickMemberAction(' + groupPanelData.group.id + ',' + m.user_id + ')">踢出</button>';
                if (m.role === 'admin') html += '<button class="gp-btn gp-btn-muted" onclick="setAdminAction(' + groupPanelData.group.id + ',' + m.user_id + ')">取消管理</button>';
                else html += '<button class="gp-btn gp-btn-secondary" onclick="setAdminAction(' + groupPanelData.group.id + ',' + m.user_id + ')">设管理</button>';
                html += '<button class="gp-btn gp-btn-primary" onclick="transferOwnerAction(' + groupPanelData.group.id + ',' + m.user_id + ')">转让</button>';
            } else if (myRole === 'admin' && m.role === 'member') {
                if (isMuted) html += '<button class="gp-btn gp-btn-warn" onclick="unmuteMemberAction(' + groupPanelData.group.id + ',' + m.user_id + ')">解禁</button>';
                else html += '<button class="gp-btn gp-btn-warn" onclick="muteMemberAction(' + groupPanelData.group.id + ',' + m.user_id + ')">禁言</button>';
                html += '<button class="gp-btn gp-btn-danger" onclick="kickMemberAction(' + groupPanelData.group.id + ',' + m.user_id + ')">踢出</button>';
            }
        }
        html += '</div></div>';
    });

    html += '</div>';
    document.getElementById('groupPanelBody').innerHTML = html;
}

// ---- 群操作函数 ----
function addMemberAction(groupId) { /* ... 同上 ... */
    var userId = parseInt(document.getElementById('gpAddMemberInput').value);
    if (!userId) { alert('请输入有效的用户 ID'); return; }
    api('POST', '/groups/' + groupId + '/add-member', { user_id: userId }).then(function() {
        document.getElementById('gpAddMemberInput').value = ''; reloadGroupPanel();
    }).catch(function(e) { alert('添加失败: ' + e.message); });
}

function kickMemberAction(groupId, userId) {
    if (!confirm('确认踢出该成员？')) return;
    api('POST', '/groups/' + groupId + '/kick', { user_id: userId }).then(function() { reloadGroupPanel(); }).catch(function(e) { alert('操作失败: ' + e.message); });
}

function muteMemberAction(groupId, userId) {
    var minutes = prompt('请输入禁言分钟数:', '10');
    if (!minutes) return;
    minutes = parseInt(minutes); if (!minutes || minutes < 1) return;
    api('POST', '/groups/' + groupId + '/mute', { user_id: userId, minutes: minutes }).then(function() { reloadGroupPanel(); }).catch(function(e) { alert('操作失败: ' + e.message); });
}

function unmuteMemberAction(groupId, userId) {
    api('POST', '/groups/' + groupId + '/unmute', { user_id: userId }).then(function() { reloadGroupPanel(); }).catch(function(e) { alert('操作失败: ' + e.message); });
}

function setAdminAction(groupId, userId) {
    api('PUT', '/groups/' + groupId + '/admin/' + userId).then(function() { reloadGroupPanel(); }).catch(function(e) { alert('操作失败: ' + e.message); });
}

function transferOwnerAction(groupId, userId) {
    if (!confirm('确认将群主转让给该成员？此操作不可撤销。')) return;
    api('PUT', '/groups/' + groupId + '/transfer', { new_owner_id: userId }).then(function() { reloadGroupPanel(); }).catch(function(e) { alert('操作失败: ' + e.message); });
}

function reloadGroupPanel() {
    if (currentGroupPanelKey) {
        loadGroupPanel(currentGroupPanelKey);
        loadGroups().then(function() { buildConversations(); sortConversations(); renderSidebar(); });
    }
}

// ========================================

