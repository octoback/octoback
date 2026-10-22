package handler

import (
	"handfree-work/octo-backup/internal/base/web_"
	backupHandler "handfree-work/octo-backup/internal/handler/backup"
	"handfree-work/octo-backup/internal/handler/basic"
	sysHandler "handfree-work/octo-backup/internal/handler/sys"
	"handfree-work/octo-backup/internal/middleware"
	"handfree-work/octo-backup/internal/svc"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/recover"
)

// NewApp 创建用于测试及嵌入场景的完整 HTTP 应用。
func NewApp(svcCtx *svc.ServiceContext) *fiber.App {
	app := fiber.New(fiber.Config{ErrorHandler: web_.ErrorHandler})
	app.Use(recover.New())
	Register(app, svcCtx)
	return app
}

// Register 按 basic、sys 和 backup 模块统一注册 handler。
func Register(app *fiber.App, svcCtx *svc.ServiceContext) {
	app.Use(middleware.Audit(svcCtx))
	basic.Register(app, svcCtx)
	sysHandler.Register(app, svcCtx)
	backupHandler.Register(app, svcCtx)
}
