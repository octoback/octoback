package middleware

import (
	"handfree-work/octo-backup/internal/base/web_"
	"handfree-work/octo-backup/internal/models"
	"handfree-work/octo-backup/internal/svc"
	"net/http/httptest"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

func TestAuditMiddlewarePersistsAuthenticatedApiOperation(t *testing.T) {
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.AutoMigrate(&models.AuditLog{}); err != nil {
		t.Fatal(err)
	}
	app := fiber.New()
	app.Use(Audit(&svc.ServiceContext{Db: database}))
	app.Post("/api/test", func(c fiber.Ctx) error {
		c.Locals("authClaims", &web_.Claims{UserId: 7, Username: "alice"})
		web_.AuditLog(c, "测试操作")
		return c.SendStatus(200)
	})
	if _, err := app.Test(httptest.NewRequest("POST", "/api/test", nil)); err != nil {
		t.Fatal(err)
	}
	var entry models.AuditLog
	if err := database.First(&entry).Error; err != nil {
		t.Fatal(err)
	}
	if entry.UserId != 7 || entry.Username != "alice" || entry.Operation != "测试操作" || entry.Path != "/api/test" || entry.Method != "POST" {
		t.Fatalf("unexpected audit entry: %+v", entry)
	}
}
