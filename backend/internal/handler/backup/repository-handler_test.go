package backup

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"handfree-work/octo-backup/internal/base/web_"
	"handfree-work/octo-backup/internal/svc"

	"github.com/gofiber/fiber/v3"
)

func TestRegisterRepositoryRoutesRequiresAuthentication(t *testing.T) {
	app := fiber.New()
	Register(app, &svc.ServiceContext{Auth: web_.Config{Secret: "test-secret"}})

	paths := []string{
		"/api/plugin/instance/snapshots",
		"/api/plugin/instance/stats",
		"/api/plugin/instance/snapshot-stats",
		"/api/plugin/instance/refresh-repository-data",
		"/api/plugin/instance/check",
		"/api/plugin/instance/snapshot-browser",
		"/api/plugin/instance/forget-snapshot",
	}
	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			request := httptest.NewRequest("POST", path, nil)
			response, err := app.Test(request)
			if err != nil {
				t.Fatalf("POST %s: %v", path, err)
			}
			defer response.Body.Close()

			var result struct {
				Code int `json:"code"`
			}
			if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
				t.Fatalf("decode response for %s: %v", path, err)
			}
			if response.StatusCode != fiber.StatusOK || result.Code != fiber.StatusUnauthorized {
				t.Fatalf("POST %s status/code = %d/%d, want %d/%d", path, response.StatusCode, result.Code, fiber.StatusOK, fiber.StatusUnauthorized)
			}
		})
	}
}
