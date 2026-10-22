package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"strings"
	"time"

	"handfree-work/octo-backup/internal/base/db_"
	"handfree-work/octo-backup/internal/base/error_"
	"handfree-work/octo-backup/internal/base/log_"
	"handfree-work/octo-backup/internal/models"
	restic "handfree-work/octo-backup/internal/modules/backup/restic"
	"handfree-work/octo-backup/internal/modules/plugin"
	pluginservice "handfree-work/octo-backup/internal/modules/plugin/service"
	sshaccess "handfree-work/octo-backup/internal/plugins/access/ssh"
	sshsource "handfree-work/octo-backup/internal/plugins/source/ssh"
	"handfree-work/octo-backup/internal/svc"
)

type BackupExecuteService struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

type BackupExecuteResult struct {
	Id     int64  `json:"id"`
	Status string `json:"status"`
}

func NewBackupExecuteService(ctx context.Context, s *svc.ServiceContext) *BackupExecuteService {
	return &BackupExecuteService{ctx: ctx, svcCtx: s}
}

// backupTaskEntry 将日志级别和消息编码为可读的单行日志。
func backupTaskEntry(level string, message string) string {
	level = strings.ToUpper(strings.TrimSpace(level))
	if level != "WARN" && level != "ERROR" {
		level = "INFO"
	}
	message = strings.TrimSpace(message)
	return fmt.Sprintf("%s [%s] %s\n", time.Now().Format("2006-01-02 15:04:05.000"), level, message)
}

func serializeBackupSummary(summary map[string]any) (string, error) {
	if summary == nil {
		return "", nil
	}
	data, err := json.Marshal(summary)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func serializeBackupStats(stats map[string]any) (string, error) {
	return serializeBackupSummary(stats)
}

func appendBackupTask(db *gorm.DB, logId int64, level string, message string) {
	if db == nil || logId <= 0 || strings.TrimSpace(message) == "" {
		return
	}
	if err := db.Create(&models.BackupTaskLog{BackupTaskId: logId, Level: strings.ToUpper(strings.TrimSpace(level)), Message: strings.TrimSpace(message), CreatedAt: time.Now().UnixMilli()}).Error; err != nil {
		log_.Logger.Warn("创建备份任务日志失败", zap.Int64("backupTaskId", logId), zap.Error(err))
	}
}

func (s *BackupExecuteService) Execute(id int64, logId int64) (*BackupExecuteResult, error) {
	currentProgress := 0
	appendLog := func(level string, message string) {
		appendBackupTask(s.svcCtx.Db, logId, level, message)
	}
	updateLog := func(status string, nextProgress int, stage, result, message string) {
		if status == "failed" && nextProgress == 100 {
			nextProgress = currentProgress
		}
		currentProgress = nextProgress
		now := time.Now().Unix()
		fields := map[string]any{"status": status, "progress": nextProgress, "stage": stage, "result": result, "error": message}
		if status == "success" || status == "failed" {
			fields["finished_at"] = now
		}
		_ = s.svcCtx.Db.Model(&models.BackupTask{}).Where("id = ?", logId).Updates(fields).Error
		level := "INFO"
		logMessage := stage
		if message != "" {
			level = "ERROR"
			logMessage = stage + "：" + message
		}
		appendLog(level, logMessage)
	}
	saveSummary := func(summary map[string]any) {
		data, err := serializeBackupSummary(summary)
		if err != nil {
			appendLog("WARN", "序列化 Restic summary 失败："+err.Error())
			return
		}
		if data == "" {
			return
		}
		if err := s.svcCtx.Db.Model(&models.BackupTask{}).Where("id = ?", logId).Update("summary", data).Error; err != nil {
			appendLog("WARN", "保存 Restic summary 失败："+err.Error())
		}
	}
	saveStats := func(stats map[string]any) {
		data, err := serializeBackupStats(stats)
		if err != nil {
			appendLog("WARN", "序列化 Restic stats 失败："+err.Error())
			return
		}
		if data == "" {
			return
		}
		if err := s.svcCtx.Db.Model(&models.BackupTask{}).Where("id = ?", logId).Update("stats", data).Error; err != nil {
			appendLog("WARN", "保存 Restic stats 失败："+err.Error())
		}
	}
	updateLog("running", 5, "读取备份计划", "", "")
	log_.Logger.Info("读取备份计划", zap.Int64("planId", id))
	dao := db_.New[models.BackupPlan](db_.NewCtx(s.ctx, s.svcCtx.Db))
	plan, err := dao.GetById(id)
	if err != nil {
		updateLog("failed", 100, "读取备份计划", "", err.Error())
		return nil, error_.NewApiError(err)
	}
	if plan == nil {
		updateLog("failed", 100, "读取备份计划", "", "备份计划不存在")
		return nil, error_.NewTextError("备份计划不存在")
	}
	appendLog("INFO", fmt.Sprintf("备份计划读取完成：%s", plan.Name))
	var sourceInstance models.PluginInstance
	var repositoryInstance models.PluginInstance
	if result := s.svcCtx.Db.Select("name").First(&sourceInstance, plan.SourceId); result.Error == nil {
		if repositoryResult := s.svcCtx.Db.Select("name").First(&repositoryInstance, plan.RepositoryId); repositoryResult.Error == nil {
			appendLog("INFO", fmt.Sprintf("备份来源：%s；存储仓库：%s", sourceInstance.Name, repositoryInstance.Name))
		}
	}
	if !plan.Enabled {
		updateLog("failed", 100, "校验备份计划", "", "已禁用的备份计划不能执行")
		return nil, error_.NewTextError("已禁用的备份计划不能执行")
	}
	setStatus := func(status, message string) {
		plan.LastStatus, plan.LastError = status, message
		now := time.Now().Unix()
		plan.LastRunAt = &now
		_, _ = dao.UpdateById(id, plan)
		if message == "" {
			appendLog("INFO", "备份计划状态更新为 "+status)
			return
		}
		appendLog("ERROR", "备份计划状态更新为 "+status+"："+message)
	}
	setStatus("running", "")
	log_.Logger.Info("备份计划状态更新为 running", zap.Int64("planId", id))
	instance := pluginservice.NewPluginInstanceService(s.ctx, s.svcCtx)
	source, err := instance.Instance(plan.SourceId)
	if err != nil {
		appendLog("ERROR", "加载备份来源插件失败："+err.Error())
		setStatus("failed", err.Error())
		return nil, err
	}
	appendLog("INFO", "备份来源插件加载完成")
	result, err := s.doBackup(plan, source, func(progress int, stage string) {
		updateLog("running", progress, stage, "", "")
	}, appendLog)
	if err != nil {
		if result != nil {
			saveSummary(result.Summary)
		}
		updateLog("failed", 100, "执行备份", "", err.Error())
		setStatus("failed", err.Error())
		return nil, err
	}
	saveSummary(result.Summary)
	if result.SnapshotId == "" {
		message := "备份结果缺少快照 Id"
		updateLog("failed", 100, "校验备份结果", "", message)
		setStatus("failed", message)
		return nil, error_.NewTextError("%s", message)
	}
	saveStats(result.Stats)
	_ = s.svcCtx.Db.Model(&models.BackupTask{}).Where("id = ?", logId).Updates(map[string]any{"plan_id": plan.Id, "source_id": plan.SourceId, "repository_id": plan.RepositoryId, "snapshot_id": result.SnapshotId}).Error
	setStatus("success", "")
	updateLog("success", 100, "完成", "", "")
	return &BackupExecuteResult{Id: id, Status: result.Status}, nil
}

func (s *BackupExecuteService) doBackup(plan *models.BackupPlan, source plugin.PluginInstance, progress func(int, string), appendLog func(string, string)) (*restic.BackupResult, error) {
	appendLog("INFO", "开始加载备份来源配置")
	sourceConfig, ok := source.(*sshsource.SshSource)
	if !ok {
		return nil, error_.NewTextError("备份来源插件不支持备份配置")
	}
	instance := pluginservice.NewPluginInstanceService(s.ctx, s.svcCtx)
	appendLog("INFO", "开始加载存储仓库配置")
	repositoryConfig, err := instance.ResticRepositoryConfig(plan.RepositoryId)
	if err != nil {
		return nil, err
	}
	accessIdValue := int64(sourceConfig.AccessId)
	if accessIdValue <= 0 {
		return nil, error_.NewTextError("SSH 授权配置不存在")
	}
	accessPlugin, err := instance.Instance(accessIdValue)
	if err != nil {
		return nil, err
	}
	sshConfig, ok := accessPlugin.(*sshaccess.SshAccess)
	if !ok {
		return nil, error_.NewTextError("SSH 授权插件类型无效")
	}
	paths := restic.StringSlice(sourceConfig.Paths)
	excludePaths := restic.StringSlice(sourceConfig.ExcludePaths)
	appendLog("INFO", fmt.Sprintf("备份来源配置加载完成：%d 个路径，%d 个排除路径", len(paths), len(excludePaths)))
	binary, err := restic.LoadResticBinary()
	if err != nil {
		return nil, err
	}
	clientConfig := repositoryConfig
	clientConfig.Environment = restic.RemoteResticEnvironment
	clientConfig.ExecutionAccess = sshConfig
	clientConfig.Version = s.svcCtx.Restic.Version
	clientConfig.RemoteDirectory = ".octo_backup"
	clientConfig.ResticBinary = binary
	clientConfig.Compression = plan.Compression
	clientConfig.KeepPolicy = map[string]any{}
	clientConfig.Progress = progress
	clientConfig.Log = appendLog
	if plan.KeepPolicy != "" {
		if err := json.Unmarshal([]byte(plan.KeepPolicy), &clientConfig.KeepPolicy); err != nil {
			return nil, error_.NewWrapError("解析备份保留策略失败", err)
		}
	}
	backupRequest := restic.BackupRequest{
		RepositoryTag: plan.RepoTag,
		SourcePaths:   paths,
		ExcludePaths:  excludePaths,
	}
	client := restic.NewResticClient(s.svcCtx, clientConfig)
	appendLog("INFO", "开始执行 Restic 备份")
	result, err := client.Backup(s.ctx, backupRequest)
	if err != nil {
		return result, error_.NewWrapError("执行备份任务失败", err)
	}
	appendLog("INFO", "开始查询新快照统计")
	stats, statsErr := restic.NewResticClient(s.svcCtx, clientConfig).SnapshotStats(s.ctx, result.SnapshotId)
	if statsErr != nil {
		appendLog("WARN", "查询快照统计失败："+statsErr.Error())
	} else {
		result.Stats = stats
		appendLog("INFO", "快照统计查询完成")
	}
	if _, refreshErr := instance.RefreshRepositoryData(plan.RepositoryId); refreshErr != nil {
		appendLog("WARN", "备份完成后刷新仓库详情缓存失败："+refreshErr.Error())
		log_.Logger.Warn("备份完成后刷新仓库详情缓存失败", zap.Int64("repositoryId", plan.RepositoryId), zap.Error(refreshErr))
	} else {
		appendLog("INFO", "备份完成后仓库详情缓存刷新成功")
	}
	return result, nil
}
