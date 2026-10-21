package backup

import (
	"handfree-work/octo-backup/internal/base/error_"
	"handfree-work/octo-backup/internal/base/error_/code_"
	"handfree-work/octo-backup/internal/base/web_"
	logic "handfree-work/octo-backup/internal/modules/backup"
	"handfree-work/octo-backup/internal/svc"
	"strconv"

	"github.com/gofiber/fiber/v3"
)

// RegisterBackupPlan 注册备份计划和运行记录接口。
func RegisterBackupPlan(app *fiber.App, s *svc.ServiceContext) {
	r := app.Group("/api/plan")
	web_.RegisterRoutes(r, s.Auth,
		web_.Route{Path: "/page", Permission: web_.Read, Handler: web_.HandleJSON(backupPlanPage(s))},
		web_.Route{Path: "/list", Permission: web_.Read, Handler: web_.Handle(backupPlanList(s))},
		web_.Route{Path: "/info", Permission: web_.Read, Handler: web_.Handle(backupPlanInfo(s))},
		web_.Route{Path: "/create", Permission: web_.Write, Handler: web_.HandleJSON(backupPlanCreate(s))},
		web_.Route{Path: "/update", Permission: web_.Write, Handler: web_.HandleJSON(backupPlanUpdate(s))},
		web_.Route{Path: "/delete", Permission: web_.Admin, Handler: web_.Handle(backupPlanDelete(s))},
		web_.Route{Path: "/run", Permission: web_.Write, Handler: web_.HandleJSON(backupPlanRun(s))},
		web_.Route{Path: "/run/info", Permission: web_.Read, Handler: web_.Handle(backupPlanLog(s))},
		web_.Route{Path: "/run/log", Permission: web_.Read, Handler: web_.Handle(backupPlanLogContent(s))},
		web_.Route{Path: "/run/page", Permission: web_.Read, Handler: web_.HandleJSON(backupPlanLogPage(s))},
	)
}

// RegisterRestoreTask 注册独立恢复任务接口。
func RegisterRestoreTask(app *fiber.App, s *svc.ServiceContext) {
	r := app.Group("/api/restore/task")
	web_.RegisterRoutes(r, s.Auth,
		web_.Route{Path: "/create", Permission: web_.Write, Handler: web_.HandleJSON(restoreTaskCreate(s))},
		web_.Route{Path: "/page", Permission: web_.Read, Handler: web_.HandleJSON(restoreTaskPage(s))},
		web_.Route{Path: "/info", Permission: web_.Read, Handler: web_.Handle(restoreTaskInfo(s))},
		web_.Route{Path: "/log", Permission: web_.Read, Handler: web_.Handle(restoreTaskLog(s))},
		web_.Route{Path: "/delete", Permission: web_.Write, Handler: web_.Handle(restoreTaskDelete(s))},
	)
}

func restoreTaskCreate(s *svc.ServiceContext) func(fiber.Ctx, *logic.RestoreTaskInput) (any, error) {
	return func(c fiber.Ctx, input *logic.RestoreTaskInput) (any, error) {
		return logic.NewRestoreTaskService(c.Context(), s).Create(input)
	}
}

func restoreTaskInfo(s *svc.ServiceContext) func(fiber.Ctx) (any, error) {
	return func(c fiber.Ctx) (any, error) {
		id, err := strconv.ParseInt(c.Query("id"), 10, 64)
		if err != nil {
			return nil, error_.NewCodeTextError(code_.ParamError, "恢复任务 Id 无效")
		}
		return logic.NewRestoreTaskService(c.Context(), s).Info(id)
	}
}
func restoreTaskPage(s *svc.ServiceContext) func(fiber.Ctx, *logic.RestoreTaskPageQuery) (any, error) {
	return func(c fiber.Ctx, q *logic.RestoreTaskPageQuery) (any, error) {
		return logic.NewRestoreTaskService(c.Context(), s).Page(q)
	}
}
func restoreTaskLog(s *svc.ServiceContext) func(fiber.Ctx) (any, error) {
	return func(c fiber.Ctx) (any, error) {
		id, err := strconv.ParseInt(c.Query("id"), 10, 64)
		if err != nil {
			return nil, error_.NewCodeTextError(code_.ParamError, "恢复任务 Id 无效")
		}
		return logic.NewRestoreTaskService(c.Context(), s).LogContent(id)
	}
}
func restoreTaskDelete(s *svc.ServiceContext) func(fiber.Ctx) (any, error) {
	return func(c fiber.Ctx) (any, error) {
		id, err := strconv.ParseInt(c.Query("id"), 10, 64)
		if err != nil {
			return nil, error_.NewCodeTextError(code_.ParamError, "恢复任务 Id 无效")
		}
		if err := logic.NewRestoreTaskService(c.Context(), s).Delete(id); err != nil {
			return nil, err
		}
		return fiber.Map{}, nil
	}
}
func backupPlanList(s *svc.ServiceContext) func(fiber.Ctx) (any, error) {
	return func(c fiber.Ctx) (any, error) {
		return logic.NewBackupPlanService(c.Context(), s).List()
	}
}

// backupPlanLogContent 返回指定备份运行记录的可读日志和 Restic summary。
// @Summary 查询备份日志详情
// @Description 返回全量可读执行日志以及解析后的 Restic summary 对象。
// @Tags 备份计划
// @Accept json
// @Produce json
// @Param id query int true "运行记录 Id"
// @Success 200 {object} map[string]interface{}
// @Router /api/plan/run/log [post]
func backupPlanLogContent(s *svc.ServiceContext) func(fiber.Ctx) (any, error) {
	return func(c fiber.Ctx) (any, error) {
		id, err := strconv.ParseInt(c.Query("id"), 10, 64)
		if err != nil {
			return nil, error_.NewCodeTextError(code_.ParamError, "运行记录 Id 无效")
		}
		return logic.NewBackupPlanService(c.Context(), s).LogContent(id)
	}
}
func backupPlanLogPage(s *svc.ServiceContext) func(fiber.Ctx, *logic.BackupTaskPageQuery) (any, error) {
	return func(c fiber.Ctx, q *logic.BackupTaskPageQuery) (any, error) {
		return logic.NewBackupPlanService(c.Context(), s).LogPage(q)
	}
}
func backupPlanRun(s *svc.ServiceContext) func(fiber.Ctx, *struct{}) (any, error) {
	return func(c fiber.Ctx, _ *struct{}) (any, error) {
		defer web_.AuditLog(c, "执行备份计划")
		id, err := strconv.ParseInt(c.Query("id"), 10, 64)
		if err != nil {
			return nil, error_.NewCodeTextError(code_.ParamError, "备份计划 Id 无效")
		}
		return logic.NewBackupPlanService(c.Context(), s).Run(id)
	}
}
func backupPlanLog(s *svc.ServiceContext) func(fiber.Ctx) (any, error) {
	return func(c fiber.Ctx) (any, error) {
		id, err := strconv.ParseInt(c.Query("id"), 10, 64)
		if err != nil {
			return nil, error_.NewCodeTextError(code_.ParamError, "运行记录 Id 无效")
		}
		return logic.NewBackupPlanService(c.Context(), s).Log(id)
	}
}
func backupPlanPage(s *svc.ServiceContext) func(fiber.Ctx, *logic.BackupPlanPageQuery) (any, error) {
	return func(c fiber.Ctx, q *logic.BackupPlanPageQuery) (any, error) {
		return logic.NewBackupPlanService(c.Context(), s).Page(q)
	}
}
func backupPlanInfo(s *svc.ServiceContext) func(fiber.Ctx) (any, error) {
	return func(c fiber.Ctx) (any, error) {
		id, err := strconv.ParseInt(c.Query("id"), 10, 64)
		if err != nil {
			return nil, error_.NewCodeTextError(code_.ParamError, "备份计划 Id 无效")
		}
		return logic.NewBackupPlanService(c.Context(), s).Info(id)
	}
}
func backupPlanCreate(s *svc.ServiceContext) func(fiber.Ctx, *logic.BackupPlanInput) (any, error) {
	return func(c fiber.Ctx, in *logic.BackupPlanInput) (any, error) {
		defer web_.AuditLog(c, "创建备份计划")
		return logic.NewBackupPlanService(c.Context(), s).Create(in)
	}
}
func backupPlanUpdate(s *svc.ServiceContext) func(fiber.Ctx, *logic.BackupPlanInput) (any, error) {
	return func(c fiber.Ctx, in *logic.BackupPlanInput) (any, error) {
		defer web_.AuditLog(c, "更新备份计划")
		id, err := strconv.ParseInt(c.Query("id"), 10, 64)
		if err != nil {
			return nil, error_.NewCodeTextError(code_.ParamError, "备份计划 Id 无效")
		}
		return logic.NewBackupPlanService(c.Context(), s).Update(id, in)
	}
}
func backupPlanDelete(s *svc.ServiceContext) func(fiber.Ctx) (any, error) {
	return func(c fiber.Ctx) (any, error) {
		defer web_.AuditLog(c, "删除备份计划")
		id, err := strconv.ParseInt(c.Query("id"), 10, 64)
		if err != nil {
			return nil, error_.NewCodeTextError(code_.ParamError, "备份计划 Id 无效")
		}
		if err = logic.NewBackupPlanService(c.Context(), s).Delete(id); err != nil {
			return nil, err
		}
		return fiber.Map{}, nil
	}
}
