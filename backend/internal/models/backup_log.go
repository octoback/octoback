package models

import "handfree-work/octo-backup/internal/base/db_"

type BackupLog struct {
	db_.BaseModel
	PlanId       int64   `json:"planId" gorm:"not null;index"`
	SourceId     *int64  `json:"sourceId" gorm:"index"`
	RepositoryId *int64  `json:"repositoryId" gorm:"index:idx_backup_log_repo_snapshot"`
	SnapshotId   *string `json:"snapshotId" gorm:"size:128;index:idx_backup_log_repo_snapshot"`
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

func (BackupLog) TableName() string { return "backup_log" }
