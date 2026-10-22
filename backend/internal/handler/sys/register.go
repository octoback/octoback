package sys

import (
	"handfree-work/octo-backup/internal/svc"

	"github.com/gofiber/fiber/v3"
)

// Register 注册 sys 模块下的用户管理和审计日志接口。
func Register(app *fiber.App, svcCtx *svc.ServiceContext) {
	RegisterUsers(app, svcCtx)
	RegisterAudit(app, svcCtx)
}
