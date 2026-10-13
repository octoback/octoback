package models

import "handfree-work/octo-backup/internal/base/db_"

// BackupLogContent 保存一次备份执行产生的完整可读日志。
type BackupLogContent struct {
	db_.BaseModel
	BackupLogId int64  `json:"backupLogId" gorm:"not null;uniqueIndex"`
	Content     string `json:"content" gorm:"type:longtext;not null"`
}

// TableName 返回备份日志内容表名。
func (BackupLogContent) TableName() string { return "backup_log_content" }
