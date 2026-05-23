checkAuth();

var navUserEl = document.getElementById('navUser');
navUserEl.textContent = getUserNickname();
navUserEl.href = '/profile';

// ========================================
// 核心状态
// ========================================
let sidebarFriends = [];
let sidebarGroups = [];
let sidebarAIBots = [];
let conversations = [];
let lastTimestamps = new Map();
let chatTabs = new Map();
let activeTabKey = null;
let unreadCounts = new Map();
let lastMessages = new Map();
let chatInitialized = false;
let pendingOnlineSync = false;
let onlineSyncRunning = false;
let quotedMessageByTab = new Map();
let pendingSends = new Map();
const MESSAGE_RECALL_WINDOW_MS = 2 * 60 * 1000;

// 群相关缓存
let groupMembersMap = new Map();   // groupId → members[]
let groupDNDMap = new Map();        // groupId → bool

// 群详情面板状态
let currentGroupPanelKey = null;
let groupPanelData = null;
let groupPanelActiveTab = 'info';

// @ 提及状态
let mentionActiveInput = null;
let mentionActiveKey = null;

// ========================================
// 数据加载
// ========================================
async function loadFriends() {
    try {
        const data = await api('GET', '/friends');
        sidebarFriends = data.friends || [];
    } catch(e) { sidebarFriends = []; }
}

async function loadGroups() {
    try {
        const data = await api('GET', '/groups');
        sidebarGroups = data.groups || [];
    } catch(e) { sidebarGroups = []; }
}

async function loadAIBots() {
    try {
        const data = await api('GET', '/ai/bots');
        sidebarAIBots = data.bots || [];
    } catch(e) { sidebarAIBots = []; }
}

// 预加载所有群的 DND 状态和成员列表
async function loadGroupMetadata() {
    var promises = sidebarGroups.map(function(g) {
        return api('GET', '/groups/' + g.id).then(function(data) {
            groupDNDMap.set(g.id, data.dnd || false);
        }).catch(function() {});
    });
    await Promise.all(promises);
}

// ========================================
// 统一会话列表构建 & 排序
// ========================================
function buildConversations() {
    conversations = [];
    function pushConversation(conv) {
        if (conversations.find(function(c) { return c.key === conv.key; })) return;
        conversations.push(conv);
    }

    pushConversation({
        key: 'broadcast',
        type: 'broadcast',
        targetId: 0,
        name: '广播消息',
        avatarChar: '📢'
    });
    sidebarFriends.forEach(function(f) {
        var name = f.remark || f.nickname || f.username || ('用户' + f.friend_id);
        conversations.push({
            key: 'user:' + f.friend_id, type: 'user', targetId: f.friend_id,
            name: name, avatarChar: name.charAt(0)
        });
    });
    sidebarAIBots.forEach(function(bot) {
        if (bot.status === 'disabled') return;
        var name = getAIBotName(bot);
        pushConversation({
            key: 'user:' + bot.user_id, type: 'user', targetId: bot.user_id,
            name: name, avatarChar: 'AI'
        });
    });
    sidebarGroups.forEach(function(g) {
        pushConversation({
            key: 'group:' + g.id, type: 'group', targetId: g.id,
            name: g.name, avatarChar: g.name.charAt(0)
        });
    });
}

function sortConversations() {
    conversations.sort(function(a, b) {
        if (a.key === 'broadcast') return -1;
        if (b.key === 'broadcast') return 1;
        var ta = lastTimestamps.get(a.key);
        var tb = lastTimestamps.get(b.key);
        if (ta && !tb) return -1;
        if (!ta && tb) return 1;
        if (!ta && !tb) return 0;
        return tb - ta;
    });
}

function formatTime(ts) {
    if (!ts) return '';
    var d = new Date(ts);
    var now = new Date();
    var today = new Date(now.getFullYear(), now.getMonth(), now.getDate());
    var yesterday = new Date(today.getTime() - 86400000);
    var msgDay = new Date(d.getFullYear(), d.getMonth(), d.getDate());
    if (msgDay.getTime() === today.getTime()) {
        var h = d.getHours(), m = d.getMinutes();
        return (h < 10 ? '0' : '') + h + ':' + (m < 10 ? '0' : '') + m;
    } else if (msgDay.getTime() === yesterday.getTime()) {
        return '昨天';
    } else if (msgDay.getTime() > today.getTime() - 604800000) {
        var days = ['日', '一', '二', '三', '四', '五', '六'];
        return '周' + days[d.getDay()];
    } else {
        return (d.getMonth() + 1) + '/' + d.getDate();
    }
}

async function loadLastMessages() {
    var fetches = conversations.map(function(c) {
        var url;
        if (c.type === 'user') url = '/history?peer_id=' + c.targetId + '&page=1&page_size=1';
        else if (c.type === 'group') url = '/history/group/' + c.targetId + '?page=1&page_size=1';
        else url = '/history/broadcast?page=1&page_size=1';
        return api('GET', url).then(function(data) {
            if (data.messages && data.messages.length > 0) {
                var msg = data.messages[0];
                lastMessages.set(c.key, msg);
                if (msg.created_at) lastTimestamps.set(c.key, new Date(msg.created_at));
            }
        }).catch(function() {});
    });
    await Promise.all(fetches);
}

// ========================================

