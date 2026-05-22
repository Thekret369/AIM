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

    // 按当前登录用户隔离本地缓存 key
    function currentUserScope() {
        try {
            if (typeof getUserId === 'function') {
                var userId = getUserId();
                if (userId > 0) return 'uid:' + userId;
            }
            if (typeof getUsername === 'function') {
                var username = getUsername();
                if (username) return 'username:' + encodeURIComponent(username);
            }
        } catch (e) {}
        return 'uid:0';
    }

    function scopeConvKey(convKey) {
        return currentUserScope() + ':' + convKey;
    }

    // 构造复合键
    function makeKey(scopedConvKey, msgId) {
        return scopedConvKey + '_' + msgId;
    }

    function buildMessageRecord(convKey, msg) {
        var scopedKey = scopeConvKey(convKey);
        return {
            conv_msg_id: makeKey(scopedKey, msg.id),
            conv_key: scopedKey,
            raw_conv_key: convKey,
            user_scope: currentUserScope(),
            msg_id: msg.id,
            data: msg,
            created_at: msg.created_at || ''
        };
    }

    function mergeMessageRecord(existing, incoming) {
        if (!existing || !existing.data) return incoming;
        if (existing.data._read && !incoming.data._read) {
            incoming.data._read = true;
        }
        return incoming;
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
                var scopedKey = scopeConvKey(convKey);
                var key = makeKey(scopedKey, msg.id);
                var req = store.get(key);
                req.onsuccess = function(e) {
                    var record = mergeMessageRecord(e.target.result, buildMessageRecord(convKey, msg));
                    store.put(record);
                };
                req.onerror = function(e) { reject(e.target.error); };
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
                var scopedKey = scopeConvKey(convKey);
                for (let i = 0; i < msgs.length; i++) {
                    let msg = msgs[i];
                    let req = store.get(makeKey(scopedKey, msg.id));
                    req.onsuccess = function(e) {
                        var record = mergeMessageRecord(e.target.result, buildMessageRecord(convKey, msg));
                        store.put(record);
                    };
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
                var scopedKey = scopeConvKey(convKey);
                idx.openCursor(IDBKeyRange.only(scopedKey)).onsuccess = function(e) {
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
                var scopedKey = scopeConvKey(convKey);
                // 逆序遍历取最新 N 条
                idx.openCursor(IDBKeyRange.only(scopedKey), 'prev').onsuccess = function(e) {
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
                var req = store.get(scopeConvKey(convKey));
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
                var scopedKey = scopeConvKey(convKey);
                var req = store.get(scopedKey);
                req.onsuccess = function(e) {
                    var meta = e.target.result || { conv_key: scopedKey };
                    meta.raw_conv_key = convKey;
                    meta.user_scope = currentUserScope();
                    var oldID = meta.last_msg_id || 0;
                    meta.last_msg_id = msgId > oldID ? msgId : oldID;
                    meta.last_sync_at = Date.now();
                    store.put(meta);
                };
                req.onerror = function(e) { reject(e.target.error); };
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
                var scopedKey = scopeConvKey(convKey);
                for (let i = 0; i < msgIds.length; i++) {
                    var key = makeKey(scopedKey, msgIds[i]);
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
                var scopedKey = scopeConvKey(convKey);
                var req = idx.openKeyCursor(IDBKeyRange.only(scopedKey));
                req.onsuccess = function(e) {
                    var cursor = e.target.result;
                    if (cursor) { store.delete(cursor.primaryKey); cursor.continue(); }
                };
                tx.oncomplete = function() { resolve(); };
                tx.onerror = function(e) { reject(e.target.error); };
            });
        });
    }

    // 删除当前用户本地缓存中的单条消息；服务端仍以个人软删除记录为准。
    function deleteMessage(convKey, msgId) {
        if (!msgId) return Promise.resolve();
        return open().then(function(db) {
            return new Promise(function(resolve, reject) {
                var tx = db.transaction('messages', 'readwrite');
                var store = tx.objectStore('messages');
                var scopedKey = scopeConvKey(convKey);
                store.delete(makeKey(scopedKey, msgId));
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
                var scope = currentUserScope();
                store.openCursor().onsuccess = function(e) {
                    var cursor = e.target.result;
                    if (cursor) {
                        if (cursor.value.user_scope === scope) {
                            keys.push({
                                conv_key: cursor.value.raw_conv_key || cursor.value.conv_key,
                                last_msg_id: cursor.value.last_msg_id || 0,
                                last_sync_at: cursor.value.last_sync_at || 0
                            });
                        }
                        cursor.continue();
                    }
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
        deleteMessage: deleteMessage,
        getCachedConvKeys: getCachedConvKeys
    };
})();
