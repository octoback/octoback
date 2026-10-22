package sys

import (
	"handfree-work/octo-backup/internal/base/web_"
	logic "handfree-work/octo-backup/internal/modules/audit"
	"handfree-work/octo-backup/internal/svc"

	"github.com/gofiber/fiber/v3"
)

// RegisterAudit 注册审计日志查询接口。
func RegisterAudit(app *fiber.App, s *svc.ServiceContext) {
	r := app.Group("/api/audit")
	web_.RegisterRoutes(r, s.Auth, web_.Route{Path: "/page", Permission: web_.Read, Handler: web_.HandleJSON(auditPage(s))})
}

// auditPage godoc
// @Summary 查询审计日志
// @Description 查询当前系统中的用户写操作和认证结果审计记录。
// @Tags 审计
// @Accept json
// @Produce json
// @Security bearerAuth
// @Param request body logic.AuditLogPageQuery true "分页和筛选条件"
// @Success 200 {object} logic.AuditLogPageResult
// @Router /api/audit/page [post]
func auditPage(s *svc.ServiceContext) func(fiber.Ctx, *logic.AuditLogPageQuery) (*logic.AuditLogPageResult, error) {
	return func(c fiber.Ctx, q *logic.AuditLogPageQuery) (*logic.AuditLogPageResult, error) {
		return logic.NewService(c.Context(), s).Page(q)
	}
}
