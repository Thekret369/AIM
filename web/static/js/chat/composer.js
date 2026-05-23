// 附件上传 & 录音
// ========================================
var fileInput = null;
var audioRecorder = null;
var audioChunks = [];
var currentAttachKey = null;
var recording = false;

(function initFileInput() {
    fileInput = document.createElement('input');
    fileInput.type = 'file';
    fileInput.style.display = 'none';
    document.body.appendChild(fileInput);
})();

function toggleAttachMenu(key, btnEl) {
    var menu = document.getElementById('attachMenu');
    if (menu.classList.contains('hidden') || currentAttachKey !== key) {
        currentAttachKey = key;
        var rect = btnEl.getBoundingClientRect();
        menu.style.left = rect.left + 'px';
        menu.style.bottom = (window.innerHeight - rect.top + 6) + 'px';
        menu.classList.remove('hidden');
    } else {
        menu.classList.add('hidden');
        currentAttachKey = null;
    }
}

// 点击菜单外部关闭
document.addEventListener('click', function(e) {
    var menu = document.getElementById('attachMenu');
    if (!menu.classList.contains('hidden') && !menu.contains(e.target) && !e.target.classList.contains('attach-btn')) {
        menu.classList.add('hidden');
        currentAttachKey = null;
    }
});

function pickImage() {
    document.getElementById('attachMenu').classList.add('hidden');
    fileInput.accept = 'image/*';
    fileInput.onchange = function(e) {
        if (e.target.files[0]) uploadAndSend(currentAttachKey, e.target.files[0]);
        fileInput.value = '';
    };
    fileInput.click();
}

function pickFile() {
    document.getElementById('attachMenu').classList.add('hidden');
    fileInput.accept = '';
    fileInput.onchange = function(e) {
        if (e.target.files[0]) uploadAndSend(currentAttachKey, e.target.files[0]);
        fileInput.value = '';
    };
    fileInput.click();
}

function handleRecordToggle() {
    if (recording) stopRecord();
    else startRecord();
}

function startRecord() {
    document.getElementById('attachMenu').classList.add('hidden');
    navigator.mediaDevices.getUserMedia({ audio: true }).then(function(stream) {
        audioRecorder = new MediaRecorder(stream);
        audioChunks = [];
        audioRecorder.ondataavailable = function(e) { audioChunks.push(e.data); };
        audioRecorder.onstop = function() {
            stream.getTracks().forEach(function(t) { t.stop(); });
            var blob = new Blob(audioChunks, { type: 'audio/webm' });
            uploadAndSend(currentAttachKey, new File([blob], 'recording.webm', { type: 'audio/webm' }));
        };
        audioRecorder.start();
        recording = true;
        updateRecordUI();
    }).catch(function(e) {
        alert('无法访问麦克风: ' + e.message);
    });
}

function stopRecord() {
    if (audioRecorder) {
        audioRecorder.stop();
        audioRecorder = null;
        recording = false;
        updateRecordUI();
    }
}

function updateRecordUI() {
    var item = document.getElementById('attachRecordItem');
    if (!item) return;
    if (recording) {
        item.classList.add('recording');
        item.querySelector('span').textContent = '停止录音';
    } else {
        item.classList.remove('recording');
        item.querySelector('span').textContent = '录音';
    }
}

async function uploadAndSend(key, file) {
    if (!key) return;
    var formData = new FormData();
    formData.append('file', file);

    try {
        var resp = await fetch('/api/upload', {
            method: 'POST',
            headers: { 'Authorization': 'Bearer ' + getToken() },
            body: formData
        });
        var data = await resp.json();
        if (!resp.ok) throw new Error(data.error || '上传失败');

        var type = 'file';
        if (file.type.startsWith('image/')) type = 'image';
        else if (file.type.startsWith('audio/')) type = 'audio';

        var msg = {
            type: type,
            from_user_id: getUserId(),
            content: data.url,
            file_name: data.file_name,
            file_size: data.file_size,
            thumbnail_url: data.thumbnail_url || ''
        };

        var tab = chatTabs.get(key);
        if (!tab) return;
        if (tab.type === 'user') msg.to_user_id = tab.targetId;
        else if (tab.type === 'group') msg.group_id = tab.targetId;
        var quoted = quotedMessageByTab.get(key);
        if (quoted) msg.quote_message_id = quoted.id;

        var requestId = sendWS(msg);
        registerPendingSend(requestId, key, msg);
        clearQuoteReply(key);
    } catch(e) {
        alert('上传失败: ' + e.message);
    }
}

// ========================================
// 发送消息（含 @ 提及提取）
// ========================================
function sendMessage(key) {
    var tab = chatTabs.get(key || activeTabKey);
    if (!tab) return;
    var content = tab.inputEl.value.trim();
    if (!content) return;

    var msg = { type: 'text', from_user_id: getUserId(), content: content };
    if (tab.type === 'user') { msg.to_user_id = tab.targetId; }
    else if (tab.type === 'group') {
        msg.group_id = tab.targetId;
        // 提取 @ 提及的用户 ID
        var members = groupMembersMap.get(tab.targetId) || [];
        var mentionIds = [];
        var re = /@(\S+)/g;
        var match;
        while ((match = re.exec(content)) !== null) {
            var atName = match[1].toLowerCase();
            var found = members.find(function(m) {
                var dn = (m.nickname || m.username || '').toLowerCase();
                return dn === atName;
            });
            if (found && mentionIds.indexOf(found.user_id) === -1) mentionIds.push(found.user_id);
        }
        if (mentionIds.length > 0) msg.mentions = JSON.stringify(mentionIds);
    }
    var quoted = quotedMessageByTab.get(key || activeTabKey);
    if (quoted) msg.quote_message_id = quoted.id;

    var requestId = sendWS(msg);
    registerPendingSend(requestId, key || activeTabKey, msg);
    tab.inputEl.value = '';
    clearQuoteReply(key || activeTabKey);
    if (tab.type === 'user') sendTyping(tab.targetId, 0, false);
    else if (tab.type === 'group') sendTyping(0, tab.targetId, false);
};

// ========================================

