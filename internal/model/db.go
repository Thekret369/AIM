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
	)
	if err != nil {
		return err
	}

	// SQLite 需要手动开启外键约束
	DB.Exec("PRAGMA foreign_keys = ON")
	if err := ensureIndexes(); err != nil {
		return err
	}

	log.Println("[db] 数据库初始化完成，迁移已执行")
	return nil
}

func ensureIndexes() error {
	statements := []string{
		"CREATE INDEX IF NOT EXISTS idx_messages_from_to_id ON messages (from_user_id, to_user_id, id)",
		"CREATE INDEX IF NOT EXISTS idx_messages_to_from_id ON messages (to_user_id, from_user_id, id)",
		"CREATE INDEX IF NOT EXISTS idx_messages_group_id_id ON messages (group_id, id)",
		"CREATE INDEX IF NOT EXISTS idx_messages_from_to_created ON messages (from_user_id, to_user_id, created_at)",
		"CREATE INDEX IF NOT EXISTS idx_messages_to_from_created ON messages (to_user_id, from_user_id, created_at)",
		"CREATE INDEX IF NOT EXISTS idx_messages_group_created ON messages (group_id, created_at)",
		"CREATE INDEX IF NOT EXISTS idx_message_reads_user_peer_group ON message_reads (user_id, peer_user_id, group_id)",
		"CREATE INDEX IF NOT EXISTS idx_group_members_group_user ON group_members (group_id, user_id)",
		"CREATE INDEX IF NOT EXISTS idx_friend_relations_user_friend ON friend_relations (user_id, friend_id)",
	}
	for _, stmt := range statements {
		if err := DB.Exec(stmt).Error; err != nil {
			return err
		}
	}
	return nil
}
