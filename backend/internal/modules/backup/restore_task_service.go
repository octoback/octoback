package logic

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"gorm.io/gorm"
	"handfree-work/octo-backup/internal/base/error_"
	"handfree-work/octo-backup/internal/models"
	backup "handfree-work/octo-backup/internal/modules/backup/restic"
	pluginservice "handfree-work/octo-backup/internal/modules/plugin/service"
	sshaccess "handfree-work/octo-backup/internal/plugins/access/ssh"
	sshsource "handfree-work/octo-backup/internal/plugins/source/ssh"
	"handfree-work/octo-backup/internal/svc"
)

// RestoreTaskInput 描述一次恢复任务的创建参数。
type RestoreTaskInput struct {
	RepositoryId   int64    `json:"repositoryId"`
	SnapshotId     string   `json:"snapshotId"`
	SourceId       int64    `json:"sourceId"`
	TargetAccessId int64    `json:"targetAccessId"`
	TargetPath     string   `json:"targetPath"`
	SelectedPaths  []string `json:"selectedPaths"`
}

// RestoreTaskService 管理独立的恢复任务记录和执行流程。
type RestoreTaskService struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// RestoreTaskPageQuery 描述恢复任务列表的分页和状态筛选条件。
type RestoreTaskPageQuery struct {
	Offset int64  `json:"offset"`
	Limit  int64  `json:"limit"`
	Status string `json:"status"`
}

// RestoreTaskPageResult 返回恢复任务分页记录。
type RestoreTaskPageResult struct {
	Offset  int64            `json:"offset"`
	Limit   int64            `json:"limit"`
	Records []map[string]any `json:"records"`
	Total   int64            `json:"total"`
}

// NewRestoreTaskService 创建恢复任务服务。
func NewRestoreTaskService(ctx context.Context, s *svc.ServiceContext) *RestoreTaskService {
	return &RestoreTaskService{ctx: ctx, svcCtx: s}
}

// Create 创建恢复任务并异步执行。
func (s *RestoreTaskService) Create(in *RestoreTaskInput) (map[string]any, error) {
	if in == nil || in.RepositoryId <= 0 || strings.TrimSpace(in.SnapshotId) == "" || strings.TrimSpace(in.TargetPath) == "" || len(in.SelectedPaths) == 0 {
		return nil, error_.NewTextError("恢复仓库、快照、目标主机、目标目录和恢复路径不能为空")
	}
	service := pluginservice.NewPluginInstanceService(s.ctx, s.svcCtx)
	sourceName := "恢复任务"
	if in.SourceId > 0 {
		source, err := service.Instance(in.SourceId)
		if err != nil {
			return nil, err
		}
		sourcePlugin, ok := source.(*sshsource.SshSource)
		if !ok {
			return nil, error_.NewTextError("恢复来源不是 SSH 来源")
		}
		var sourceRow models.PluginInstance
		if err := s.svcCtx.Db.First(&sourceRow, in.SourceId).Error; err != nil {
			return nil, error_.NewWrapError("查询恢复数据来源名称失败", err)
		}
		sourceName = sourceRow.Name
		if in.TargetAccessId <= 0 {
			in.TargetAccessId = int64(sourcePlugin.AccessId)
		}
	}
	if in.TargetAccessId <= 0 {
		return nil, error_.NewTextError("恢复目标主机不能为空")
	}
	access, err := service.Instance(in.TargetAccessId)
	if err != nil {
		return nil, err
	}
	if _, ok := access.(*sshaccess.SshAccess); !ok {
		return nil, error_.NewTextError("恢复目标不是 SSH 授权")
	}
	var repository models.PluginInstance
	if err := s.svcCtx.Db.First(&repository, in.RepositoryId).Error; err != nil {
		return nil, error_.NewWrapError("查询恢复仓库名称失败", err)
	}
	var target models.PluginInstance
	if err := s.svcCtx.Db.First(&target, in.TargetAccessId).Error; err != nil {
		return nil, error_.NewWrapError("查询恢复目标主机名称失败", err)
	}
	paths, err := json.Marshal(in.SelectedPaths)
	if err != nil {
		return nil, error_.NewWrapError("编码恢复路径失败", err)
	}
	createdAt := time.Now()
	title := sourceName + " - " + createdAt.Format("2006-01-02 15:04:05")
	task := &models.RestoreTask{Title: title, RepositoryId: in.RepositoryId, RepositoryName: repository.Name, SnapshotId: strings.TrimSpace(in.SnapshotId), SourceId: in.SourceId, TargetAccessId: in.TargetAccessId, TargetAccessName: target.Name, TargetPath: strings.TrimSpace(in.TargetPath), SelectedPaths: string(paths), Status: "queued", Stage: "排队中", StartedAt: createdAt.Unix()}
	if err := s.svcCtx.Db.Create(task).Error; err != nil {
		return nil, error_.NewWrapError("创建恢复任务失败", err)
	}
	go NewRestoreTaskService(context.Background(), s.svcCtx).execute(*task.Id)
	return map[string]any{"id": task.Id, "status": task.Status}, nil
}

func (s *RestoreTaskService) execute(id int64) {
	var task models.RestoreTask
	if s.svcCtx.Db.First(&task, id).Error != nil {
		return
	}
	update := func(status, stage, message string) {
		fields := map[string]any{"status": status, "stage": stage, "error": message}
		if status == "success" || status == "failed" {
			now := time.Now().Unix()
			fields["finished_at"] = now
			fields["progress"] = 100
		}
		_ = s.svcCtx.Db.Model(&models.RestoreTask{}).Where("id = ?", id).Updates(fields).Error
	}
	update("running", "准备恢复环境", "")
	service := pluginservice.NewPluginInstanceService(s.ctx, s.svcCtx)
	repositoryConfig, err := service.ResticRepositoryConfig(task.RepositoryId)
	if err != nil {
		update("failed", "加载仓库", err.Error())
		return
	}
	accessPlugin, err := service.Instance(task.TargetAccessId)
	if err != nil {
		update("failed", "加载目标主机", err.Error())
		return
	}
	targetAccess, ok := accessPlugin.(*sshaccess.SshAccess)
	if !ok {
		update("failed", "校验目标主机", "目标主机授权类型无效")
		return
	}
	var paths []string
	if json.Unmarshal([]byte(task.SelectedPaths), &paths) != nil {
		update("failed", "解析恢复路径", "恢复路径格式无效")
		return
	}
	repositoryConfig.Environment = backup.RemoteResticEnvironment
	repositoryConfig.ExecutionAccess = targetAccess
	repositoryConfig.Version = s.svcCtx.Restic.Version
	repositoryConfig.RemoteDirectory = ".octo_backup"
	client := backup.NewResticClient(s.svcCtx, repositoryConfig)
	output, err := client.Restore(s.ctx, backup.RestoreRequest{SnapshotId: task.SnapshotId, TargetPath: task.TargetPath, IncludePaths: paths})
	_ = s.svcCtx.Db.Model(&models.RestoreTaskContent{}).Where("restore_task_id = ?", id).Assign(map[string]any{"content": output}).FirstOrCreate(&models.RestoreTaskContent{RestoreTaskId: id, Content: output}).Error
	if err != nil {
		update("failed", "执行恢复", err.Error())
		return
	}
	update("success", "完成", "")
}

// Info 查询恢复任务详情。
func (s *RestoreTaskService) Info(id int64) (map[string]any, error) {
	var row models.RestoreTask
	if err := s.svcCtx.Db.First(&row, id).Error; err != nil {
		return nil, error_.NewWrapError("查询恢复任务失败", err)
	}
	return s.present(&row), nil
}

// LogContent 查询恢复任务保存的完整 Restic 输出。
func (s *RestoreTaskService) LogContent(id int64) (map[string]any, error) {
	var content models.RestoreTaskContent
	if err := s.svcCtx.Db.Where("restore_task_id = ?", id).First(&content).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return map[string]any{"id": id, "content": ""}, nil
		}
		return nil, error_.NewApiError(err)
	}
	return map[string]any{"id": id, "content": content.Content}, nil
}

// Page 分页查询恢复任务记录。
func (s *RestoreTaskService) Page(q *RestoreTaskPageQuery) (*RestoreTaskPageResult, error) {
	off, lim := int64(0), int64(20)
	if q != nil {
		off, lim = q.Offset, q.Limit
	}
	if off < 0 {
		off = 0
	}
	if lim <= 0 {
		lim = 20
	}
	if lim > 1000 {
		lim = 1000
	}
	db := s.svcCtx.Db.Model(&models.RestoreTask{})
	if q != nil && strings.TrimSpace(q.Status) != "" {
		db = db.Where("status = ?", strings.TrimSpace(q.Status))
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, error_.NewApiError(err)
	}
	var rows []models.RestoreTask
	if err := db.Order("id desc").Offset(int(off)).Limit(int(lim)).Find(&rows).Error; err != nil {
		return nil, error_.NewApiError(err)
	}
	records := make([]map[string]any, 0, len(rows))
	instanceIds := make([]int64, 0, len(rows)*3)
	for i := range rows {
		instanceIds = append(instanceIds, rows[i].RepositoryId, rows[i].TargetAccessId)
		if rows[i].SourceId > 0 {
			instanceIds = append(instanceIds, rows[i].SourceId)
		}
		records = append(records, s.present(&rows[i]))
	}
	if len(instanceIds) > 0 {
		var instances []models.PluginInstance
		if err := s.svcCtx.Db.Select("id, name").Where("id IN ?", instanceIds).Find(&instances).Error; err != nil {
			return nil, error_.NewWrapError("查询恢复任务关联名称失败", err)
		}
		names := make(map[int64]string, len(instances))
		for _, instance := range instances {
			if instance.Id != nil {
				names[*instance.Id] = instance.Name
			}
		}
		for i := range rows {
			if rows[i].RepositoryName == "" {
				records[i]["repositoryName"] = names[rows[i].RepositoryId]
			}
			if rows[i].TargetAccessName == "" {
				records[i]["targetAccessName"] = names[rows[i].TargetAccessId]
			}
			if rows[i].Title == "" {
				sourceName := names[rows[i].SourceId]
				if sourceName == "" {
					sourceName = "恢复任务"
				}
				records[i]["title"] = sourceName + " - " + time.Unix(rows[i].StartedAt, 0).Format("2006-01-02 15:04:05")
			}
		}
	}
	return &RestoreTaskPageResult{Offset: off, Limit: lim, Records: records, Total: total}, nil
}

// Delete 删除已结束的恢复任务及其输出日志。
func (s *RestoreTaskService) Delete(id int64) error {
	var row models.RestoreTask
	if err := s.svcCtx.Db.First(&row, id).Error; err != nil {
		return error_.NewWrapError("查询恢复任务失败", err)
	}
	if row.Status == "running" || row.Status == "queued" {
		return error_.NewTextError("运行中的恢复任务不能删除")
	}
	tx := s.svcCtx.Db.Begin()
	if err := tx.Where("restore_task_id = ?", id).Delete(&models.RestoreTaskContent{}).Error; err != nil {
		tx.Rollback()
		return error_.NewWrapError("删除恢复任务日志失败", err)
	}
	if err := tx.Delete(&models.RestoreTask{}, id).Error; err != nil {
		tx.Rollback()
		return error_.NewWrapError("删除恢复任务失败", err)
	}
	if err := tx.Commit().Error; err != nil {
		return error_.NewWrapError("提交删除恢复任务失败", err)
	}
	return nil
}

func (s *RestoreTaskService) present(row *models.RestoreTask) map[string]any {
	var finishedAt *int64
	if row.FinishedAt != nil {
		value := *row.FinishedAt * 1000
		finishedAt = &value
	}
	return map[string]any{"id": row.Id, "title": row.Title, "repositoryId": row.RepositoryId, "repositoryName": row.RepositoryName, "snapshotId": row.SnapshotId, "sourceId": row.SourceId, "targetAccessId": row.TargetAccessId, "targetAccessName": row.TargetAccessName, "targetPath": row.TargetPath, "status": row.Status, "progress": row.Progress, "stage": row.Stage, "result": row.Result, "error": row.Error, "startedAt": row.StartedAt * 1000, "finishedAt": finishedAt}
}
