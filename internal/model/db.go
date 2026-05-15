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

	log.Println("[db] 数据库初始化完成，迁移已执行")
	return nil
}
