package sys_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"handfree-work/octo-backup/internal/base/db_"
	"handfree-work/octo-backup/internal/base/web_"
	"handfree-work/octo-backup/internal/handler"
	"handfree-work/octo-backup/internal/models"
	"handfree-work/octo-backup/internal/svc"

	"github.com/gofiber/fiber/v3"
)

func TestUserCRUD(t *testing.T) {
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
	registered := doJSONRequest(t, app, http.MethodPost, "/api/auth/register", map[string]string{
		"username": "admin",
		"password": "secret",
	})
	if registered.Code != 0 {
		t.Fatalf("register business code = %d, want 0", registered.Code)
	}
	login := doJSONRequest(t, app, http.MethodPost, "/api/auth/login", map[string]string{
		"username": "admin",
		"password": "secret",
	})
	token := login.Data["token"].(string)

	created := doJSONRequest(t, app, http.MethodPost, "/api/user/create", map[string]string{
		"username": "alice",
		"password": "secret",
		"nickName": "Alice",
	}, token)
	if created.Code != 0 {
		t.Fatalf("create business code = %d, want 0", created.Code)
	}
	if created.Data["username"] != "alice" {
		t.Fatalf("created username = %#v, want alice", created.Data["username"])
	}
	if _, exists := created.Data["password"]; exists {
		t.Fatal("created response must not contain password")
	}
	userId := int64(created.Data["id"].(float64))

	list := doJSONRequest(t, app, http.MethodPost, "/api/user/list", map[string]any{}, token)
	if list.Code != 0 {
		t.Fatalf("list business code = %d, want 0", list.Code)
	}
	users, ok := list.Data["records"].([]any)
	if !ok || len(users) != 2 {
		t.Fatalf("list items = %#v, want two users", list.Data["items"])
	}

	found := doJSONRequest(t, app, http.MethodPost, "/api/user/info?id="+fmt.Sprint(userId), nil, token)
	if found.Code != 0 {
		t.Fatalf("get business code = %d, want 0", found.Code)
	}

	updated := doJSONRequest(t, app, http.MethodPost, "/api/user/update?id="+fmt.Sprint(userId), map[string]string{
		"nickName": "Alice Updated",
	}, token)
	if updated.Code != 0 {
		t.Fatalf("update business code = %d, want 0", updated.Code)
	}
	if updated.Data["nickName"] != "Alice Updated" {
		t.Fatalf("updated nickname = %#v, want Alice Updated", updated.Data["nickName"])
	}

	deleted := doJSONRequest(t, app, http.MethodPost, "/api/user/delete?id="+fmt.Sprint(userId), nil, token)
	if deleted.Code != 0 {
		t.Fatalf("delete business code = %d, want 0", deleted.Code)
	}

	notFound := doJSONRequest(t, app, http.MethodPost, "/api/user/info?id="+fmt.Sprint(userId), nil, token)
	if notFound.Code == 0 {
		t.Fatalf("get deleted user response = %#v, want business error", notFound)
	}

	invalid := doJSONRequest(t, app, http.MethodPost, "/api/user/create", map[string]string{
		"password": "secret",
	}, token)
	if invalid.Code == 0 {
		t.Fatalf("invalid create response = %#v, want business error", invalid)
	}

}

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
