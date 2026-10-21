package basic_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"handfree-work/octo-backup/internal/base/db_"
	"handfree-work/octo-backup/internal/base/web_"
	"handfree-work/octo-backup/internal/handler"
	"handfree-work/octo-backup/internal/models"
	"handfree-work/octo-backup/internal/svc"

	"github.com/gofiber/fiber/v3"
)

func TestSwaggerDocumentation(t *testing.T) {
	db, err := db_.OpenSQLite(filepath.Join(t.TempDir(), "db.sqlite"))
	if err != nil {
		t.Fatalf("OpenSQLite() error = %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("DB() error = %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db_.Migrate(db, &models.SysUser{}); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}
	app := handler.NewApp(&svc.ServiceContext{
		Db:   db,
		Auth: web_.Config{Secret: "test-secret", TokenTTL: time.Hour},
	})

	index := doRawRequest(t, app, "/swagger/index.html")
	if index.StatusCode != http.StatusOK {
		t.Fatalf("GET /swagger/index.html status = %d, want %d", index.StatusCode, http.StatusOK)
	}
	if !strings.Contains(index.Body, "SwaggerUIBundle") {
		t.Fatal("Swagger UI 页面必须加载 SwaggerUIBundle")
	}

	document := doRawRequest(t, app, "/swagger/doc.json")
	if document.StatusCode != http.StatusOK {
		t.Fatalf("GET /swagger/doc.json status = %d, want %d", document.StatusCode, http.StatusOK)
	}
	for _, expected := range []string{
		"\"/api/auth/login\"",
		"\"/api/auth/register\"",
		"\"/api/user/create\"",
		"\"/api/user/list\"",
		"\"/api/user/page\"",
		"\"/api/user/info\"",
		"\"/api/user/update\"",
		"\"/api/user/delete\"",
		"\"/api/user/batchDelete\"",
		"\"bearerAuth\"",
	} {
		if !strings.Contains(document.Body, expected) {
			t.Fatalf("OpenAPI 文档缺少 %s", expected)
		}
	}
	if strings.Contains(document.Body, "\"host\":") {
		t.Fatal("OpenAPI 文档不应固定 host，以便 Swagger UI 使用当前服务地址")
	}
	for _, unsupportedMethod := range []string{"\"get\":", "\"put\":", "\"delete\":"} {
		if strings.Contains(document.Body, unsupportedMethod) {
			t.Fatalf("OpenAPI 文档不应包含 %s 业务接口", unsupportedMethod)
		}
	}
}

type rawResponse struct {
	StatusCode int
	Body       string
}

func doRawRequest(t *testing.T, app *fiber.App, path string) rawResponse {
	t.Helper()
	res, err := app.Test(httptest.NewRequest(http.MethodGet, path, nil))
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	t.Cleanup(func() { _ = res.Body.Close() })
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("read %s response: %v", path, err)
	}
	return rawResponse{StatusCode: res.StatusCode, Body: string(body)}
}
