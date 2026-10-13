package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"strings"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
	"handfree-work/octo-backup/internal/base/db_"
	"handfree-work/octo-backup/internal/base/error_"
	"handfree-work/octo-backup/internal/base/log_"
	"handfree-work/octo-backup/internal/models"
	cronservice "handfree-work/octo-backup/internal/modules/cron"
	"handfree-work/octo-backup/internal/modules/plugin"
	"handfree-work/octo-backup/internal/svc"
)

type BackupPlanService struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewBackupPlanService(ctx context.Context, s *svc.ServiceContext) *BackupPlanService {
	return &BackupPlanService{ctx: ctx, svcCtx: s}
}

type BackupPlanInput struct {
	Name         string         `json:"name"`
	SourceId     int64          `json:"sourceId"`
	RepositoryId int64          `json:"repositoryId"`
	RepoTag      string         `json:"repoTag"`
	Compression  string         `json:"compression"`
	KeepPolicy   map[string]any `json:"keepPolicy"`
	Schedule     string         `json:"schedule"`
	Enabled      *bool          `json:"enabled"`
}
type BackupPlanPageQuery struct {
	Offset int64  `json:"offset"`
	Limit  int64  `json:"limit"`
	Name   string `json:"name"`
}
type BackupPlanPageResult struct {
	Offset  int64            `json:"offset"`
	Limit   int64            `json:"limit"`
	Records []map[string]any `json:"records"`
	Total   int64            `json:"total"`
}

// BackupPlanSimple 表示备份计划下拉选项。
type BackupPlanSimple struct {
	Id   int64  `json:"id"`
	Name string `json:"name"`
}
type BackupLogPageQuery struct {
	Offset int64 `json:"offset"`
	Limit  int64 `json:"limit"`
	PlanId int64 `json:"planId"`
}
type BackupLogPageResult struct {
	Offset  int64            `json:"offset"`
	Limit   int64            `json:"limit"`
	Records []map[string]any `json:"records"`
	Total   int64            `json:"total"`
}

func (s *BackupPlanService) validate(in *BackupPlanInput) error {
	if in == nil || strings.TrimSpace(in.Name) == "" || in.SourceId <= 0 || in.RepositoryId <= 0 || strings.TrimSpace(in.Schedule) == "" {
		return error_.NewTextError("备份计划名称、来源、仓库和调度不能为空")
	}
	if err := cronservice.ValidateSchedule(strings.TrimSpace(in.Schedule)); err != nil {
		return err
	}
	normalized, err := normalizeRepoTag(in.RepoTag)
	if err != nil {
		return err
	}
	in.RepoTag = normalized
	if in.Compression == "" {
		in.Compression = "auto"
	}
	if in.Compression != "auto" && in.Compression != "off" && in.Compression != "max" {
		return error_.NewTextError("压缩级别必须是 auto、off 或 max")
	}
	if in.KeepPolicy == nil {
		in.KeepPolicy = map[string]any{"mode": "none"}
	}
	policyData, err := json.Marshal(in.KeepPolicy)
	if err != nil {
		return error_.NewWrapError("编码备份保留策略失败", err)
	}
	if len(policyData) == 0 {
		return error_.NewTextError("备份保留策略不能为空")
	}
	dao := db_.New[models.PluginInstance](db_.NewCtx(s.ctx, s.svcCtx.Db))
	source, err := dao.GetById(in.SourceId)
	if err != nil {
		return error_.NewApiError(err)
	}
	if source == nil || source.PluginType != string(plugin.TypeSource) {
		return error_.NewTextError("备份来源不存在: %d", in.SourceId)
	}
	repo, err := dao.GetById(in.RepositoryId)
	if err != nil {
		return error_.NewApiError(err)
	}
	if repo == nil || repo.PluginType != string(plugin.TypeRepository) {
		return error_.NewTextError("存储仓库不存在: %d", in.RepositoryId)
	}
	return nil
}

// List 返回全部备份计划的轻量选项数据。
func (s *BackupPlanService) List() ([]BackupPlanSimple, error) {
	var rows []BackupPlanSimple
	if err := s.svcCtx.Db.Model(&models.BackupPlan{}).Select("id, name").Order("name asc, id asc").Find(&rows).Error; err != nil {
		return nil, error_.NewApiError(err)
	}
	if rows == nil {
		rows = make([]BackupPlanSimple, 0)
	}
	return rows, nil
}

func normalizeRepoTag(value string) (string, error) {
	value = strings.TrimSpace(strings.ReplaceAll(value, "\\", "/"))
	if value == "" {
		return "", nil
	}
	if strings.HasPrefix(value, "/") || strings.Contains(value, ":") {
		return "", error_.NewTextError("仓库目录必须是相对路径")
	}
	cleaned := path.Clean(value)
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return "", error_.NewTextError("仓库目录不能越界")
	}
	return strings.Trim(cleaned, "/"), nil
}

func (s *BackupPlanService) checkRepoTag(id, repositoryId int64, tag string) error {
	var count int64
	db := s.svcCtx.Db.Model(&models.BackupPlan{}).Where("repository_id = ? AND repo_tag = ?", repositoryId, tag)
	if id > 0 {
		db = db.Where("id <> ?", id)
	}
	if err := db.Count(&count).Error; err != nil {
		return error_.NewTextError("检查仓库标签是否重复失败: %v", err)
	}
	if count > 0 {
		return error_.NewTextError("同一存储仓库下的标签已被其他备份计划使用")
	}
	return nil
}
func (s *BackupPlanService) Create(in *BackupPlanInput) (map[string]any, error) {
	if err := s.validate(in); err != nil {
		return nil, err
	}
	if err := s.checkRepoTag(0, in.RepositoryId, in.RepoTag); err != nil {
		return nil, err
	}
	enabled := true
	if in.Enabled != nil {
		enabled = *in.Enabled
	}
	policyData, _ := json.Marshal(in.KeepPolicy)
	row := &models.BackupPlan{Name: strings.TrimSpace(in.Name), SourceId: in.SourceId, RepositoryId: in.RepositoryId, RepoTag: in.RepoTag, Compression: in.Compression, KeepPolicy: string(policyData), Schedule: strings.TrimSpace(in.Schedule), Enabled: enabled, LastStatus: "pending"}
	if err := db_.New[models.BackupPlan](db_.NewCtx(s.ctx, s.svcCtx.Db)).Create(row); err != nil {
		return nil, err
	}
	if err := s.registerPlan(row); err != nil {
		return nil, error_.NewWrapError("备份计划已保存但注册定时任务失败", err)
	}
	return s.present(row), nil
}
func (s *BackupPlanService) Update(id int64, in *BackupPlanInput) (map[string]any, error) {
	dao := db_.New[models.BackupPlan](db_.NewCtx(s.ctx, s.svcCtx.Db))
	row, err := dao.GetById(id)
	if err != nil {
		return nil, error_.NewApiError(err)
	}
	if row == nil {
		return nil, error_.NewTextError("备份计划不存在")
	}
	if err := s.validate(in); err != nil {
		return nil, err
	}
	if err := s.checkRepoTag(id, in.RepositoryId, in.RepoTag); err != nil {
		return nil, err
	}
	row.Name, row.SourceId, row.RepositoryId, row.RepoTag, row.Schedule = strings.TrimSpace(in.Name), in.SourceId, in.RepositoryId, in.RepoTag, strings.TrimSpace(in.Schedule)
	row.Compression = in.Compression
	policyData, _ := json.Marshal(in.KeepPolicy)
	row.KeepPolicy = string(policyData)
	if in.Enabled != nil {
		row.Enabled = *in.Enabled
	}
	if _, err = dao.UpdateById(id, row); err != nil {
		return nil, err
	}
	if err := s.registerPlan(row); err != nil {
		return nil, error_.NewWrapError("备份计划已更新但同步定时任务失败", err)
	}
	return s.present(row), nil
}
func (s *BackupPlanService) Info(id int64) (map[string]any, error) {
	row, err := db_.New[models.BackupPlan](db_.NewCtx(s.ctx, s.svcCtx.Db)).GetById(id)
	if err != nil {
		return nil, error_.NewApiError(err)
	}
	if row == nil {
		return nil, error_.NewTextError("备份计划不存在")
	}
	return s.present(row), nil
}
func (s *BackupPlanService) Delete(id int64) error {
	dao := db_.New[models.BackupPlan](db_.NewCtx(s.ctx, s.svcCtx.Db))
	n, err := dao.Delete(&id)
	if err != nil {
		return err
	}
	if n == 0 {
		return error_.NewTextError("备份计划不存在")
	}
	cronservice.Unregister("backup_plan", id)
	return nil
}

func (s *BackupPlanService) registerPlan(plan *models.BackupPlan) error {
	if plan == nil || plan.Id == nil {
		return error_.NewTextError("备份计划 Id 为空")
	}
	planId := *plan.Id
	if !plan.Enabled {
		cronservice.Unregister("backup_plan", planId)
		return nil
	}
	return cronservice.Register("backup_plan", planId, plan.Schedule, func() {
		log_.Logger.Info("cron 触发备份计划", zap.Int64("planId", planId))
		if _, err := NewBackupPlanService(context.Background(), s.svcCtx).Run(planId); err != nil {
			log_.Logger.Error("cron 触发备份计划失败", zap.Int64("planId", planId), zap.Error(err))
		}
	})
}

// RegisterAll 注册数据库中所有启用的备份计划。
func (s *BackupPlanService) RegisterAll() error {
	var plans []models.BackupPlan
	log_.Logger.Info("开始批量注册备份计划定时任务")
	if err := s.svcCtx.Db.Where("enabled = ?", true).Find(&plans).Error; err != nil {
		return error_.NewWrapError("加载备份计划定时任务失败", err)
	}
	for i := range plans {
		if err := s.registerPlan(&plans[i]); err != nil {
			return err
		}
	}
	log_.Logger.Info("备份计划定时任务批量注册完成", zap.Int("count", len(plans)))
	return nil
}

func (s *BackupPlanService) Run(id int64) (map[string]any, error) {
	now := time.Now().Unix()
	plan, err := db_.New[models.BackupPlan](db_.NewCtx(s.ctx, s.svcCtx.Db)).GetById(id)
	if err != nil || plan == nil {
		return nil, error_.NewTextError("备份计划不存在: %d", id)
	}
	sourceId := plan.SourceId
	repositoryId := plan.RepositoryId
	row := &models.BackupLog{PlanId: id, SourceId: &sourceId, RepositoryId: &repositoryId, Status: "queued", Progress: 0, Stage: "排队中", StartedAt: now}
	if err := db_.New[models.BackupLog](db_.NewCtx(s.ctx, s.svcCtx.Db)).Create(row); err != nil {
		return nil, err
	}
	appendBackupLog(s.svcCtx.Db, *row.Id, "INFO", fmt.Sprintf("cron 触发备份计划，已创建运行记录：%d", *row.Id))
	go NewBackupExecuteService(context.Background(), s.svcCtx).Execute(id, *row.Id)
	return map[string]any{"logId": row.Id, "status": row.Status}, nil
}

func (s *BackupPlanService) Log(id int64) (map[string]any, error) {
	row, err := db_.New[models.BackupLog](db_.NewCtx(s.ctx, s.svcCtx.Db)).GetById(id)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, error_.NewTextError("备份运行记录不存在")
	}
	var finishedAt *int64
	if row.FinishedAt != nil {
		value := *row.FinishedAt * 1000
		finishedAt = &value
	}
	return map[string]any{"id": row.Id, "planId": row.PlanId, "status": row.Status, "progress": row.Progress, "stage": row.Stage, "result": row.Result, "error": row.Error, "startedAt": row.StartedAt * 1000, "finishedAt": finishedAt}, nil
}

// LogContent 查询指定运行记录保存的完整 Restic 输出日志。
func (s *BackupPlanService) LogContent(id int64) (map[string]any, error) {
	var content models.BackupLogContent
	var logRow models.BackupLog
	if err := s.svcCtx.Db.First(&logRow, id).Error; err != nil {
		return nil, error_.NewApiError(err)
	}
	contentResult := s.svcCtx.Db.Where("backup_log_id = ?", id).First(&content)
	if contentResult.Error != nil && contentResult.Error != gorm.ErrRecordNotFound {
		return nil, error_.NewApiError(contentResult.Error)
	}
	var summary any
	if strings.TrimSpace(logRow.Summary) != "" {
		if err := json.Unmarshal([]byte(logRow.Summary), &summary); err != nil {
			return nil, error_.NewWrapError("解析 Restic summary 失败", err)
		}
	}
	return map[string]any{"id": id, "content": content.Content, "summary": summary}, nil
}
func (s *BackupPlanService) LogPage(q *BackupLogPageQuery) (*BackupLogPageResult, error) {
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
	db := s.svcCtx.Db.Table("backup_log l").Select("l.id, l.plan_id, p.name AS plan_name, p.repository_id, l.status, l.progress, l.stage, l.result, l.error, l.started_at, l.finished_at").Joins("LEFT JOIN backup_plan p ON p.id = l.plan_id")
	if q != nil && q.PlanId > 0 {
		db = db.Where("l.plan_id = ?", q.PlanId)
	}
	var total int64
	if err := db.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, error_.NewApiError(err)
	}
	var rows []struct {
		Id                   *int64 `gorm:"column:id"`
		PlanId               int64  `gorm:"column:plan_id"`
		PlanName             string `gorm:"column:plan_name"`
		RepositoryId         int64  `gorm:"column:repository_id"`
		Status               string
		Progress             int
		Stage, Result, Error string
		StartedAt            int64  `gorm:"column:started_at"`
		FinishedAt           *int64 `gorm:"column:finished_at"`
	}
	if err := db.Order("l.id desc").Offset(int(off)).Limit(int(lim)).Scan(&rows).Error; err != nil {
		return nil, error_.NewApiError(err)
	}
	records := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		var finishedAt *int64
		if row.FinishedAt != nil {
			value := *row.FinishedAt * 1000
			finishedAt = &value
		}
		records = append(records, map[string]any{"id": row.Id, "planId": row.PlanId, "planName": row.PlanName, "repositoryId": row.RepositoryId, "status": row.Status, "progress": row.Progress, "stage": row.Stage, "result": row.Result, "error": row.Error, "startedAt": row.StartedAt * 1000, "finishedAt": finishedAt})
	}
	return &BackupLogPageResult{Offset: off, Limit: lim, Records: records, Total: total}, nil
}
func (s *BackupPlanService) Page(q *BackupPlanPageQuery) (*BackupPlanPageResult, error) {
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
	page := &db_.Page{Start: off, Limit: lim}
	filters := []func(*gorm.DB){}
	if q != nil && strings.TrimSpace(q.Name) != "" {
		name := strings.TrimSpace(q.Name)
		filters = append(filters, func(db *gorm.DB) { db.Where("name LIKE ?", "%"+name+"%") })
	}
	rows, err := db_.New[models.BackupPlan](db_.NewCtx(s.ctx, s.svcCtx.Db)).FindPage(&db_.PageReq[models.BackupPlan]{Query: &models.BackupPlan{}, Page: page}, filters...)
	if err != nil {
		return nil, err
	}
	out := make([]map[string]any, 0, len(*rows))
	for i := range *rows {
		out = append(out, s.present(&(*rows)[i]))
	}
	return &BackupPlanPageResult{Offset: off, Limit: lim, Records: out, Total: page.Total}, nil
}
func (s *BackupPlanService) present(row *models.BackupPlan) map[string]any {
	var policy map[string]any
	if row.KeepPolicy != "" {
		_ = json.Unmarshal([]byte(row.KeepPolicy), &policy)
	}
	return map[string]any{"id": row.Id, "name": row.Name, "sourceId": row.SourceId, "repositoryId": row.RepositoryId, "repoTag": row.RepoTag, "keepPolicy": policy, "compression": row.Compression, "schedule": row.Schedule, "enabled": row.Enabled, "lastStatus": row.LastStatus, "lastError": row.LastError, "lastRunAt": row.LastRunAt, "createdAt": row.CreatedAt, "updatedAt": row.UpdatedAt}
}
