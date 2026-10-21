package basic

import (
	"handfree-work/octo-backup/internal/base/error_"
	"handfree-work/octo-backup/internal/base/web_"
	logic "handfree-work/octo-backup/internal/modules/user"
	"handfree-work/octo-backup/internal/svc"

	"github.com/gofiber/fiber/v3"
)

func RegisterAuth(app *fiber.App, svcCtx *svc.ServiceContext) {
	routes := app.Group("/api/auth")
	web_.RegisterRoutes(routes, svcCtx.Auth,
		web_.Route{Path: "/register", Permission: web_.Guest, Handler: web_.HandleJSON(registerUser(svcCtx))},
		web_.Route{Path: "/login", Permission: web_.Guest, Handler: web_.HandleJSON(loginUser(svcCtx))},
	)
}

// registerUser godoc
// @Summary 注册用户
// @Description 注册首个用户会自动授予 admin 角色，后续注册用户为 read 角色。
// @Tags 认证
// @Accept json
// @Produce json
// @Param request body logic.CreateUserInput true "注册信息"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 409 {object} map[string]interface{}
// @Router /api/auth/register [post]
func registerUser(svcCtx *svc.ServiceContext) func(fiber.Ctx, *logic.CreateUserInput) (any, error) {
	return func(c fiber.Ctx, input *logic.CreateUserInput) (any, error) {
		defer web_.AuditLog(c, "注册用户")
		if true {
			//当前禁止注册用户
			return nil, error_.NewTextError("当前系统已禁用用户注册功能")
		}
		return logic.NewUserService(c.Context(), svcCtx).Register(input)
	}
}

// loginUser godoc
// @Summary 用户登录
// @Description 登录成功后返回有效期默认 7 天的 JWT Bearer token。
// @Tags 认证
// @Accept json
// @Produce json
// @Param request body logic.LoginInput true "登录信息"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /api/auth/login [post]
func loginUser(svcCtx *svc.ServiceContext) func(fiber.Ctx, *logic.LoginInput) (any, error) {
	return func(c fiber.Ctx, input *logic.LoginInput) (any, error) {
		if input != nil {
			c.Locals("auditUsername", input.Username)
		}
		user, err := logic.NewUserService(c.Context(), svcCtx).Login(input)
		if err != nil {
			web_.AuditLog(c, "登录失败")
			return nil, err
		}
		if user.Id == nil {
			return nil, error_.NewTextError("用户数据错误")
		}
		token, expiresAt, err := svcCtx.Auth.Issue(*user.Id, user.Username, user.Role)
		if err != nil {
			web_.AuditLog(c, "登录失败")
			return nil, error_.NewWrapError("签发登录凭证失败", err)
		}
		c.Locals("authClaims", &web_.Claims{UserId: *user.Id, Username: user.Username, Role: user.Role})
		web_.AuditLog(c, "登录成功")
		return fiber.Map{
			"token":     token,
			"expiresAt": expiresAt.Unix(),
			"user": fiber.Map{
				"id":       user.Id,
				"username": user.Username,
				"role":     user.Role,
			},
		}, nil
	}
}
