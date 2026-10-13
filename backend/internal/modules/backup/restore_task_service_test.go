package logic

import (
	"context"
	"path/filepath"
	"testing"

	"handfree-work/octo-backup/internal/base/db_"
	"handfree-work/octo-backup/internal/models"
	"handfree-work/octo-backup/internal/svc"
)

func newRestoreTaskTestService(t *testing.T) *RestoreTaskService {
	t.Helper()
	database, err := db_.OpenSQLite(filepath.Join(t.TempDir(), "restore-tasks.db"))
	if err != nil {
		t.Fatal(err)
	}
	sqlDb, err := database.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDb.Close() })
	if err := database.AutoMigrate(&models.RestoreTask{}, &models.RestoreTaskContent{}, &models.PluginInstance{}); err != nil {
		t.Fatal(err)
	}
	return NewRestoreTaskService(context.Background(), &svc.ServiceContext{Db: database})
}

func TestRestoreTaskPageFiltersAndPaginates(t *testing.T) {
	service := newRestoreTaskTestService(t)
	for _, row := range []models.RestoreTask{
		{SnapshotId: "snapshot-a", TargetPath: "/restore/a", Status: "success"},
		{SnapshotId: "snapshot-b", TargetPath: "/restore/b", Status: "failed"},
	} {
		if err := service.svcCtx.Db.Create(&row).Error; err != nil {
			t.Fatal(err)
		}
	}

	result, err := service.Page(&RestoreTaskPageQuery{Offset: 0, Limit: 1, Status: "failed"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Total != 1 || len(result.Records) != 1 || result.Records[0]["snapshotId"] != "snapshot-b" || result.Records[0]["repositoryName"] != "" || result.Records[0]["targetAccessName"] != "" {
		t.Fatalf("分页筛选结果不正确: %#v", result)
	}
}

func TestRestoreTaskPresentIncludesTitleAndInstanceNames(t *testing.T) {
	service := newRestoreTaskTestService(t)
	row := &models.RestoreTask{Title: "业务数据 - 2026-10-12 10:20:30", RepositoryName: "生产仓库", TargetAccessName: "应用主机", SnapshotId: "snapshot-id"}
	presented := service.present(row)
	if presented["title"] != row.Title || presented["repositoryName"] != row.RepositoryName || presented["targetAccessName"] != row.TargetAccessName {
		t.Fatalf("任务展示信息缺失: %#v", presented)
	}
}

func TestRestoreTaskDeleteRejectsRunningAndRemovesContent(t *testing.T) {
	service := newRestoreTaskTestService(t)
	running := models.RestoreTask{SnapshotId: "running", TargetPath: "/restore", Status: "running"}
	completed := models.RestoreTask{SnapshotId: "completed", TargetPath: "/restore", Status: "success"}
	if err := service.svcCtx.Db.Create(&running).Error; err != nil {
		t.Fatal(err)
	}
	if err := service.svcCtx.Db.Create(&completed).Error; err != nil {
		t.Fatal(err)
	}
	if err := service.svcCtx.Db.Create(&models.RestoreTaskContent{RestoreTaskId: *completed.Id, Content: "output"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := service.Delete(*running.Id); err == nil {
		t.Fatal("运行中的恢复任务应拒绝删除")
	}
	if err := service.Delete(*completed.Id); err != nil {
		t.Fatal(err)
	}
	var taskCount, contentCount int64
	service.svcCtx.Db.Model(&models.RestoreTask{}).Where("id = ?", *completed.Id).Count(&taskCount)
	service.svcCtx.Db.Model(&models.RestoreTaskContent{}).Where("restore_task_id = ?", *completed.Id).Count(&contentCount)
	if taskCount != 0 || contentCount != 0 {
		t.Fatalf("删除未清理任务和输出: task=%d content=%d", taskCount, contentCount)
	}
}
