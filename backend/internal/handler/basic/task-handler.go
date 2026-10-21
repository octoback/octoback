package basic

import (
	"handfree-work/octo-backup/internal/base/error_"
	"handfree-work/octo-backup/internal/base/error_/code_"
	"handfree-work/octo-backup/internal/base/task_"
	"handfree-work/octo-backup/internal/base/web_"

	"github.com/gofiber/fiber/v3"
)

// RegisterTask 注册通用内存任务状态查询接口。
func RegisterTask(app *fiber.App, auth web_.Config) {
	web_.RegisterRoutes(app.Group("/api/task"), auth,
		web_.Route{Path: "/status", Permission: web_.Read, Handler: web_.Handle(taskStatus)},
	)
}

func taskStatus(c fiber.Ctx) (any, error) {
	taskId := c.Query("taskId")
	if taskId == "" {
		return nil, error_.NewTextError("任务 Id 不能为空")
	}
	task, ok := task_.DefaultExecutor.GetById(taskId)
	if !ok {
		return nil, error_.NewCodeTextError(code_.TaskNotFound, "任务不存在或已过期")
	}
	return task, nil
}
