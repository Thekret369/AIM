// AIM 消息本地存储 — IndexedDB 封装
// 提供消息离线缓存、增量同步元数据、已读状态持久化

var msgStore = (function() {
    var DB_NAME = 'AIM';
    var DB_VERSION = 1;
    var db = null;

    // 打开或创建数据库
    function open() {
        return new Promise(function(resolve, reject) {
            if (db) return resolve(db);
            var req = indexedDB.open(DB_NAME, DB_VERSION);
            req.onupgradeneeded = function(e) {
                var d = e.target.result;
                if (!d.objectStoreNames.contains('messages')) {
                    var store = d.createObjectStore('messages', { keyPath: 'conv_msg_id' });
                    store.createIndex('conv_key', 'conv_key', { unique: false });
                    store.createIndex('msg_id', 'msg_id', { unique: false });
                }
                if (!d.objectStoreNames.contains('meta')) {
                    d.createObjectStore('meta', { keyPath: 'conv_key' });
                }
            };
            req.onsuccess = function(e) {
                db = e.target.result;
                resolve(db);
            };
            req.onerror = function(e) {
                console.error('[msgStore] open failed:', e.target.error);
                reject(e.target.error);
            };
        });
    }

    // 构造复合键
    function makeKey(convKey, msgId) {
        return convKey + '_' + msgId;
    }

    // ========================================
    // 消息存取
    // ========================================

    // 存入单条消息（WS 实时到达）
    function putMessage(convKey, msg) {
        return open().then(function(db) {
            return new Promise(function(resolve, reject) {
                var tx = db.transaction('messages', 'readwrite');
                var store = tx.objectStore('messages');
                var record = {
                    conv_msg_id: makeKey(convKey, msg.id),
                    conv_key: convKey,
                    msg_id: msg.id,
                    data: msg,
                    created_at: msg.created_at || ''
                };
                store.put(record);
                tx.oncomplete = function() { resolve(); };
                tx.onerror = function(e) { reject(e.target.error); };
            });
        });
    }

    // 批量存入消息（从服务端拉取后）
    function putMessages(convKey, msgs) {
        if (!msgs || msgs.length === 0) return Promise.resolve();
        return open().then(function(db) {
            return new Promise(function(resolve, reject) {
                var tx = db.transaction('messages', 'readwrite');
                var store = tx.objectStore('messages');
                for (var i = 0; i < msgs.length; i++) {
                    var msg = msgs[i];
                    store.put({
                        conv_msg_id: makeKey(convKey, msg.id),
                        conv_key: convKey,
                        msg_id: msg.id,
                        data: msg,
                        created_at: msg.created_at || ''
                    });
                }
                tx.oncomplete = function() { resolve(); };
                tx.onerror = function(e) { reject(e.target.error); };
            });
        });
    }

    // 获取会话全部消息（按 msg_id 升序）
    function getMessages(convKey) {
        return open().then(function(db) {
            return new Promise(function(resolve, reject) {
                var tx = db.transaction('messages', 'readonly');
                var store = tx.objectStore('messages');
                var idx = store.index('conv_key');
                var msgs = [];
                idx.openCursor(IDBKeyRange.only(convKey)).onsuccess = function(e) {
                    var cursor = e.target.result;
                    if (cursor) {
                        msgs.push(cursor.value.data);
                        cursor.continue();
                    } else {
                        msgs.sort(function(a, b) { return a.id - b.id; });
                        resolve(msgs);
                    }
                };
                tx.onerror = function(e) { reject(e.target.error); };
            });
        });
    }

    // 获取会话最后 N 条消息
    function getRecentMessages(convKey, limit) {
        return open().then(function(db) {
            return new Promise(function(resolve, reject) {
                var tx = db.transaction('messages', 'readonly');
                var store = tx.objectStore('messages');
                var idx = store.index('conv_key');
                var msgs = [];
                // 逆序遍历取最新 N 条
                idx.openCursor(IDBKeyRange.only(convKey), 'prev').onsuccess = function(e) {
                    var cursor = e.target.result;
                    if (cursor && msgs.length < limit) {
                        msgs.push(cursor.value.data);
                        cursor.continue();
                    } else {
                        msgs.sort(function(a, b) { return a.id - b.id; });
                        resolve(msgs);
                    }
                };
            });
        });
    }

    // ========================================
    // 同步元数据
    // ========================================

    // 获取会话最后一条消息 ID
    function getLastMsgID(convKey) {
        return open().then(function(db) {
            return new Promise(function(resolve, reject) {
                var tx = db.transaction('meta', 'readonly');
                var store = tx.objectStore('meta');
                var req = store.get(convKey);
                req.onsuccess = function(e) {
                    var meta = e.target.result;
                    resolve(meta ? (meta.last_msg_id || 0) : 0);
                };
                req.onerror = function(e) { reject(e.target.error); };
            });
        });
    }

    // 更新会话最后一条消息 ID
    function setLastMsgID(convKey, msgId) {
        return open().then(function(db) {
            return new Promise(function(resolve, reject) {
                var tx = db.transaction('meta', 'readwrite');
                var store = tx.objectStore('meta');
                store.put({ conv_key: convKey, last_msg_id: msgId, last_sync_at: Date.now() });
                tx.oncomplete = function() { resolve(); };
                tx.onerror = function(e) { reject(e.target.error); };
            });
        });
    }

    // ========================================
    // 已读状态（单聊 _read 标记持久化）
    // ========================================

    // 更新多条消息的 _read 状态
    function markRead(convKey, msgIds) {
        if (!msgIds || msgIds.length === 0) return Promise.resolve();
        return open().then(function(db) {
            return new Promise(function(resolve, reject) {
                var tx = db.transaction('messages', 'readwrite');
                var store = tx.objectStore('messages');
                var count = 0;
                function next() { if (++count === msgIds.length) resolve(); }
                for (let i = 0; i < msgIds.length; i++) {
                    var key = makeKey(convKey, msgIds[i]);
                    var req = store.get(key);
                    req.onsuccess = function(e) {
                        var rec = e.target.result;
                        if (rec) {
                            rec.data._read = true;
                            store.put(rec);
                        }
                        next();
                    };
                    req.onerror = function() { next(); };
                }
            });
        });
    }

    // 删除会话所有消息（清理用）
    function clearConversation(convKey) {
        return open().then(function(db) {
            return new Promise(function(resolve, reject) {
                var tx = db.transaction('messages', 'readwrite');
                var store = tx.objectStore('messages');
                var idx = store.index('conv_key');
                var req = idx.openKeyCursor(IDBKeyRange.only(convKey));
                req.onsuccess = function(e) {
                    var cursor = e.target.result;
                    if (cursor) { store.delete(cursor.primaryKey); cursor.continue(); }
                };
                tx.oncomplete = function() { resolve(); };
                tx.onerror = function(e) { reject(e.target.error); };
            });
        });
    }

    // 获取所有有缓存的会话 key 列表（用于侧边栏初始化）
    function getCachedConvKeys() {
        return open().then(function(db) {
            return new Promise(function(resolve, reject) {
                var tx = db.transaction('meta', 'readonly');
                var store = tx.objectStore('meta');
                var keys = [];
                store.openCursor().onsuccess = function(e) {
                    var cursor = e.target.result;
                    if (cursor) { keys.push(cursor.value); cursor.continue(); }
                    else resolve(keys);
                };
            });
        });
    }

    return {
        open: open,
        putMessage: putMessage,
        putMessages: putMessages,
        getMessages: getMessages,
        getRecentMessages: getRecentMessages,
        getLastMsgID: getLastMsgID,
        setLastMsgID: setLastMsgID,
        markRead: markRead,
        clearConversation: clearConversation,
        getCachedConvKeys: getCachedConvKeys
    };
})();
