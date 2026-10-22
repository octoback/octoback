package sys

import (
	"strconv"

	"handfree-work/octo-backup/internal/base/error_"
	"handfree-work/octo-backup/internal/base/error_/code_"
	"handfree-work/octo-backup/internal/base/web_"
	"handfree-work/octo-backup/internal/models"
	logic "handfree-work/octo-backup/internal/modules/user"
	"handfree-work/octo-backup/internal/svc"

	"github.com/gofiber/fiber/v3"
)

// RegisterUsers 注册用户管理接口。
func RegisterUsers(app *fiber.App, svcCtx *svc.ServiceContext) {
	users := app.Group("/api/user")
	web_.RegisterRoutes(users, svcCtx.Auth,
		web_.Route{Path: "/create", Permission: web_.Admin, Handler: web_.HandleJSON(createUser(svcCtx))},
		web_.Route{Path: "/list", Permission: web_.Read, Handler: web_.HandleJSON(listUsers(svcCtx))},
		web_.Route{Path: "/page", Permission: web_.Read, Handler: web_.HandleJSON(listUsers(svcCtx))},
		web_.Route{Path: "/info", Permission: web_.Read, Handler: web_.Handle(getUser(svcCtx))},
		web_.Route{Path: "/update", Permission: web_.Write, Handler: web_.HandleJSON(updateUser(svcCtx))},
		web_.Route{Path: "/delete", Permission: web_.Admin, Handler: web_.Handle(deleteUser(svcCtx))},
		web_.Route{Path: "/batchDelete", Permission: web_.Admin, Handler: web_.HandleJSON(batchDeleteUser(svcCtx))},
	)
}

// createUser godoc
// @Summary 创建用户
// @Description 需要 admin 权限，可设置 admin、write 或 read 角色。
// @Tags 用户
// @Accept json
// @Produce json
// @Security bearerAuth
// @Param request body logic.CreateUserInput true "用户信息"
// @Success 201 {object} UserResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Router /api/user/create [post]
func createUser(svcCtx *svc.ServiceContext) func(fiber.Ctx, *logic.CreateUserInput) (*models.SysUser, error) {
	return func(c fiber.Ctx, input *logic.CreateUserInput) (*models.SysUser, error) {
		defer web_.AuditLog(c, "创建用户")
		return logic.NewUserService(c.Context(), svcCtx).Create(input)
	}
}

// listUsers godoc
// @Summary 获取用户列表
// @Description 需要 read、write 或 admin 权限。
// @Tags 用户
// @Produce json
// @Security bearerAuth
// @Success 200 {object} UserListResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Router /api/user/list [post]
// @Router /api/user/page [post]
func listUsers(svcCtx *svc.ServiceContext) func(fiber.Ctx, *logic.UserPageQuery) (*logic.UserPageResult, error) {
	return func(c fiber.Ctx, query *logic.UserPageQuery) (*logic.UserPageResult, error) {
		return logic.NewUserService(c.Context(), svcCtx).FindPage(query)
	}
}

// getUser godoc
// @Summary 获取用户详情
// @Description 需要 read、write 或 admin 权限。
// @Tags 用户
// @Produce json
// @Security bearerAuth
// @Param id query int true "用户 Id"
// @Success 200 {object} UserResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/user/info [post]
func getUser(svcCtx *svc.ServiceContext) func(fiber.Ctx) (*models.SysUser, error) {
	return func(c fiber.Ctx) (*models.SysUser, error) {
		id, err := parseUserId(c)
		if err != nil {
			return nil, error_.NewCodeTextError(code_.ParamError, "用户 Id 无效")
		}
		return logic.NewUserService(c.Context(), svcCtx).Get(id)
	}
}

// updateUser godoc
// @Summary 更新用户
// @Description 需要 write 或 admin 权限；非管理员只能更新自己的资料且不能修改角色。
// @Tags 用户
// @Accept json
// @Produce json
// @Security bearerAuth
// @Param id query int true "用户 Id"
// @Param request body logic.UpdateUserInput true "更新信息"
// @Success 200 {object} UserResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/user/update [post]
func updateUser(svcCtx *svc.ServiceContext) func(fiber.Ctx, *logic.UpdateUserInput) (*models.SysUser, error) {
	return func(c fiber.Ctx, input *logic.UpdateUserInput) (*models.SysUser, error) {
		defer web_.AuditLog(c, "更新用户")
		id, err := parseUserId(c)
		if err != nil {
			return nil, error_.NewCodeTextError(code_.ParamError, "用户 Id 无效")
		}
		claims, ok := web_.ClaimsFromContext(c)
		if !ok {
			return nil, error_.NewCodeTextError(code_.AuthError, "未登录或登录已过期")
		}
		if claims.Role != web_.RoleAdmin && claims.UserId != id {
			return nil, error_.NewCodeTextError(code_.PermissionError, "没有访问权限")
		}
		if claims.Role != web_.RoleAdmin && input.Role != nil {
			return nil, error_.NewCodeTextError(code_.PermissionError, "只有管理员可以调整角色")
		}
		return logic.NewUserService(c.Context(), svcCtx).Update(id, input)
	}
}

// deleteUser godoc
// @Summary 删除用户
// @Description 需要 admin 权限。
// @Tags 用户
// @Produce json
// @Security bearerAuth
// @Param id query int true "用户 Id"
// @Success 204
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/user/delete [post]
func deleteUser(svcCtx *svc.ServiceContext) func(fiber.Ctx) (fiber.Map, error) {
	return func(c fiber.Ctx) (fiber.Map, error) {
		defer web_.AuditLog(c, "删除用户")
		id, err := parseUserId(c)
		if err != nil {
			return nil, error_.NewCodeTextError(code_.ParamError, "用户 Id 无效")
		}
		if err := logic.NewUserService(c.Context(), svcCtx).Delete(id); err != nil {
			return nil, err
		}
		return fiber.Map{}, nil
	}
}

type batchDeleteInput struct {
	Ids []int64 `json:"ids"`
}

// batchDeleteUser godoc
// @Summary 批量删除用户
// @Description 需要 admin 权限。
// @Tags 用户
// @Accept json
// @Produce json
// @Security bearerAuth
// @Param request body batchDeleteInput true "用户 Id 列表"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Router /api/user/batchDelete [post]
func batchDeleteUser(svcCtx *svc.ServiceContext) func(fiber.Ctx, *batchDeleteInput) (fiber.Map, error) {
	return func(c fiber.Ctx, input *batchDeleteInput) (fiber.Map, error) {
		defer web_.AuditLog(c, "批量删除用户")
		if len(input.Ids) == 0 {
			return nil, error_.NewCodeTextError(code_.ParamError, "用户 Id 不能为空")
		}
		service := logic.NewUserService(c.Context(), svcCtx)
		for _, id := range input.Ids {
			if err := service.Delete(id); err != nil {
				return nil, err
			}
		}
		return fiber.Map{}, nil
	}
}

func parseUserId(c fiber.Ctx) (int64, error) {
	return strconv.ParseInt(c.Query("id"), 10, 64)
}

// ErrorResponse 表示接口错误响应。
type ErrorResponse struct {
	Error string `json:"error" example:"没有访问权限"`
}

// UserResponse 表示单个用户的成功响应。
type UserResponse struct {
	Data map[string]any `json:"data"`
}

// UserListResponse 表示用户列表的成功响应。
type UserListResponse struct {
	Data struct {
		Items []map[string]any `json:"items"`
	} `json:"data"`
}
