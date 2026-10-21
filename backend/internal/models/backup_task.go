package models

import "handfree-work/octo-backup/internal/base/db_"

// BackupTask 保存一次备份任务及其执行状态。
type BackupTask struct {
	db_.BaseModel
	PlanId       int64   `json:"planId" gorm:"not null;index"`
	SourceId     *int64  `json:"sourceId" gorm:"index"`
	RepositoryId *int64  `json:"repositoryId" gorm:"index:idx_backup_task_repo_snapshot"`
	SnapshotId   *string `json:"snapshotId" gorm:"size:128;index:idx_backup_task_repo_snapshot"`
	Status       string  `json:"status" gorm:"size:30;not null;index"`
	Progress     int     `json:"progress" gorm:"not null;default:0"`
	Stage        string  `json:"stage" gorm:"size:100"`
	Result       string  `json:"result" gorm:"type:text"`
	Summary      string  `json:"summary" gorm:"type:longtext"`
	Stats        string  `json:"stats" gorm:"type:longtext"`
	Error        string  `json:"error" gorm:"type:text"`
	StartedAt    int64   `json:"startedAt"`
	FinishedAt   *int64  `json:"finishedAt"`
}

func (BackupTask) TableName() string { return "backup_task" }
