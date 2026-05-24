package model

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/mysql"
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
	Version         string
	Name            string
	Statements      []string
	MySQLStatements []string
}

func (m schemaMigration) statements(dialect string) []string {
	if dialect == "mysql" && len(m.MySQLStatements) > 0 {
		return m.MySQLStatements
	}
	return m.Statements
}

// InitDB initializes the configured database connection and runs migrations.
func InitDB(driver, dsn string) error {
	dialector, normalizedDriver, err := databaseDialector(driver, dsn)
	if err != nil {
		return err
	}

	DB, err = gorm.Open(dialector, &gorm.Config{
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

	if normalizedDriver == "sqlite" {
		// SQLite needs foreign keys enabled per connection.
		if err := DB.Exec("PRAGMA foreign_keys = ON").Error; err != nil {
			return err
		}
	}
	if err := EnsureIndexes(); err != nil {
		return err
	}

	log.Printf("[db] 数据库初始化完成，driver=%s，迁移已执行", normalizedDriver)
	return nil
}

func databaseDialector(driver, dsn string) (gorm.Dialector, string, error) {
	normalizedDriver := strings.ToLower(strings.TrimSpace(driver))
	if normalizedDriver == "" {
		normalizedDriver = "sqlite"
	}
	dsn = strings.TrimSpace(dsn)
	if dsn == "" {
		return nil, normalizedDriver, errors.New("database dsn is empty")
	}

	switch normalizedDriver {
	case "sqlite":
		return sqlite.Open(dsn), normalizedDriver, nil
	case "mysql":
		return mysql.Open(dsn), normalizedDriver, nil
	default:
		return nil, normalizedDriver, fmt.Errorf("unsupported database driver: %s", normalizedDriver)
	}
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
		if err := applySchemaMigration(migration, DB.Dialector.Name()); err != nil {
			return err
		}
	}
	return nil
}

func applySchemaMigration(migration schemaMigration, dialect string) error {
	var count int64
	if err := DB.Model(&SchemaMigration{}).
		Where("version = ?", migration.Version).
		Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	statements := migration.statements(dialect)
	if len(statements) == 0 {
		return nil
	}

	return DB.Transaction(func(tx *gorm.DB) error {
		for _, stmt := range statements {
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
			MySQLStatements: []string{
				"UPDATE message_reads SET conversation_type = 'user', conversation_id = peer_user_id WHERE conversation_type = '' AND peer_user_id IS NOT NULL",
				"UPDATE message_reads SET conversation_type = 'group', conversation_id = group_id WHERE conversation_type = '' AND group_id IS NOT NULL",
				"UPDATE message_reads mr JOIN (SELECT user_id, conversation_type, conversation_id, MAX(last_read_msg_id) AS max_read_id FROM message_reads WHERE conversation_type <> '' AND conversation_id > 0 GROUP BY user_id, conversation_type, conversation_id) agg ON agg.user_id = mr.user_id AND agg.conversation_type = mr.conversation_type AND agg.conversation_id = mr.conversation_id SET mr.last_read_msg_id = agg.max_read_id WHERE mr.conversation_type <> '' AND mr.conversation_id > 0",
				"DELETE mr FROM message_reads mr JOIN (SELECT MIN(id) AS keep_id, user_id, conversation_type, conversation_id FROM message_reads WHERE conversation_type <> '' AND conversation_id > 0 GROUP BY user_id, conversation_type, conversation_id) keepers ON keepers.user_id = mr.user_id AND keepers.conversation_type = mr.conversation_type AND keepers.conversation_id = mr.conversation_id WHERE mr.id <> keepers.keep_id AND mr.conversation_type <> '' AND mr.conversation_id > 0",
				"CREATE INDEX idx_messages_from_to_id ON messages (from_user_id, to_user_id, id)",
				"CREATE INDEX idx_messages_to_from_id ON messages (to_user_id, from_user_id, id)",
				"CREATE INDEX idx_messages_group_id_id ON messages (group_id, id)",
				"CREATE INDEX idx_messages_from_to_created ON messages (from_user_id, to_user_id, created_at)",
				"CREATE INDEX idx_messages_to_from_created ON messages (to_user_id, from_user_id, created_at)",
				"CREATE INDEX idx_messages_group_created ON messages (group_id, created_at)",
				"CREATE INDEX idx_message_reads_user_peer_group ON message_reads (user_id, peer_user_id, group_id)",
				"CREATE UNIQUE INDEX idx_message_reads_user_conversation ON message_reads (user_id, conversation_type, conversation_id)",
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
			MySQLStatements: []string{
				"UPDATE friend_relations fr JOIN (SELECT user_id, friend_id FROM friend_relations WHERE status = 'accepted' GROUP BY user_id, friend_id) accepted ON accepted.user_id = fr.user_id AND accepted.friend_id = fr.friend_id SET fr.status = 'accepted'",
				"DELETE fr FROM friend_relations fr JOIN (SELECT MIN(id) AS keep_id, user_id, friend_id FROM friend_relations GROUP BY user_id, friend_id) keepers ON keepers.user_id = fr.user_id AND keepers.friend_id = fr.friend_id WHERE fr.id <> keepers.keep_id",
				"UPDATE group_members gm JOIN `groups` g ON g.id = gm.group_id AND g.owner_id = gm.user_id SET gm.role = 'owner'",
				"UPDATE group_members gm JOIN (SELECT group_id, user_id FROM group_members WHERE role = 'admin' GROUP BY group_id, user_id) admins ON admins.group_id = gm.group_id AND admins.user_id = gm.user_id SET gm.role = 'admin' WHERE gm.role <> 'owner'",
				"DELETE gm FROM group_members gm JOIN (SELECT MIN(id) AS keep_id, group_id, user_id FROM group_members GROUP BY group_id, user_id) keepers ON keepers.group_id = gm.group_id AND keepers.user_id = gm.user_id WHERE gm.id <> keepers.keep_id",
				"DELETE cg FROM contact_groups cg JOIN (SELECT MIN(id) AS keep_id, user_id, name FROM contact_groups GROUP BY user_id, name) keepers ON keepers.user_id = cg.user_id AND keepers.name = cg.name WHERE cg.id <> keepers.keep_id",
				"CREATE UNIQUE INDEX idx_friend_relations_unique_pair ON friend_relations (user_id, friend_id)",
				"CREATE UNIQUE INDEX idx_group_members_unique_member ON group_members (group_id, user_id)",
				"CREATE UNIQUE INDEX idx_contact_groups_unique_name ON contact_groups (user_id, name)",
				"CREATE UNIQUE INDEX idx_ai_bot_knowledge_bases_unique_pair ON ai_bot_knowledge_bases (ai_bot_id, knowledge_base_id)",
				"CREATE INDEX idx_ai_bots_owner_status ON ai_bots (owner_id, status)",
				"CREATE INDEX idx_ai_bots_system_status ON ai_bots (is_system, status)",
				"CREATE INDEX idx_ai_knowledge_bases_owner_status ON ai_knowledge_bases (owner_id, status)",
				"CREATE INDEX idx_ai_knowledge_bases_system_status ON ai_knowledge_bases (is_system, status)",
				"CREATE INDEX idx_ai_knowledge_documents_kb_status ON ai_knowledge_documents (knowledge_base_id, status)",
				"CREATE INDEX idx_ai_token_usages_user_created ON ai_token_usages (user_id, created_at)",
				"CREATE INDEX idx_ai_token_usages_bot_created ON ai_token_usages (bot_id, created_at)",
			},
		},
		{
			Version: "2026052201",
			Name:    "message personal deletion visibility",
			Statements: []string{
				"CREATE UNIQUE INDEX IF NOT EXISTS idx_message_deletions_user_message ON message_deletions (user_id, message_id)",
				"CREATE INDEX IF NOT EXISTS idx_message_deletions_message_user ON message_deletions (message_id, user_id)",
			},
			MySQLStatements: []string{
				"CREATE INDEX idx_message_deletions_message_user ON message_deletions (message_id, user_id)",
			},
		},
	}
}
