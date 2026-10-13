package logic

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"handfree-work/octo-backup/internal/base/db_"
	"handfree-work/octo-backup/internal/base/error_"
	"handfree-work/octo-backup/internal/models"
	backup "handfree-work/octo-backup/internal/modules/backup/restic"
	"handfree-work/octo-backup/internal/modules/plugin"
	"handfree-work/octo-backup/internal/modules/plugin/secret"
	aliyunaccess "handfree-work/octo-backup/internal/plugins/access/aliyun"
	sshaccess "handfree-work/octo-backup/internal/plugins/access/ssh"
	aliossrepository "handfree-work/octo-backup/internal/plugins/repository/alioss"
	sftprepository "handfree-work/octo-backup/internal/plugins/repository/sftp"
	"handfree-work/octo-backup/internal/svc"
)

const pluginMasterKeySetting = "plugin.master-key.v1"

type PluginInstanceService struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	vars   map[string]any
	varId  int64
}

func (s *PluginInstanceService) GetVar(key string) any { return s.vars[key] }
func (s *PluginInstanceService) SetVar(key string, value any) error {
	if s.vars == nil {
		s.vars = map[string]any{}
	}
	s.vars[key] = value
	raw, err := yaml.Marshal(s.vars)
	if err != nil {
		return err
	}
	return s.svcCtx.Db.Model(&models.PluginInstance{}).Where("id = ?", s.varId).Update("vars", string(raw)).Error
}

func NewPluginInstanceService(ctx context.Context, s *svc.ServiceContext) *PluginInstanceService {
	return &PluginInstanceService{ctx: ctx, svcCtx: s}
}

type PluginInstanceInput struct {
	Name        string         `json:"name"`
	PluginType  string         `json:"pluginType"`
	PluginName  string         `json:"pluginName"`
	Config      map[string]any `json:"config"`
	Description string         `json:"description"`
}
type PluginInstancePageQuery struct {
	Offset     int64  `json:"offset"`
	Limit      int64  `json:"limit"`
	PluginType string `json:"pluginType"`
	PluginName string `json:"pluginName"`
	Name       string `json:"name"`
}
type PluginInstancePageResult struct {
	Offset  int64            `json:"offset"`
	Limit   int64            `json:"limit"`
	Records []map[string]any `json:"records"`
	Total   int64            `json:"total"`
}

func (s *PluginInstanceService) GetSimpleByIDs(ids []int64) ([]map[string]any, error) {
	if len(ids) == 0 {
		return []map[string]any{}, nil
	}
	dao := db_.New[models.PluginInstance](db_.NewCtx(s.ctx, s.svcCtx.Db))
	rows, err := dao.FindList(&models.PluginInstance{}, nil, func(db *gorm.DB) {
		db.Where("id IN ?", ids)
	})
	if err != nil {
		return nil, err
	}
	out := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		icon := ""
		if definition, ok := s.svcCtx.Plugins.Get(row.PluginName); ok {
			icon = definition.Metadata.Icon
		}
		out = append(out, map[string]any{"id": row.Id, "name": row.Name, "icon": icon, "pluginName": row.PluginName, "pluginType": row.PluginType})
	}
	return out, nil
}

func (s *PluginInstanceService) Update(id int64, in *PluginInstanceInput) (map[string]any, error) {
	row, err := db_.New[models.PluginInstance](db_.NewCtx(s.ctx, s.svcCtx.Db)).GetById(id)
	if err != nil || row == nil {
		return nil, error_.NewTextError("插件实例不存在")
	}
	if in == nil {
		return s.present(row), nil
	}
	if strings.TrimSpace(in.Name) != "" {
		row.Name = strings.TrimSpace(in.Name)
	}
	row.Description = strings.TrimSpace(in.Description)
	if in.Config != nil {
		definition, ok := s.svcCtx.Plugins.Get(row.PluginName)
		if !ok {
			return nil, error_.NewTextError("插件不存在")
		}
		codec, err := s.secretCodec()
		if err != nil {
			return nil, err
		}
		var existing map[string]any
		if row.ConfigYAML != "" {
			if existing, err = codec.DecodeYAML([]byte(row.ConfigYAML), secretFields(definition.Metadata.Fields)); err != nil {
				return nil, error_.NewTextError("解析已有插件配置失败")
			}
		}
		if row.PluginType == string(plugin.TypeRepository) {
			for _, field := range definition.Metadata.Fields {
				if !field.Encrypt {
					continue
				}
				password, exists := existing[field.Key].(string)
				if !exists || strings.TrimSpace(password) == "" {
					continue
				}
				if incoming, exists := in.Config[field.Key].(string); exists && strings.TrimSpace(incoming) != "" && incoming != maskSecret(password) {
					return nil, error_.NewTextError("存储仓库敏感配置设置后不可修改: %s", field.Title)
				}
				in.Config[field.Key] = maskSecret(password)
			}
		}
		config := mergePluginConfig(existing, in.Config, definition.Metadata.Fields)
		if err := validateRequiredPluginFields(config, definition.Metadata.Fields); err != nil {
			return nil, err
		}
		plain, e := yaml.Marshal(config)
		if e != nil {
			return nil, e
		}
		raw, _, e := codec.ProtectYAML(plain, secretFields(definition.Metadata.Fields), nil)
		if e != nil {
			return nil, e
		}
		row.ConfigYAML = string(raw)
	}
	if _, err = db_.New[models.PluginInstance](db_.NewCtx(s.ctx, s.svcCtx.Db)).UpdateById(id, row); err != nil {
		return nil, err
	}
	return s.present(row), nil
}

func (s *PluginInstanceService) Info(id int64) (map[string]any, error) {
	row, err := db_.New[models.PluginInstance](db_.NewCtx(s.ctx, s.svcCtx.Db)).GetById(id)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, error_.NewTextError("插件实例不存在")
	}
	return s.present(row), nil
}

// Snapshots 查询仓库快照。
func (s *PluginInstanceService) Snapshots(id int64) (any, error) {
	data, err := s.CachedRepositoryData(id)
	if err != nil {
		return nil, err
	}
	return s.attachSnapshotAssociations(id, data["snapshots"]), nil
}

// RepositoryStats 查询整个仓库的实际数据统计。
func (s *PluginInstanceService) RepositoryStats(id int64) (any, error) {
	data, err := s.CachedRepositoryData(id)
	if err != nil {
		return nil, err
	}
	return data["stats"], nil
}

// RefreshRepositoryData 查询 Restic 仓库并更新详情缓存。
func (s *PluginInstanceService) RefreshRepositoryData(id int64, summary map[string]any) (map[string]any, error) {
	client, err := s.repositoryClient(id)
	if err != nil {
		return nil, err
	}
	snapshots, err := client.Snapshots(s.ctx)
	if err != nil {
		return nil, error_.NewWrapError("刷新仓库快照缓存失败", err)
	}
	snapshots = s.attachSnapshotAssociations(id, snapshots)
	stats, err := client.Stats(s.ctx)
	if err != nil {
		return nil, error_.NewWrapError("刷新仓库统计缓存失败", err)
	}
	if summary == nil {
		cached, cacheErr := s.CachedRepositoryData(id)
		if cacheErr == nil {
			if cachedSummary, ok := cached["summary"].(map[string]any); ok {
				summary = cachedSummary
			}
		}
	}
	data := map[string]any{"snapshots": snapshots, "stats": stats, "summary": summary, "updatedAt": time.Now().UnixMilli()}
	raw, err := yaml.Marshal(data)
	if err != nil {
		return nil, error_.NewWrapError("序列化仓库详情缓存失败", err)
	}
	repositoryId := id
	row := &models.RepoDataInfo{RepositoryId: &repositoryId, DataYAML: string(raw)}
	if err := s.svcCtx.Db.Where("repository_id = ?", id).Assign(map[string]any{"data_yaml": string(raw)}).FirstOrCreate(row).Error; err != nil {
		return nil, error_.NewWrapError("保存仓库详情缓存失败", err)
	}
	return data, nil
}

// attachSnapshotAssociations 将备份记录中的业务关联补充到 Restic 快照元数据。
func (s *PluginInstanceService) attachSnapshotAssociations(repositoryId int64, value any) any {
	items, ok := value.([]any)
	if !ok {
		return value
	}
	associations := make(map[string]models.BackupLog)
	var logs []models.BackupLog
	if s.svcCtx.Db.Where("repository_id = ? AND snapshot_id <> ''", repositoryId).Find(&logs).Error == nil {
		for _, row := range logs {
			if row.SnapshotId != nil && strings.TrimSpace(*row.SnapshotId) != "" {
				associations[*row.SnapshotId] = row
			}
		}
	}
	for _, item := range items {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		snapshotId := strings.TrimSpace(fmt.Sprint(entry["id"]))
		if snapshotId == "" {
			snapshotId = strings.TrimSpace(fmt.Sprint(entry["snapshot_id"]))
		}
		entry["repositoryId"] = repositoryId
		entry["snapshotId"] = snapshotId
		if row, exists := associations[snapshotId]; exists {
			entry["planId"] = row.PlanId
			entry["sourceId"] = row.SourceId
			entry["associationMissing"] = false
		} else {
			entry["associationMissing"] = true
		}
	}
	return items
}

// CachedRepositoryData 读取仓库详情缓存，尚未刷新时返回空数据。
func (s *PluginInstanceService) CachedRepositoryData(id int64) (map[string]any, error) {
	var row models.RepoDataInfo
	if err := s.svcCtx.Db.Where("repository_id = ?", id).First(&row).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return map[string]any{"snapshots": []any{}, "stats": map[string]any{}}, nil
		}
		return nil, error_.NewWrapError("读取仓库详情缓存失败", err)
	}
	data := map[string]any{}
	if err := yaml.Unmarshal([]byte(row.DataYAML), &data); err != nil {
		return nil, error_.NewWrapError("解析仓库详情缓存失败", err)
	}
	return data, nil
}

func (s *PluginInstanceService) repositoryClient(id int64) (*backup.ResticClient, error) {
	clientConfig, err := s.ResticRepositoryConfig(id)
	if err != nil {
		return nil, err
	}
	clientConfig.Environment = backup.LocalResticEnvironment
	clientConfig.Version = s.svcCtx.Restic.Version
	return backup.NewResticClient(s.svcCtx, clientConfig), nil
}

// ResticRepositoryConfig 将仓库插件配置转换为 Restic 可执行配置。
func (s *PluginInstanceService) ResticRepositoryConfig(id int64) (backup.ClientConfig, error) {
	config, err := s.Config(id)
	if err != nil {
		return backup.ClientConfig{}, err
	}
	instance, err := s.Instance(id)
	if err != nil {
		return backup.ClientConfig{}, err
	}
	switch repository := instance.(type) {
	case *sftprepository.SftpRepository:
		access, err := s.resolveAccess(config["accessId"])
		if err != nil {
			return backup.ClientConfig{}, err
		}
		accessConfig, ok := access["config"].(map[string]any)
		if !ok {
			return backup.ClientConfig{}, error_.NewTextError("仓库 SSH 授权配置无效")
		}
		accessPlugin, err := s.svcCtx.Plugins.NewInstance(fmt.Sprint(access["pluginName"]), accessConfig)
		if err != nil {
			return backup.ClientConfig{}, err
		}
		sshConfig, ok := accessPlugin.(*sshaccess.SshAccess)
		if !ok {
			return backup.ClientConfig{}, error_.NewTextError("仓库 SSH 授权类型无效")
		}
		return backup.ClientConfig{RepositoryAccess: sshConfig, RepositoryPath: repository.Path, RepositoryPassword: repository.Password}, nil
	case *aliossrepository.AliOSSRepository:
		access, err := s.resolveAccess(config["accessId"])
		if err != nil {
			return backup.ClientConfig{}, err
		}
		accessConfig, ok := access["config"].(map[string]any)
		if !ok {
			return backup.ClientConfig{}, error_.NewTextError("阿里云授权配置无效")
		}
		accessPlugin, err := s.svcCtx.Plugins.NewInstance(fmt.Sprint(access["pluginName"]), accessConfig)
		if err != nil {
			return backup.ClientConfig{}, err
		}
		aliyunConfig, ok := accessPlugin.(*aliyunaccess.AliyunAccess)
		if !ok {
			return backup.ClientConfig{}, error_.NewTextError("阿里云授权类型无效")
		}
		accessKeyId, accessKeySecret, err := aliyunConfig.Credentials()
		if err != nil {
			return backup.ClientConfig{}, err
		}
		repositoryURL, password, environment, err := repository.RepositoryConfig(accessKeyId, accessKeySecret)
		if err != nil {
			return backup.ClientConfig{}, err
		}
		return backup.ClientConfig{RepositoryURL: repositoryURL, RepositoryPassword: password, RepositoryEnvironment: environment}, nil
	default:
		return backup.ClientConfig{}, error_.NewTextError("存储仓库插件不支持 Restic 配置")
	}
}

func (s *PluginInstanceService) CheckRepository(id int64) error {
	client, err := s.repositoryClient(id)
	if err != nil {
		return err
	}
	return client.Check(s.ctx)
}

// SnapshotBrowser 查询指定 Restic 快照中的文件条目。
func (s *PluginInstanceService) SnapshotBrowser(id int64, snapshotId string) (any, error) {
	clientConfig, err := s.ResticRepositoryConfig(id)
	if err != nil {
		return nil, err
	}
	clientConfig.Environment = backup.LocalResticEnvironment
	clientConfig.Version = s.svcCtx.Restic.Version
	client := backup.NewResticClient(s.svcCtx, clientConfig)
	return client.SnapshotBrowser(s.ctx, snapshotId)
}

// ForgetSnapshot 删除仓库中的指定 Restic 快照并刷新仓库缓存。
func (s *PluginInstanceService) ForgetSnapshot(id int64, snapshotId string) error {
	if strings.TrimSpace(snapshotId) == "" {
		return error_.NewTextError("删除仓库快照失败: 快照 Id 不能为空")
	}
	client, err := s.repositoryClient(id)
	if err != nil {
		return err
	}
	if err := client.ForgetSnapshot(s.ctx, snapshotId); err != nil {
		return error_.NewWrapError("删除仓库快照失败", err)
	}
	if _, err := s.RefreshRepositoryData(id, nil); err != nil {
		return error_.NewWrapError("删除仓库快照后刷新仓库数据失败", err)
	}
	return nil
}

// Config returns the decrypted runtime configuration for an instance.
func (s *PluginInstanceService) Config(id int64) (map[string]any, error) {
	row, err := db_.New[models.PluginInstance](db_.NewCtx(s.ctx, s.svcCtx.Db)).GetById(id)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, error_.NewTextError("插件实例不存在")
	}
	definition, ok := s.svcCtx.Plugins.Get(row.PluginName)
	if !ok {
		return nil, error_.NewTextError("插件不存在")
	}
	codec, err := s.secretCodec()
	if err != nil {
		return nil, err
	}
	config, err := codec.DecodeYAML([]byte(row.ConfigYAML), secretFields(definition.Metadata.Fields))
	if err != nil {
		return nil, error_.NewWrapError("解析插件配置失败", err)
	}
	return config, nil
}

// Instance loads an instance configuration and lets its registry definition
// construct the plugin object. Config remains a read-only configuration API.
func (s *PluginInstanceService) Instance(id int64) (plugin.PluginInstance, error) {
	row, err := db_.New[models.PluginInstance](db_.NewCtx(s.ctx, s.svcCtx.Db)).GetById(id)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, error_.NewTextError("插件实例不存在")
	}
	definition, ok := s.svcCtx.Plugins.Get(row.PluginName)
	if !ok {
		return nil, error_.NewTextError("插件不存在")
	}
	config, err := s.Config(id)
	if err != nil {
		return nil, err
	}
	return definition.NewPluginInstance(config)
}

func (s *PluginInstanceService) Metadata(t, name string) []plugin.Metadata {
	return s.svcCtx.Plugins.Metadata(plugin.PluginType(t), name)
}
func (s *PluginInstanceService) Create(in *PluginInstanceInput) (map[string]any, error) {
	if in == nil || strings.TrimSpace(in.Name) == "" || in.PluginName == "" {
		return nil, error_.NewTextError("插件名称和类型不能为空")
	}
	d, ok := s.svcCtx.Plugins.Get(in.PluginName)
	if !ok {
		return nil, error_.NewTextError("插件不存在")
	}
	if in.PluginType != "" && string(d.Metadata.Type) != in.PluginType {
		return nil, error_.NewTextError("插件类型不匹配")
	}
	config := mergePluginConfig(nil, in.Config, d.Metadata.Fields)
	if err := validateRequiredPluginFields(config, d.Metadata.Fields); err != nil {
		return nil, err
	}
	codec, err := s.secretCodec()
	if err != nil {
		return nil, err
	}
	plain, err := yaml.Marshal(config)
	if err != nil {
		return nil, err
	}
	raw, _, err := codec.ProtectYAML(plain, secretFields(d.Metadata.Fields), nil)
	if err != nil {
		return nil, err
	}
	row := &models.PluginInstance{Name: strings.TrimSpace(in.Name), PluginType: string(d.Metadata.Type), PluginName: in.PluginName, ConfigYAML: string(raw), Description: strings.TrimSpace(in.Description)}
	if err = db_.New[models.PluginInstance](db_.NewCtx(s.ctx, s.svcCtx.Db)).Create(row); err != nil {
		return nil, err
	}
	return s.present(row), nil
}
func (s *PluginInstanceService) Page(q *PluginInstancePageQuery) (*PluginInstancePageResult, error) {
	off, lim := int64(0), int64(20)
	typ, pluginName := "", ""
	if q != nil {
		off, lim, typ, pluginName = q.Offset, q.Limit, q.PluginType, q.PluginName
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
	dao := db_.New[models.PluginInstance](db_.NewCtx(s.ctx, s.svcCtx.Db))
	page := &db_.Page{Start: off, Limit: lim}
	query := &models.PluginInstance{}
	if typ != "" {
		query.PluginType = typ
	}
	if pluginName != "" {
		query.PluginName = pluginName
	}
	var filters []func(*gorm.DB)
	if q != nil && strings.TrimSpace(q.Name) != "" {
		name := strings.TrimSpace(q.Name)
		filters = append(filters, func(db *gorm.DB) { db.Where("name LIKE ?", "%"+name+"%") })
	}
	rows, err := dao.FindPage(&db_.PageReq[models.PluginInstance]{Query: query, Page: page}, filters...)
	if err != nil {
		return nil, err
	}
	out := []map[string]any{}
	for i := range *rows {
		row := &(*rows)[i]
		out = append(out, map[string]any{
			"id":          row.Id,
			"name":        row.Name,
			"pluginType":  row.PluginType,
			"pluginName":  row.PluginName,
			"description": row.Description,
			"createdAt":   row.CreatedAt,
			"updatedAt":   row.UpdatedAt,
		})
	}
	return &PluginInstancePageResult{Offset: off, Limit: lim, Records: out, Total: page.Total}, nil
}
func (s *PluginInstanceService) Action(id int64, action string, params map[string]any) (any, error) {
	row, err := db_.New[models.PluginInstance](db_.NewCtx(s.ctx, s.svcCtx.Db)).GetById(id)
	if err != nil || row == nil {
		return nil, error_.NewTextError("插件实例不存在")
	}
	definition, ok := s.svcCtx.Plugins.Get(row.PluginName)
	if !ok {
		return nil, error_.NewTextError("插件不存在")
	}
	if row.PluginType != string(definition.Metadata.Type) {
		return nil, error_.NewTextError("插件实例类型与插件定义不匹配")
	}
	codec, err := s.secretCodec()
	if err != nil {
		return nil, err
	}
	config, err := codec.DecodeYAML([]byte(row.ConfigYAML), secretFields(definition.Metadata.Fields))
	if err != nil {
		return nil, err
	}
	vars := map[string]any{}
	if strings.TrimSpace(row.Vars) != "" {
		if err := yaml.Unmarshal([]byte(row.Vars), &vars); err != nil {
			return nil, error_.NewWrapError("解析插件状态失败", err)
		}
	}
	s.vars, s.varId = vars, id
	pluginContext := &plugin.PluginContext{PluginService: s}
	instance, err := s.svcCtx.Plugins.NewInstance(row.PluginName, config, pluginContext)
	if err != nil {
		return nil, error_.NewWrapError(fmt.Sprintf("插件 %s 实例化失败", row.PluginName), err)
	}
	result, err := instance.ExecuteAction(s.ctx, action, params)
	if err != nil {
		return nil, error_.NewWrapError(fmt.Sprintf("插件 %s action %s 执行失败", row.PluginName, action), err)
	}
	return result, nil
}

func (s *PluginInstanceService) resolveAccess(value any) (map[string]any, error) {
	id, err := strconv.ParseInt(strings.TrimSpace(fmt.Sprint(value)), 10, 64)
	if err != nil || id <= 0 {
		return nil, error_.NewTextError("仓库授权 Id 无效")
	}
	row, err := db_.New[models.PluginInstance](db_.NewCtx(s.ctx, s.svcCtx.Db)).GetById(id)
	if err != nil {
		return nil, err
	}
	if row == nil || row.PluginType != string(plugin.TypeAccess) {
		return nil, error_.NewTextError("仓库授权不存在")
	}
	definition, ok := s.svcCtx.Plugins.Get(row.PluginName)
	if !ok || definition.Metadata.Type != plugin.TypeAccess {
		return nil, error_.NewTextError("授权插件不存在")
	}
	codec, err := s.secretCodec()
	if err != nil {
		return nil, err
	}
	config, err := codec.DecodeYAML([]byte(row.ConfigYAML), secretFields(definition.Metadata.Fields))
	if err != nil {
		return nil, err
	}
	return map[string]any{"id": row.Id, "name": row.Name, "pluginName": row.PluginName, "config": config}, nil
}

// ResolveAccess 根据授权 Id 返回解密后的授权实例配置，供插件 action 复用。
func (s *PluginInstanceService) ResolveAccess(value any) (map[string]any, error) {
	return s.resolveAccess(value)
}

func (s *PluginInstanceService) Delete(id int64) error {
	dao := db_.New[models.PluginInstance](db_.NewCtx(s.ctx, s.svcCtx.Db))
	row, err := dao.GetById(id)
	if err != nil {
		return err
	}
	if row == nil {
		return error_.NewTextError("插件实例不存在")
	}
	if row.PluginType == string(plugin.TypeAccess) {
		repositories, err := dao.FindList(&models.PluginInstance{PluginType: string(plugin.TypeRepository)}, nil)
		if err != nil {
			return err
		}
		for _, repository := range repositories {
			referenced, err := repositoryReferencesAccess(repository.ConfigYAML, id)
			if err != nil {
				return error_.NewTextError("检查仓库授权引用失败")
			}
			if referenced {
				return error_.NewTextError("授权仍被仓库引用")
			}
		}
	}
	_, err = dao.Delete(&id)
	return err
}
func (s *PluginInstanceService) present(row *models.PluginInstance) map[string]any {
	var fields []plugin.FieldSpec
	if definition, ok := s.svcCtx.Plugins.Get(row.PluginName); ok {
		fields = definition.Metadata.Fields
	}
	var decoded map[string]any
	if codec, err := s.secretCodec(); err == nil {
		decoded, _ = codec.DecodeYAML([]byte(row.ConfigYAML), secretFields(fields))
	}
	config := presentPluginConfig(row.ConfigYAML, fields, decoded)
	return map[string]any{"id": row.Id, "name": row.Name, "pluginType": row.PluginType, "pluginName": row.PluginName, "description": row.Description, "config": config, "createdAt": row.CreatedAt, "updatedAt": row.UpdatedAt}
}

func mergePluginConfig(existing, incoming map[string]any, fields []plugin.FieldSpec) map[string]any {
	merged := make(map[string]any, len(fields))
	for _, field := range fields {
		if value, ok := existing[field.Key]; ok {
			merged[field.Key] = value
		}
		value, ok := incoming[field.Key]
		if !ok {
			continue
		}
		if text, ok := value.(string); ok && text == "" {
			merged[field.Key] = ""
			continue
		}
		if field.Encrypt {
			if text, ok := value.(string); ok {
				if old, ok := existing[field.Key].(string); ok && text == maskSecret(old) {
					continue
				}
			}
		}
		merged[field.Key] = value
	}
	return merged
}

func validateRequiredPluginFields(config map[string]any, fields []plugin.FieldSpec) error {
	for _, field := range fields {
		if !field.Required {
			continue
		}
		value, ok := config[field.Key]
		if !ok || value == nil || strings.TrimSpace(fmt.Sprint(value)) == "" {
			return error_.NewTextError("%s不能为空", field.Title)
		}
	}
	return nil
}

func repositoryReferencesAccess(configYAML string, accessId int64) (bool, error) {
	var config map[string]any
	if err := yaml.Unmarshal([]byte(configYAML), &config); err != nil {
		return false, err
	}
	value, ok := config["accessId"]
	return ok && strings.TrimSpace(fmt.Sprint(value)) == fmt.Sprint(accessId), nil
}

func secretFields(fields []plugin.FieldSpec) secret.FieldMetadata {
	metadata := make(secret.FieldMetadata, len(fields))
	for _, field := range fields {
		metadata[field.Key] = secret.Field{Encrypt: field.Encrypt}
	}
	return metadata
}

func (s *PluginInstanceService) secretCodec() (*secret.Codec, error) {
	db := s.svcCtx.Db
	var setting models.SysSetting
	err := db.Where("key = ?", pluginMasterKeySetting).First(&setting).Error
	if err == nil {
		return codecFromSetting(setting.Setting)
	}
	if err != gorm.ErrRecordNotFound {
		return nil, err
	}

	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}
	encoded := base64.RawStdEncoding.EncodeToString(key)
	if err := db.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "key"}}, DoNothing: true}).Create(&models.SysSetting{Key: pluginMasterKeySetting, Setting: encoded}).Error; err != nil {
		return nil, err
	}
	if err := db.Where("key = ?", pluginMasterKeySetting).First(&setting).Error; err != nil {
		return nil, err
	}
	return codecFromSetting(setting.Setting)
}

func codecFromSetting(value string) (*secret.Codec, error) {
	key, err := base64.RawStdEncoding.DecodeString(value)
	if err != nil {
		return nil, error_.NewTextError("插件主密钥无效")
	}
	codec, err := secret.NewCodec(key)
	if err != nil {
		return nil, error_.NewTextError("初始化插件加密器失败")
	}
	return codec, nil
}

func presentPluginConfig(configYAML string, fields []plugin.FieldSpec, decoded map[string]any) map[string]any {
	config := map[string]any{"configured": false}
	var values map[string]any
	if configYAML == "" || yaml.Unmarshal([]byte(configYAML), &values) != nil {
		return config
	}
	if decoded == nil {
		decoded = values
	}
	for _, field := range fields {
		value, exists := decoded[field.Key]
		if !exists || value == nil || value == "" {
			continue
		}
		config["configured"] = true
		if field.Encrypt {
			if text, ok := value.(string); ok {
				config[field.Key] = maskSecret(text)
			}
			continue
		}
		config[field.Key] = value
	}
	return config
}

func maskSecret(value string) string {
	runes := []rune(value)
	if len(runes) <= 4 {
		return strings.Repeat("*", len(runes))
	}
	return string(runes[:2]) + strings.Repeat("*", len(runes)-4) + string(runes[len(runes)-2:])
}
