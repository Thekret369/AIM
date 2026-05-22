package model

import (
	"errors"
	"log"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// SchemaMigration 记录已经执行过的手写数据库迁移。
type SchemaMigration struct {
	Version   string    `gorm:"primaryKey;size:32" json:"version"`
	Name      string    `gorm:"size:128;not null" json:"name"`
	AppliedAt time.Time `gorm:"not null" json:"applied_at"`
}

type schemaMigration struct {
	Version    string
	Name       string
	Statements []string
}

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
		&MessageDeletion{},
		&AIBot{},
		&AIKnowledgeBase{},
		&AIKnowledgeDocument{},
		&AIBotKnowledgeBase{},
		&AITokenUsage{},
		&SchemaMigration{},
	)
	if err != nil {
		return err
	}

	// SQLite 需要手动开启外键约束
	if err := DB.Exec("PRAGMA foreign_keys = ON").Error; err != nil {
		return err
	}
	if err := EnsureIndexes(); err != nil {
		return err
	}

	log.Println("[db] 数据库初始化完成，迁移已执行")
	return nil
}

// EnsureIndexes 兼容旧调用入口，实际执行带版本记录的 schema migrations。
func EnsureIndexes() error {
	if DB == nil {
		return errors.New("database is not initialized")
	}
	if err := DB.AutoMigrate(&SchemaMigration{}, &MessageDeletion{}); err != nil {
		return err
	}
	for _, migration := range schemaMigrations() {
		if err := applySchemaMigration(migration); err != nil {
			return err
		}
	}
	return nil
}

func applySchemaMigration(migration schemaMigration) error {
	var count int64
	if err := DB.Model(&SchemaMigration{}).
		Where("version = ?", migration.Version).
		Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	return DB.Transaction(func(tx *gorm.DB) error {
		for _, stmt := range migration.Statements {
			if err := tx.Exec(stmt).Error; err != nil {
				return err
			}
		}
		return tx.Create(&SchemaMigration{
			Version:   migration.Version,
			Name:      migration.Name,
			AppliedAt: time.Now(),
		}).Error
	})
}

func schemaMigrations() []schemaMigration {
	return []schemaMigration{
		{
			Version: "2026052101",
			Name:    "message read watermarks and indexes",
			Statements: []string{
				"UPDATE message_reads SET conversation_type = 'user', conversation_id = peer_user_id WHERE conversation_type = '' AND peer_user_id IS NOT NULL",
				"UPDATE message_reads SET conversation_type = 'group', conversation_id = group_id WHERE conversation_type = '' AND group_id IS NOT NULL",
				"UPDATE message_reads SET last_read_msg_id = (SELECT MAX(mr2.last_read_msg_id) FROM message_reads mr2 WHERE mr2.user_id = message_reads.user_id AND mr2.conversation_type = message_reads.conversation_type AND mr2.conversation_id = message_reads.conversation_id) WHERE conversation_type <> '' AND conversation_id > 0",
				"DELETE FROM message_reads WHERE conversation_type <> '' AND conversation_id > 0 AND id NOT IN (SELECT MIN(id) FROM message_reads WHERE conversation_type <> '' AND conversation_id > 0 GROUP BY user_id, conversation_type, conversation_id)",
				"CREATE INDEX IF NOT EXISTS idx_messages_from_to_id ON messages (from_user_id, to_user_id, id)",
				"CREATE INDEX IF NOT EXISTS idx_messages_to_from_id ON messages (to_user_id, from_user_id, id)",
				"CREATE INDEX IF NOT EXISTS idx_messages_group_id_id ON messages (group_id, id)",
				"CREATE INDEX IF NOT EXISTS idx_messages_quote_message_id ON messages (quote_message_id)",
				"CREATE INDEX IF NOT EXISTS idx_messages_from_to_created ON messages (from_user_id, to_user_id, created_at)",
				"CREATE INDEX IF NOT EXISTS idx_messages_to_from_created ON messages (to_user_id, from_user_id, created_at)",
				"CREATE INDEX IF NOT EXISTS idx_messages_group_created ON messages (group_id, created_at)",
				"CREATE INDEX IF NOT EXISTS idx_message_reads_user_peer_group ON message_reads (user_id, peer_user_id, group_id)",
				"CREATE UNIQUE INDEX IF NOT EXISTS idx_message_reads_user_conversation ON message_reads (user_id, conversation_type, conversation_id)",
			},
		},
		{
			Version: "2026052102",
			Name:    "unique relationship constraints",
			Statements: []string{
				"UPDATE friend_relations SET status = 'accepted' WHERE EXISTS (SELECT 1 FROM friend_relations fr2 WHERE fr2.user_id = friend_relations.user_id AND fr2.friend_id = friend_relations.friend_id AND fr2.status = 'accepted')",
				"DELETE FROM friend_relations WHERE id NOT IN (SELECT MIN(id) FROM friend_relations GROUP BY user_id, friend_id)",
				"UPDATE group_members SET role = 'owner' WHERE EXISTS (SELECT 1 FROM groups WHERE groups.id = group_members.group_id AND groups.owner_id = group_members.user_id)",
				"UPDATE group_members SET role = 'admin' WHERE role <> 'owner' AND EXISTS (SELECT 1 FROM group_members gm2 WHERE gm2.group_id = group_members.group_id AND gm2.user_id = group_members.user_id AND gm2.role = 'admin')",
				"DELETE FROM group_members WHERE id NOT IN (SELECT MIN(id) FROM group_members GROUP BY group_id, user_id)",
				"DELETE FROM contact_groups WHERE id NOT IN (SELECT MIN(id) FROM contact_groups GROUP BY user_id, name)",
				"CREATE UNIQUE INDEX IF NOT EXISTS idx_friend_relations_unique_pair ON friend_relations (user_id, friend_id)",
				"CREATE UNIQUE INDEX IF NOT EXISTS idx_group_members_unique_member ON group_members (group_id, user_id)",
				"CREATE UNIQUE INDEX IF NOT EXISTS idx_contact_groups_unique_name ON contact_groups (user_id, name)",
				"CREATE UNIQUE INDEX IF NOT EXISTS idx_ai_bot_knowledge_bases_unique_pair ON ai_bot_knowledge_bases (ai_bot_id, knowledge_base_id)",
				"CREATE INDEX IF NOT EXISTS idx_ai_bots_owner_status ON ai_bots (owner_id, status)",
				"CREATE INDEX IF NOT EXISTS idx_ai_bots_system_status ON ai_bots (is_system, status)",
				"CREATE INDEX IF NOT EXISTS idx_ai_knowledge_bases_owner_status ON ai_knowledge_bases (owner_id, status)",
				"CREATE INDEX IF NOT EXISTS idx_ai_knowledge_bases_system_status ON ai_knowledge_bases (is_system, status)",
				"CREATE INDEX IF NOT EXISTS idx_ai_knowledge_documents_kb_status ON ai_knowledge_documents (knowledge_base_id, status)",
				"CREATE INDEX IF NOT EXISTS idx_ai_token_usages_user_created ON ai_token_usages (user_id, created_at)",
				"CREATE INDEX IF NOT EXISTS idx_ai_token_usages_bot_created ON ai_token_usages (bot_id, created_at)",
			},
		},
		{
			Version: "2026052201",
			Name:    "message personal deletion visibility",
			Statements: []string{
				"CREATE UNIQUE INDEX IF NOT EXISTS idx_message_deletions_user_message ON message_deletions (user_id, message_id)",
				"CREATE INDEX IF NOT EXISTS idx_message_deletions_message_user ON message_deletions (message_id, user_id)",
			},
		},
	}
}
