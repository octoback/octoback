package backup

import (
	"strconv"

	"handfree-work/octo-backup/internal/base/error_"
	"handfree-work/octo-backup/internal/base/error_/code_"
	"handfree-work/octo-backup/internal/base/web_"
	logic "handfree-work/octo-backup/internal/modules/plugin/service"
	"handfree-work/octo-backup/internal/svc"

	"github.com/gofiber/fiber/v3"
)

// RegisterRepository 注册仓库数据和快照操作接口。
func RegisterRepository(app *fiber.App, svcCtx *svc.ServiceContext) {
	routes := app.Group("/api/plugin/instance")
	web_.RegisterRoutes(routes, svcCtx.Auth,
		web_.Route{Path: "/snapshots", Permission: web_.Read, Handler: web_.Handle(pluginInstanceSnapshots(svcCtx))},
		web_.Route{Path: "/stats", Permission: web_.Read, Handler: web_.Handle(pluginInstanceStats(svcCtx))},
		web_.Route{Path: "/snapshot-stats", Permission: web_.Read, Handler: web_.Handle(pluginInstanceSnapshotStats(svcCtx))},
		web_.Route{Path: "/refresh-repository-data", Permission: web_.Write, Handler: web_.Handle(pluginInstanceRefreshRepositoryData(svcCtx))},
		web_.Route{Path: "/check", Permission: web_.Write, Handler: web_.Handle(pluginInstanceCheck(svcCtx))},
		web_.Route{Path: "/snapshot-browser", Permission: web_.Read, Handler: web_.Handle(pluginInstanceSnapshotBrowser(svcCtx))},
		web_.Route{Path: "/forget-snapshot", Permission: web_.Write, Handler: web_.Handle(pluginInstanceForgetSnapshot(svcCtx))},
	)
}

func parseRepositoryId(c fiber.Ctx) (int64, error) {
	repositoryId, err := strconv.ParseInt(c.Query("id"), 10, 64)
	if err != nil {
		return 0, error_.NewCodeTextError(code_.ParamError, "仓库插件实例 Id 无效")
	}
	return repositoryId, nil
}

// pluginInstanceSnapshots godoc
// @Summary 查询仓库快照
// @Description 返回 Restic 快照及已存在的备份日志关联；不存在的备份 Summary 不会返回。
// @Tags 插件
// @Accept json
// @Produce json
// @Security bearerAuth
// @Param id query int64 true "仓库插件实例 Id"
// @Success 200 {object} map[string]interface{}
// @Router /api/plugin/instance/snapshots [post]
func pluginInstanceSnapshots(svcCtx *svc.ServiceContext) func(fiber.Ctx) (any, error) {
	return func(c fiber.Ctx) (any, error) {
		repositoryId, err := parseRepositoryId(c)
		if err != nil {
			return nil, err
		}
		return logic.NewPluginInstanceService(c.Context(), svcCtx).Snapshots(repositoryId)
	}
}

// pluginInstanceStats godoc
// @Summary 查询仓库统计
// @Description 返回 repository_info 中缓存的仓库统计；缓存缺失时会先刷新仓库数据。
// @Tags 插件
// @Accept json
// @Produce json
// @Security bearerAuth
// @Param id query int64 true "仓库插件实例 Id"
// @Success 200 {object} map[string]interface{}
// @Router /api/plugin/instance/stats [post]
func pluginInstanceStats(svcCtx *svc.ServiceContext) func(fiber.Ctx) (any, error) {
	return func(c fiber.Ctx) (any, error) {
		repositoryId, err := parseRepositoryId(c)
		if err != nil {
			return nil, err
		}
		return logic.NewPluginInstanceService(c.Context(), svcCtx).RepositoryStats(repositoryId)
	}
}

// pluginInstanceSnapshotStats godoc
// @Summary 查询单个快照统计
// @Description 优先读取备份日志缓存；没有统计时查询 Restic 并写入对应备份日志。
// @Tags 插件
// @Accept json
// @Produce json
// @Security bearerAuth
// @Param id query int64 true "仓库插件实例 Id"
// @Param snapshotId query string true "Restic 快照 Id"
// @Success 200 {object} map[string]interface{}
// @Router /api/plugin/instance/snapshot-stats [post]
func pluginInstanceSnapshotStats(svcCtx *svc.ServiceContext) func(fiber.Ctx) (any, error) {
	return func(c fiber.Ctx) (any, error) {
		repositoryId, err := parseRepositoryId(c)
		if err != nil {
			return nil, err
		}
		snapshotId := c.Query("snapshotId")
		return logic.NewPluginInstanceService(c.Context(), svcCtx).SnapshotStats(repositoryId, snapshotId)
	}
}

// pluginInstanceRefreshRepositoryData godoc
// @Summary 刷新仓库快照缓存
// @Description 查询 Restic 快照和仓库统计并写入 repository_info，不查询备份 Summary。
// @Tags 插件
// @Accept json
// @Produce json
// @Security bearerAuth
// @Param id query int64 true "仓库插件实例 Id"
// @Success 200 {object} map[string]interface{}
// @Router /api/plugin/instance/refresh-repository-data [post]
func pluginInstanceRefreshRepositoryData(svcCtx *svc.ServiceContext) func(fiber.Ctx) (any, error) {
	return func(c fiber.Ctx) (any, error) {
		repositoryId, err := parseRepositoryId(c)
		if err != nil {
			return nil, err
		}
		return logic.NewPluginInstanceService(c.Context(), svcCtx).RefreshRepositoryData(repositoryId)
	}
}

func pluginInstanceCheck(svcCtx *svc.ServiceContext) func(fiber.Ctx) (any, error) {
	return func(c fiber.Ctx) (any, error) {
		repositoryId, err := parseRepositoryId(c)
		if err != nil {
			return nil, err
		}
		return logic.NewPluginInstanceService(c.Context(), svcCtx).StartCheckRepository(repositoryId)
	}
}

func pluginInstanceSnapshotBrowser(svcCtx *svc.ServiceContext) func(fiber.Ctx) (any, error) {
	return func(c fiber.Ctx) (any, error) {
		repositoryId, err := parseRepositoryId(c)
		if err != nil {
			return nil, err
		}
		snapshotId := c.Query("snapshotId")
		if snapshotId == "" {
			return nil, error_.NewCodeTextError(code_.ParamError, "快照 Id 不能为空")
		}
		return logic.NewPluginInstanceService(c.Context(), svcCtx).SnapshotBrowser(repositoryId, snapshotId)
	}
}

// pluginInstanceForgetSnapshot godoc
// @Summary 删除仓库中的指定快照
// @Tags 插件
// @Accept json
// @Produce json
// @Security bearerAuth
// @Param id query int64 true "仓库插件实例 Id"
// @Param snapshotId query string true "Restic 快照 Id"
// @Success 200 {object} map[string]interface{}
// @Router /api/plugin/instance/forget-snapshot [post]
func pluginInstanceForgetSnapshot(svcCtx *svc.ServiceContext) func(fiber.Ctx) (fiber.Map, error) {
	return func(c fiber.Ctx) (fiber.Map, error) {
		repositoryId, err := parseRepositoryId(c)
		if err != nil {
			return nil, err
		}
		snapshotId := c.Query("snapshotId")
		if err := logic.NewPluginInstanceService(c.Context(), svcCtx).ForgetSnapshot(repositoryId, snapshotId); err != nil {
			return nil, err
		}
		return fiber.Map{}, nil
	}
}
