package logic

import (
	"context"
	"strings"

	"handfree-work/octo-backup/internal/base/db_"
	"handfree-work/octo-backup/internal/base/error_"
	"handfree-work/octo-backup/internal/base/web_"
	"handfree-work/octo-backup/internal/models"
	"handfree-work/octo-backup/internal/svc"
)

var (
	ErrUserNotFound   = error_.NewTextError("用户不存在")
	ErrInvalidUser    = error_.NewTextError("用户参数错误")
	ErrUsernameExists = error_.NewTextError("用户名已存在")
	ErrInvalidLogin   = error_.NewTextError("用户名或密码错误")
)

type CreateUserInput struct {
	Username string `json:"username"`
	Password string `json:"password"`
	NickName string `json:"nickName"`
	Role     string `json:"role"`
}

type UpdateUserInput struct {
	Username *string `json:"username"`
	Password *string `json:"password"`
	NickName *string `json:"nickName"`
	Role     *string `json:"role"`
}

type LoginInput struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type UserPageQuery struct {
	Offset int64 `json:"offset"`
	Limit  int64 `json:"limit"`
}

type UserPageResult struct {
	Offset  int64             `json:"offset"`
	Limit   int64             `json:"limit"`
	Records []*models.SysUser `json:"records"`
	Total   int64             `json:"total"`
}

type UserService struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUserService(ctx context.Context, svcCtx *svc.ServiceContext) *UserService {
	return &UserService{ctx: ctx, svcCtx: svcCtx}
}

func (l *UserService) Create(in *CreateUserInput) (*models.SysUser, error) {
	role := web_.RoleRead
	if in != nil && strings.TrimSpace(in.Role) != "" {
		role = strings.TrimSpace(in.Role)
	}
	return l.create(in, role)
}

func (l *UserService) Register(in *CreateUserInput) (*models.SysUser, error) {
	dao := db_.New[models.SysUser](db_.NewCtx(l.ctx, l.svcCtx.Db))
	count, err := dao.Count(&models.SysUser{})
	if err != nil {
		return nil, err
	}
	role := web_.RoleRead
	if count == 0 {
		role = web_.RoleAdmin
	}
	return l.create(in, role)
}

func (l *UserService) create(in *CreateUserInput, role string) (*models.SysUser, error) {
	if in == nil || strings.TrimSpace(in.Username) == "" || strings.TrimSpace(in.Password) == "" {
		return nil, error_.NewWrapError("创建用户失败：用户名和密码不能为空", ErrInvalidUser)
	}
	if !web_.ValidRole(role) {
		return nil, error_.NewWrapError("用户角色无效", ErrInvalidUser)
	}
	dao := db_.New[models.SysUser](db_.NewCtx(l.ctx, l.svcCtx.Db))
	username := strings.TrimSpace(in.Username)
	existing, err := dao.FindOne(&models.SysUser{Username: username})
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrUsernameExists
	}

	user := &models.SysUser{Username: username, NickName: strings.TrimSpace(in.NickName), Role: role}
	if user.NickName == "" {
		user.NickName = username
	}
	if err := user.EncryptPassword(in.Password); err != nil {
		return nil, error_.NewWrapError("加密用户密码失败", err)
	}
	if err := dao.Create(user); err != nil {
		return nil, err
	}
	return user, nil
}

func (l *UserService) Login(in *LoginInput) (*models.SysUser, error) {
	if in == nil || strings.TrimSpace(in.Username) == "" || strings.TrimSpace(in.Password) == "" {
		return nil, ErrInvalidLogin
	}
	dao := db_.New[models.SysUser](db_.NewCtx(l.ctx, l.svcCtx.Db))
	user, err := dao.FindOne(&models.SysUser{Username: strings.TrimSpace(in.Username)})
	if err != nil {
		return nil, err
	}
	if user == nil || user.CheckPassword(in.Password) != nil {
		return nil, ErrInvalidLogin
	}
	return user, nil
}

func (l *UserService) List() ([]*models.SysUser, error) {
	dao := db_.New[models.SysUser](db_.NewCtx(l.ctx, l.svcCtx.Db))
	return dao.FindList(&models.SysUser{}, nil)
}

func (l *UserService) FindPage(query *UserPageQuery) (*UserPageResult, error) {
	offset, limit := int64(0), int64(20)
	if query != nil {
		offset, limit = query.Offset, query.Limit
	}
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 1000 {
		limit = 1000
	}
	dao := db_.New[models.SysUser](db_.NewCtx(l.ctx, l.svcCtx.Db))
	page := &db_.Page{Start: offset, Limit: limit}
	rows, err := dao.FindPage(&db_.PageReq[models.SysUser]{Query: &models.SysUser{}, Page: page})
	if err != nil {
		return nil, err
	}
	records := make([]*models.SysUser, 0, len(*rows))
	for i := range *rows {
		records = append(records, &(*rows)[i])
	}
	return &UserPageResult{Offset: offset, Limit: limit, Records: records, Total: page.Total}, nil
}

func (l *UserService) Get(id int64) (*models.SysUser, error) {
	dao := db_.New[models.SysUser](db_.NewCtx(l.ctx, l.svcCtx.Db))
	user, err := dao.GetById(id)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}
	return user, nil
}

func (l *UserService) Update(id int64, in *UpdateUserInput) (*models.SysUser, error) {
	if in == nil {
		return nil, error_.NewWrapError("更新用户失败：请求体不能为空", ErrInvalidUser)
	}
	user, err := l.Get(id)
	if err != nil {
		return nil, err
	}
	if in.Username != nil {
		username := strings.TrimSpace(*in.Username)
		if username == "" {
			return nil, error_.NewWrapError("更新用户失败：用户名不能为空", ErrInvalidUser)
		}
		if username != user.Username {
			dao := db_.New[models.SysUser](db_.NewCtx(l.ctx, l.svcCtx.Db))
			existing, err := dao.FindOne(&models.SysUser{Username: username})
			if err != nil {
				return nil, err
			}
			if existing != nil {
				return nil, ErrUsernameExists
			}
			user.Username = username
		}
	}
	if in.NickName != nil {
		user.NickName = strings.TrimSpace(*in.NickName)
	}
	if in.Password != nil {
		password := strings.TrimSpace(*in.Password)
		if password != "" {
			if err := user.EncryptPassword(password); err != nil {
				return nil, error_.NewWrapError("加密用户密码失败", err)
			}
		}
	}
	if in.Role != nil {
		role := strings.TrimSpace(*in.Role)
		if !web_.ValidRole(role) {
			return nil, error_.NewWrapError("用户角色无效", ErrInvalidUser)
		}
		user.Role = role
	}
	dao := db_.New[models.SysUser](db_.NewCtx(l.ctx, l.svcCtx.Db))
	if _, err := dao.UpdateById(id, user); err != nil {
		return nil, err
	}
	return user, nil
}

func (l *UserService) Delete(id int64) error {
	if _, err := l.Get(id); err != nil {
		return err
	}
	dao := db_.New[models.SysUser](db_.NewCtx(l.ctx, l.svcCtx.Db))
	_, err := dao.Delete(&id)
	return err
}
