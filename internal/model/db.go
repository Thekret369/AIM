package model

import (
	"log"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// InitDB 初始化数据库连接并自动迁移
func InitDB(dsn string) error {
	var err error
	DB, err = gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return err
	}

	// 自动迁移所有模型
	err = DB.AutoMigrate(
		&User{},
		&Message{},
		&FriendRelation{},
		&ContactGroup{},
		&Group{},
		&GroupMember{},
		&Announcement{},
		&MessageRead{},
		&AIBot{},
	)
	if err != nil {
		return err
	}

	// SQLite 需要手动开启外键约束
	DB.Exec("PRAGMA foreign_keys = ON")
	if err := EnsureIndexes(); err != nil {
		return err
	}

	log.Println("[db] 数据库初始化完成，迁移已执行")
	return nil
}

func EnsureIndexes() error {
	statements := []string{
		"UPDATE message_reads SET conversation_type = 'user', conversation_id = peer_user_id WHERE conversation_type = '' AND peer_user_id IS NOT NULL",
		"UPDATE message_reads SET conversation_type = 'group', conversation_id = group_id WHERE conversation_type = '' AND group_id IS NOT NULL",
		"UPDATE message_reads SET last_read_msg_id = (SELECT MAX(mr2.last_read_msg_id) FROM message_reads mr2 WHERE mr2.user_id = message_reads.user_id AND mr2.conversation_type = message_reads.conversation_type AND mr2.conversation_id = message_reads.conversation_id) WHERE conversation_type <> '' AND conversation_id > 0",
		"DELETE FROM message_reads WHERE conversation_type <> '' AND conversation_id > 0 AND id NOT IN (SELECT MIN(id) FROM message_reads WHERE conversation_type <> '' AND conversation_id > 0 GROUP BY user_id, conversation_type, conversation_id)",
		"CREATE INDEX IF NOT EXISTS idx_messages_from_to_id ON messages (from_user_id, to_user_id, id)",
		"CREATE INDEX IF NOT EXISTS idx_messages_to_from_id ON messages (to_user_id, from_user_id, id)",
		"CREATE INDEX IF NOT EXISTS idx_messages_group_id_id ON messages (group_id, id)",
		"CREATE INDEX IF NOT EXISTS idx_messages_from_to_created ON messages (from_user_id, to_user_id, created_at)",
		"CREATE INDEX IF NOT EXISTS idx_messages_to_from_created ON messages (to_user_id, from_user_id, created_at)",
		"CREATE INDEX IF NOT EXISTS idx_messages_group_created ON messages (group_id, created_at)",
		"CREATE INDEX IF NOT EXISTS idx_message_reads_user_peer_group ON message_reads (user_id, peer_user_id, group_id)",
		"CREATE UNIQUE INDEX IF NOT EXISTS idx_message_reads_user_conversation ON message_reads (user_id, conversation_type, conversation_id)",
		"CREATE INDEX IF NOT EXISTS idx_group_members_group_user ON group_members (group_id, user_id)",
		"CREATE INDEX IF NOT EXISTS idx_friend_relations_user_friend ON friend_relations (user_id, friend_id)",
		"CREATE INDEX IF NOT EXISTS idx_ai_bots_owner_status ON ai_bots (owner_id, status)",
		"CREATE INDEX IF NOT EXISTS idx_ai_bots_system_status ON ai_bots (is_system, status)",
	}
	for _, stmt := range statements {
		if err := DB.Exec(stmt).Error; err != nil {
			return err
		}
	}
	return nil
}
