package backup

import (
	"handfree-work/octo-backup/internal/svc"

	"github.com/gofiber/fiber/v3"
)

// Register 注册备份计划、恢复任务、插件实例和仓库操作接口。
func Register(app *fiber.App, s *svc.ServiceContext) {
	RegisterBackupPlan(app, s)
	RegisterRestoreTask(app, s)
	RegisterPlugin(app, s)
	RegisterRepository(app, s)
}
