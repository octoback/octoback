package logic

import (
	"context"
	"encoding/json"
	"handfree-work/octo-backup/internal/base/db_"
	"handfree-work/octo-backup/internal/models"
	"handfree-work/octo-backup/internal/svc"
	"path/filepath"
	"strings"
	"testing"
)

func TestPlanSimpleList(t *testing.T) {
	database, err := db_.OpenSQLite(filepath.Join(t.TempDir(), "plans.db"))
	if err != nil {
		t.Fatal(err)
	}
	sqlDb, err := database.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDb.Close() })
	if err := database.AutoMigrate(&models.BackupPlan{}); err != nil {
		t.Fatal(err)
	}
	service := NewBackupPlanService(context.Background(), &svc.ServiceContext{Db: database})
	empty, err := service.List()
	if err != nil || empty == nil || len(empty) != 0 {
		t.Fatalf("空列表: %v %v", empty, err)
	}
	for _, name := range []string{"计划甲", "计划乙"} {
		row := models.BackupPlan{Name: name, RepoTag: name}
		if err := database.Create(&row).Error; err != nil {
			t.Fatal(err)
		}
	}
	rows, err := service.List()
	if err != nil || len(rows) != 2 {
		t.Fatalf("全部计划: %v %v", rows, err)
	}
	if rows[0].Name != "计划乙" || rows[0].Id == 0 {
		t.Fatalf("选项内容错误: %v", rows)
	}
}

func TestNormalizeRepoTag(t *testing.T) {
	tests := []struct {
		input string
		want  string
		valid bool
	}{
		{"team/app", "team/app", true},
		{" team\\app/ ", "team/app", true},
		{"", "", true},
		{"../other", "", false},
		{"a/../../other", "", false},
		{"/absolute", "", false},
		{"C:\\repo", "", false},
	}
	for _, tt := range tests {
		got, err := normalizeRepoTag(tt.input)
		if (err == nil) != tt.valid || got != tt.want {
			t.Errorf("normalizeRepoTag(%q) = %q, %v; want %q valid=%v", tt.input, got, err, tt.want, tt.valid)
		}
	}
}

func TestLogContentReturnsReadableContentAndSummary(t *testing.T) {
	database, err := db_.OpenSQLite(filepath.Join(t.TempDir(), "backup-log-content.db"))
	if err != nil {
		t.Fatal(err)
	}
	sqlDb, err := database.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDb.Close() })
	if err := database.AutoMigrate(&models.BackupLog{}, &models.BackupLogContent{}); err != nil {
		t.Fatal(err)
	}
	logRow := &models.BackupLog{PlanId: 1, Status: "success", Summary: `{"snapshot_id":"abc123","total_files_processed":8}`}
	if err := database.Create(logRow).Error; err != nil {
		t.Fatal(err)
	}
	logContent := &models.BackupLogContent{BackupLogId: *logRow.Id, Content: "[INFO] cron 触发\n[INFO] 执行完成\n"}
	if err := database.Create(logContent).Error; err != nil {
		t.Fatal(err)
	}
	service := NewBackupPlanService(context.Background(), &svc.ServiceContext{Db: database})
	detail, err := service.LogContent(*logRow.Id)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(detail["content"].(string), "cron 触发") {
		t.Fatalf("content = %v", detail["content"])
	}
	encoded, err := json.Marshal(detail["summary"])
	if err != nil {
		t.Fatal(err)
	}
	var summary map[string]any
	if err := json.Unmarshal(encoded, &summary); err != nil {
		t.Fatal(err)
	}
	if summary["snapshot_id"] != "abc123" {
		t.Fatalf("summary = %v", summary)
	}
}
