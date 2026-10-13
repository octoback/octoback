package sys

import (
	"fmt"
	"handfree-work/octo-backup/internal/base/error_"
	"handfree-work/octo-backup/internal/base/error_/code_"
	"handfree-work/octo-backup/internal/base/web_"
	logic "handfree-work/octo-backup/internal/modules/plugin/service"
	"handfree-work/octo-backup/internal/svc"
	"strconv"

	"github.com/gofiber/fiber/v3"
)

func RegisterPlugin(app *fiber.App, s *svc.ServiceContext) {
	r := app.Group("/api/plugin")
	web_.RegisterRoutes(r, s.Auth,
		web_.Route{Path: "/metadata", Permission: web_.Read, Handler: web_.HandleJSON(pluginMetadata(s))},
	)
	instances := app.Group("/api/plugin/instance")
	web_.RegisterRoutes(instances, s.Auth,
		web_.Route{Path: "/page", Permission: web_.Read, Handler: web_.HandleJSON(pluginInstancePage(s))},
		web_.Route{Path: "/info", Permission: web_.Read, Handler: web_.Handle(pluginInstanceInfo(s))},
		web_.Route{Path: "/snapshots", Permission: web_.Read, Handler: web_.Handle(pluginInstanceSnapshots(s))},
		web_.Route{Path: "/stats", Permission: web_.Read, Handler: web_.Handle(pluginInstanceStats(s))},
		web_.Route{Path: "/refresh-repository-data", Permission: web_.Write, Handler: web_.Handle(pluginInstanceRefreshRepositoryData(s))},
		web_.Route{Path: "/check", Permission: web_.Write, Handler: web_.Handle(pluginInstanceCheck(s))},
		web_.Route{Path: "/snapshot-browser", Permission: web_.Read, Handler: web_.Handle(pluginInstanceSnapshotBrowser(s))},
		web_.Route{Path: "/forget-snapshot", Permission: web_.Write, Handler: web_.Handle(pluginInstanceForgetSnapshot(s))},
		web_.Route{Path: "/simpleByIds", Permission: web_.Read, Handler: web_.HandleJSON(pluginInstanceSimpleByIDs(s))},
		web_.Route{Path: "/create", Permission: web_.Write, Handler: web_.HandleJSON(pluginInstanceCreate(s))},
		web_.Route{Path: "/update", Permission: web_.Write, Handler: web_.HandleJSON(pluginInstanceUpdate(s))},
		web_.Route{Path: "/action", Permission: web_.Write, Handler: web_.HandleJSON(pluginInstanceAction(s))},
		web_.Route{Path: "/delete", Permission: web_.Admin, Handler: web_.Handle(pluginInstanceDelete(s))},
	)
}
func pluginInstanceRefreshRepositoryData(s *svc.ServiceContext) func(fiber.Ctx) (any, error) {
	return func(c fiber.Ctx) (any, error) {
		id, err := strconv.ParseInt(c.Query("id"), 10, 64)
		if err != nil {
			return nil, error_.NewCodeTextError(code_.ParamError, "插件 Id 无效")
		}
		return logic.NewPluginInstanceService(c.Context(), s).RefreshRepositoryData(id, nil)
	}
}
func pluginInstanceSnapshots(s *svc.ServiceContext) func(fiber.Ctx) (any, error) {
	return func(c fiber.Ctx) (any, error) {
		id, err := strconv.ParseInt(c.Query("id"), 10, 64)
		if err != nil {
			return nil, error_.NewCodeTextError(code_.ParamError, "插件 Id 无效")
		}
		return logic.NewPluginInstanceService(c.Context(), s).Snapshots(id)
	}
}
func pluginInstanceStats(s *svc.ServiceContext) func(fiber.Ctx) (any, error) {
	return func(c fiber.Ctx) (any, error) {
		id, err := strconv.ParseInt(c.Query("id"), 10, 64)
		if err != nil {
			return nil, error_.NewCodeTextError(code_.ParamError, "插件 Id 无效")
		}
		return logic.NewPluginInstanceService(c.Context(), s).RepositoryStats(id)
	}
}
func pluginInstanceCheck(s *svc.ServiceContext) func(fiber.Ctx) (any, error) {
	return func(c fiber.Ctx) (any, error) {
		id, err := strconv.ParseInt(c.Query("id"), 10, 64)
		if err != nil {
			return nil, error_.NewCodeTextError(code_.ParamError, "插件 Id 无效")
		}
		if err := logic.NewPluginInstanceService(c.Context(), s).CheckRepository(id); err != nil {
			return nil, err
		}
		return fiber.Map{}, nil
	}
}
func pluginInstanceSnapshotBrowser(s *svc.ServiceContext) func(fiber.Ctx) (any, error) {
	return func(c fiber.Ctx) (any, error) {
		id, err := strconv.ParseInt(c.Query("id"), 10, 64)
		if err != nil {
			return nil, error_.NewCodeTextError(code_.ParamError, "插件 Id 无效")
		}
		snapshotId := c.Query("snapshotId")
		if snapshotId == "" {
			return nil, error_.NewCodeTextError(code_.ParamError, "快照 Id 不能为空")
		}
		return logic.NewPluginInstanceService(c.Context(), s).SnapshotBrowser(id, snapshotId)
	}
}

func pluginInstanceForgetSnapshot(s *svc.ServiceContext) func(fiber.Ctx) (any, error) {
	return func(c fiber.Ctx) (any, error) {
		id, err := strconv.ParseInt(c.Query("id"), 10, 64)
		if err != nil {
			return nil, error_.NewCodeTextError(code_.ParamError, "插件 Id 无效")
		}
		snapshotId := c.Query("snapshotId")
		if err := logic.NewPluginInstanceService(c.Context(), s).ForgetSnapshot(id, snapshotId); err != nil {
			return nil, err
		}
		return fiber.Map{}, nil
	}
}

// pluginInstanceInfo godoc
// @Summary 获取插件实例详情
// @Tags 插件
// @Accept json
// @Produce json
// @Security bearerAuth
// @Param id query int64 true "插件 Id"
// @Success 200 {object} map[string]interface{}
// @Router /api/plugin/instance/info [post]
func pluginInstanceInfo(s *svc.ServiceContext) func(fiber.Ctx) (any, error) {
	return func(c fiber.Ctx) (any, error) {
		id, err := strconv.ParseInt(c.Query("id"), 10, 64)
		if err != nil {
			return nil, error_.NewCodeTextError(code_.ParamError, "插件 Id 无效")
		}
		return logic.NewPluginInstanceService(c.Context(), s).Info(id)
	}
}

// pluginInstanceSimpleByIDs godoc
// @Summary 按 Id 批量获取插件简要信息
// @Tags 插件
// @Accept json
// @Produce json
// @Security bearerAuth
// @Param request body object{ids=[]int64} true "插件 Id 列表"
// @Success 200 {array} map[string]interface{}
// @Router /api/plugin/instance/simpleByIds [post]
type pluginInstanceIDsRequest struct {
	IDs []int64 `json:"ids"`
}

func pluginInstanceSimpleByIDs(s *svc.ServiceContext) func(fiber.Ctx, *pluginInstanceIDsRequest) (any, error) {
	return func(c fiber.Ctx, q *pluginInstanceIDsRequest) (any, error) {
		return logic.NewPluginInstanceService(c.Context(), s).GetSimpleByIDs(q.IDs)
	}
}
func pluginInstanceUpdate(s *svc.ServiceContext) func(fiber.Ctx, *logic.PluginInstanceInput) (any, error) {
	return func(c fiber.Ctx, in *logic.PluginInstanceInput) (any, error) {
		id, err := strconv.ParseInt(c.Query("id"), 10, 64)
		if err != nil {
			return nil, error_.NewCodeTextError(code_.ParamError, "插件 Id 无效")
		}
		defer web_.AuditLog(c, fmt.Sprintf("更新插件实例(%d,%v)", id, in.Name))
		return logic.NewPluginInstanceService(c.Context(), s).Update(id, in)
	}
}

type pluginMetadataRequest struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

func pluginMetadata(s *svc.ServiceContext) func(fiber.Ctx, *pluginMetadataRequest) (any, error) {
	return func(c fiber.Ctx, q *pluginMetadataRequest) (any, error) {
		return logic.NewPluginInstanceService(c.Context(), s).Metadata(q.Type, q.Name), nil
	}
}
func pluginInstancePage(s *svc.ServiceContext) func(fiber.Ctx, *logic.PluginInstancePageQuery) (any, error) {
	return func(c fiber.Ctx, q *logic.PluginInstancePageQuery) (any, error) {
		return logic.NewPluginInstanceService(c.Context(), s).Page(q)
	}
}
func pluginInstanceCreate(s *svc.ServiceContext) func(fiber.Ctx, *logic.PluginInstanceInput) (any, error) {
	return func(c fiber.Ctx, in *logic.PluginInstanceInput) (any, error) {
		defer web_.AuditLog(c, fmt.Sprintf("创建插件实例(%v)", in.PluginName))
		return logic.NewPluginInstanceService(c.Context(), s).Create(in)
	}
}

type pluginInstanceActionRequest struct {
	Id     int64          `json:"id"`
	Action string         `json:"action"`
	Params map[string]any `json:"params"`
}

func pluginInstanceAction(s *svc.ServiceContext) func(fiber.Ctx, *pluginInstanceActionRequest) (any, error) {
	return func(c fiber.Ctx, in *pluginInstanceActionRequest) (any, error) {
		defer web_.AuditLog(c, fmt.Sprintf("执行插件操作(Id=%d, action=%v)", in.Id, in.Action))
		return logic.NewPluginInstanceService(c.Context(), s).Action(in.Id, in.Action, in.Params)
	}
}
func pluginInstanceDelete(s *svc.ServiceContext) func(fiber.Ctx) (any, error) {
	return func(c fiber.Ctx) (any, error) {
		id, err := strconv.ParseInt(c.Query("id"), 10, 64)
		if err != nil {
			return nil, error_.NewCodeTextError(code_.ParamError, "插件 Id 无效")
		}
		defer web_.AuditLog(c, fmt.Sprintf("删除插件实例(Id=%d)", id))
		if err := logic.NewPluginInstanceService(c.Context(), s).Delete(id); err != nil {
			return nil, err
		}
		return fiber.Map{}, nil
	}
}
