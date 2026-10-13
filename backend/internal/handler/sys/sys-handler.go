// Package sys 承载系统级 handler。sys_setting 当前仅供内部使用，不注册对外接口。
package sys

import (
	"handfree-work/octo-backup/internal/svc"

	"github.com/gofiber/fiber/v3"
)

// Register 预留系统模块的路由装配入口。
func Register(app *fiber.App, svcCtx *svc.ServiceContext) {
	RegisterPlugin(app, svcCtx)
	RegisterBackupPlan(app, svcCtx)
	RegisterRestoreTask(app, svcCtx)
	RegisterAudit(app, svcCtx)
}
