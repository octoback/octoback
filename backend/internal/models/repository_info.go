package models

import "handfree-work/octo-backup/internal/base/db_"

// RepositoryInfo 保存仓库的 Restic 快照和统计缓存。
type RepositoryInfo struct {
	db_.BaseModel
	RepositoryId *int64 `json:"repositoryId" gorm:"column:repository_id;uniqueIndex;not null"`
	Content      string `json:"-" gorm:"column:content;type:text;not null"`
}

// TableName 返回仓库缓存表名。
func (RepositoryInfo) TableName() string { return "repository_info" }
