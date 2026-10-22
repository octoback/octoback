package basic

import (
	"handfree-work/octo-backup/internal/svc"

	"github.com/gofiber/fiber/v3"
)

// Register 注册 basic 模块下的 Swagger、认证和任务接口。
func Register(app *fiber.App, svcCtx *svc.ServiceContext) {
	RegisterSwagger(app)
	RegisterAuth(app, svcCtx)
	RegisterTask(app, svcCtx.Auth)
}
