package basic_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"handfree-work/octo-backup/internal/base/db_"
	"handfree-work/octo-backup/internal/base/web_"
	"handfree-work/octo-backup/internal/handler"
	"handfree-work/octo-backup/internal/models"
	"handfree-work/octo-backup/internal/svc"

	"github.com/gofiber/fiber/v3"
)

type response struct {
	HTTPStatus int
	Code       int
	Data       map[string]any
}

func doJSONRequest(t *testing.T, app *fiber.App, method, path string, body any, tokens ...string) response {
	t.Helper()
	var payload io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal request: %v", err)
		}
		payload = bytes.NewReader(encoded)
	}
	req := httptest.NewRequest(method, path, payload)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if len(tokens) > 0 && tokens[0] != "" {
		req.Header.Set("Authorization", "Bearer "+tokens[0])
	}
	res, err := app.Test(req)
	if err != nil {
		t.Fatalf("request %s %s: %v", method, path, err)
	}
	t.Cleanup(func() { _ = res.Body.Close() })
	decoded := struct {
		Code int            `json:"code"`
		Data map[string]any `json:"data"`
	}{}
	if err := json.NewDecoder(res.Body).Decode(&decoded); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return response{HTTPStatus: res.StatusCode, Code: decoded.Code, Data: decoded.Data}
}

func TestRegisterLoginAndDeclaredPermissions(t *testing.T) {
	db, err := db_.OpenSQLite(filepath.Join(t.TempDir(), "db.sqlite"))
	if err != nil {
		t.Fatalf("OpenSQLite() error = %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("DB() error = %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db_.Migrate(db, &models.SysUser{}, &models.AuditLog{}); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}
	app := handler.NewApp(&svc.ServiceContext{
		Db:   db,
		Auth: web_.Config{Secret: "test-secret", TokenTTL: time.Hour},
	})

	unauthorized := doJSONRequest(t, app, http.MethodPost, "/api/user/list", nil)
	if unauthorized.HTTPStatus != http.StatusOK || unauthorized.Code == 0 {
		t.Fatalf("unauthorized list response = %#v, want HTTP 200 with business error", unauthorized)
	}

	registered := doJSONRequest(t, app, http.MethodPost, "/api/auth/register", map[string]string{
		"username": "admin",
		"password": "secret",
	})
	if registered.Code != 0 {
		t.Fatalf("register business code = %d, want 0", registered.Code)
	}
	if registered.Data["role"] != "admin" {
		t.Fatalf("first registered role = %#v, want admin", registered.Data["role"])
	}

	badLogin := doJSONRequest(t, app, http.MethodPost, "/api/auth/login", map[string]string{
		"username": "admin",
		"password": "wrong",
	})
	if badLogin.HTTPStatus != http.StatusOK || badLogin.Code == 0 {
		t.Fatalf("bad login response = %#v, want HTTP 200 with business error", badLogin)
	}
	var failedAudit models.AuditLog
	if err := db.Where("operation = ?", "登录失败").First(&failedAudit).Error; err != nil || failedAudit.Username != "admin" || failedAudit.UserId != 0 {
		t.Fatalf("failed login audit = %#v, error = %v", failedAudit, err)
	}

	login := doJSONRequest(t, app, http.MethodPost, "/api/auth/login", map[string]string{
		"username": "admin",
		"password": "secret",
	})
	if login.Code != 0 {
		t.Fatalf("login business code = %d, want 0", login.Code)
	}
	var loginAudit models.AuditLog
	if err := db.Where("operation = ?", "登录成功").First(&loginAudit).Error; err != nil || loginAudit.UserId == 0 {
		t.Fatalf("successful login audit = %#v, error = %v", loginAudit, err)
	}
	token, ok := login.Data["token"].(string)
	if !ok || token == "" {
		t.Fatalf("login token = %#v, want non-empty string", login.Data["token"])
	}

	authorized := doJSONRequest(t, app, http.MethodPost, "/api/user/list", map[string]any{}, token)
	if authorized.Code != 0 {
		t.Fatalf("authorized list business code = %d, want 0", authorized.Code)
	}
	writer := doJSONRequest(t, app, http.MethodPost, "/api/user/create", map[string]string{
		"username": "writer",
		"password": "secret",
		"role":     "write",
	}, token)
	if writer.Code != 0 || writer.Data["role"] != "write" {
		t.Fatalf("writer creation response = %#v, want created write user", writer)
	}
	writerId := int64(writer.Data["id"].(float64))
	writerLogin := doJSONRequest(t, app, http.MethodPost, "/api/auth/login", map[string]string{
		"username": "writer",
		"password": "secret",
	})
	writerToken := writerLogin.Data["token"].(string)
	writerUpdate := doJSONRequest(t, app, http.MethodPost, "/api/user/update?id="+strconv.FormatInt(writerId, 10), map[string]string{
		"nickName": "updated by writer",
	}, writerToken)
	if writerUpdate.Code != 0 {
		t.Fatalf("writer update business code = %d, want 0", writerUpdate.Code)
	}
	roleEscalation := doJSONRequest(t, app, http.MethodPost, "/api/user/update?id="+strconv.FormatInt(writerId, 10), map[string]string{
		"role": "admin",
	}, writerToken)
	if roleEscalation.HTTPStatus != http.StatusOK || roleEscalation.Code == 0 {
		t.Fatalf("writer role update response = %#v, want HTTP 200 with business error", roleEscalation)
	}

	second := doJSONRequest(t, app, http.MethodPost, "/api/auth/register", map[string]string{
		"username": "reader",
		"password": "secret",
	})
	if second.Code != 0 || second.Data["role"] != "read" {
		t.Fatalf("second registered response = %#v, want created read user", second)
	}
	readerId := int64(second.Data["id"].(float64))
	readerLogin := doJSONRequest(t, app, http.MethodPost, "/api/auth/login", map[string]string{
		"username": "reader",
		"password": "secret",
	})
	readerToken, ok := readerLogin.Data["token"].(string)
	if !ok || readerToken == "" {
		t.Fatalf("reader token = %#v, want non-empty string", readerLogin.Data["token"])
	}
	forbidden := doJSONRequest(t, app, http.MethodPost, "/api/user/update?id="+strconv.FormatInt(readerId, 10), map[string]string{
		"nickName": "blocked",
	}, readerToken)
	if forbidden.HTTPStatus != http.StatusOK || forbidden.Code == 0 {
		t.Fatalf("reader update response = %#v, want HTTP 200 with business error", forbidden)
	}
}
