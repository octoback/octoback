package admin

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"

	"handfree-work/octo-backup/internal/base/error_"
	"handfree-work/octo-backup/internal/models"

	"gorm.io/gorm"
)

var (
	ErrUserNotFound = errors.New("用户不存在")
	ErrInvalidInput = errors.New("用户名和密码不能为空")
)

// ResetPassword 使用 bcrypt 更新本地管理界面中的用户密码。
func ResetPassword(db *gorm.DB, username, password string) error {
	username = strings.TrimSpace(username)
	if username == "" || strings.TrimSpace(password) == "" {
		return ErrInvalidInput
	}
	var user models.SysUser
	if err := db.Where("username = ?", username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return error_.NewWrapError(fmt.Sprintf("用户不存在: %s", username), ErrUserNotFound)
		}
		return error_.NewWrapError("查询用户", err)
	}
	digest := sha256.Sum256([]byte(password))
	if err := user.EncryptPassword(hex.EncodeToString(digest[:])); err != nil {
		return error_.NewWrapError("加密密码", err)
	}
	if err := db.Model(&models.SysUser{}).Where("id = ?", user.Id).Update("password", user.Password).Error; err != nil {
		return error_.NewWrapError("保存密码", err)
	}
	return nil
}

// Run 启动 Bubble Tea 终端界面。
func Run(db *gorm.DB, input io.Reader, output io.Writer) error {
	return runTUI(db, input, output)
}
