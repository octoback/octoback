package basic

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"handfree-work/octo-backup/internal/base/task_"
	"handfree-work/octo-backup/internal/base/web_"

	"github.com/gofiber/fiber/v3"
)

func TestTaskStatusReturnsTaskByTaskId(t *testing.T) {
	executorTask := task_.NewTask("test", "status", "测试任务", func(*task_.Task) error { return nil })
	if err := task_.DefaultExecutor.Start(executorTask); err != nil {
		t.Fatal(err)
	}
	app := fiber.New()
	app.Post("/api/task/status", web_.Handle(taskStatus))
	request := httptest.NewRequest("POST", "/api/task/status?taskId="+executorTask.TaskId, nil)
	response, err := app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var result struct {
		Code int `json:"code"`
		Data struct {
			TaskId string `json:"taskId"`
		} `json:"data"`
	}
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if result.Code != 0 || result.Data.TaskId != executorTask.TaskId {
		t.Fatalf("unexpected response: %#v", result)
	}
}
