package admin

import (
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
	"testing"

	"handfree-work/octo-backup/internal/base/db_"
	"handfree-work/octo-backup/internal/models"

	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func testDatabase(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := db_.OpenSQLite(filepath.Join(t.TempDir(), "db.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	if err := db_.Migrate(db, &models.SysUser{}); err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	return db
}

func TestResetPassword(t *testing.T) {
	db := testDatabase(t)
	user := &models.SysUser{Username: "admin", Role: "admin", NickName: "管理员"}
	if err := user.EncryptPassword("old-password"); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatal(err)
	}

	if err := ResetPassword(db, "admin", "new-password"); err != nil {
		t.Fatal(err)
	}
	var updated models.SysUser
	if err := db.First(&updated, user.Id).Error; err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256([]byte("new-password"))
	if err := bcrypt.CompareHashAndPassword([]byte(updated.Password), []byte(hex.EncodeToString(digest[:]))); err != nil {
		t.Fatalf("new password is not valid: %v", err)
	}

	if err := ResetPassword(db, "missing", "new-password"); err == nil {
		t.Fatal("expected missing user error")
	}
	if err := ResetPassword(db, "admin", " "); err == nil {
		t.Fatal("expected empty password error")
	}
}

func TestTUISelectsResetPassword(t *testing.T) {
	db := testDatabase(t)
	m := newTUIModel(db)
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if !updated.(tuiModel).form {
		t.Fatal("enter on the first menu item should open the reset form")
	}
}

func TestLocaleLabels(t *testing.T) {
	if got := labelsForLocale("en-US"); got.reset != "Reset user password" || got.quit != "Quit" {
		t.Fatalf("English labels = %#v", got)
	}
	if got := labelsForLocale("zh-CN"); got.reset != "重置用户密码" || got.quit != "退出" {
		t.Fatalf("Chinese labels = %#v", got)
	}
	if got := labelsForLocale("fr-FR"); got.reset != "重置用户密码" {
		t.Fatalf("unsupported locale should fall back to Chinese, got %#v", got)
	}
}
