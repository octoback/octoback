package models

import "handfree-work/octo-backup/internal/base/db_"

// RestoreTask 保存一次独立的恢复任务及其执行状态。
type RestoreTask struct {
	db_.BaseModel
	Title            string `json:"title" gorm:"size:200;not null;default:''"`
	RepositoryId     int64  `json:"repositoryId" gorm:"not null;index"`
	RepositoryName   string `json:"repositoryName" gorm:"size:100;not null;default:''"`
	SnapshotId       string `json:"snapshotId" gorm:"size:128;not null;index:idx_restore_task_repo_snapshot"`
	SourceId         int64  `json:"sourceId" gorm:"index"`
	TargetAccessId   int64  `json:"targetAccessId" gorm:"not null;index"`
	TargetAccessName string `json:"targetAccessName" gorm:"size:100;not null;default:''"`
	TargetPath       string `json:"targetPath" gorm:"size:2000;not null"`
	SelectedPaths    string `json:"selectedPaths" gorm:"type:text;not null"`
	Status           string `json:"status" gorm:"size:30;not null;index"`
	Progress         int    `json:"progress" gorm:"not null;default:0"`
	Stage            string `json:"stage" gorm:"size:100"`
	Result           string `json:"result" gorm:"type:text"`
	Error            string `json:"error" gorm:"type:text"`
	StartedAt        int64  `json:"startedAt"`
	FinishedAt       *int64 `json:"finishedAt"`
}

func (RestoreTask) TableName() string { return "restore_task" }

// RestoreTaskContent 保存恢复任务产生的完整命令输出。
type RestoreTaskContent struct {
	db_.BaseModel
	RestoreTaskId int64  `json:"restoreTaskId" gorm:"not null;uniqueIndex"`
	Content       string `json:"content" gorm:"type:text"`
}

func (RestoreTaskContent) TableName() string { return "restore_task_content" }
