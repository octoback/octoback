package models

import "handfree-work/octo-backup/internal/base/db_"

// BackupTaskLog 保存一条备份任务执行日志。
type BackupTaskLog struct {
	db_.BaseModel
	BackupTaskId int64  `json:"backupTaskId" gorm:"not null;index"`
	Level        string `json:"level" gorm:"size:20;not null"`
	Message      string `json:"message" gorm:"type:text;not null"`
	CreatedAt    int64  `json:"createdAt" gorm:"not null;index"`
}

func (BackupTaskLog) TableName() string { return "backup_task_log" }
