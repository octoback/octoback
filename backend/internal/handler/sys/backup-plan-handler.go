package sys

import (
	"handfree-work/octo-backup/internal/base/error_"
	"handfree-work/octo-backup/internal/base/error_/code_"
	"handfree-work/octo-backup/internal/base/web_"
	logic "handfree-work/octo-backup/internal/modules/backup"
	"handfree-work/octo-backup/internal/svc"
	"strconv"

	"github.com/gofiber/fiber/v3"
)

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
func backupPlanList(s *svc.ServiceContext) func(fiber.Ctx) (any, error) {
	return func(c fiber.Ctx) (any, error) {
		return logic.NewBackupPlanService(c.Context(), s).List()
	}
}
func backupPlanLogContent(s *svc.ServiceContext) func(fiber.Ctx) (any, error) {
	return func(c fiber.Ctx) (any, error) {
		id, err := strconv.ParseInt(c.Query("id"), 10, 64)
		if err != nil {
			return nil, error_.NewCodeTextError(code_.ParamError, "运行记录 Id 无效")
		}
		return logic.NewBackupPlanService(c.Context(), s).LogContent(id)
	}
}
func backupPlanLogPage(s *svc.ServiceContext) func(fiber.Ctx, *logic.BackupLogPageQuery) (any, error) {
	return func(c fiber.Ctx, q *logic.BackupLogPageQuery) (any, error) {
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
