package task_

import (
	"errors"
	"testing"
	"time"
)

func TestExecutorRunsTaskAndPublishesProgress(t *testing.T) {
	executor := NewExecutor(30 * time.Second)
	done := make(chan struct{})
	task := NewTask("repository-check", "1", "检查仓库", func(task *Task) error {
		task.SetTotal(2)
		task.IncrementCurrent()
		task.IncrementCurrent()
		close(done)
		return nil
	})
	if err := executor.Start(task); err != nil {
		t.Fatal(err)
	}
	<-done
	status, ok := executor.Get("repository-check", "1")
	if !ok || status.Status != StatusDone || status.Progress != 100 || status.SuccessCount != 2 {
		t.Fatalf("unexpected task status: %#v", status)
	}
	byId, ok := executor.GetById(status.TaskId)
	if !ok || byId.TaskId != status.TaskId {
		t.Fatalf("task id lookup failed: %#v", byId)
	}
}

func TestExecutorRejectsConcurrentTaskAndRecordsFailure(t *testing.T) {
	executor := NewExecutor(time.Minute)
	started := make(chan struct{})
	release := make(chan struct{})
	task := NewTask("check", "same", "检查", func(task *Task) error {
		close(started)
		<-release
		return errors.New("连接失败")
	})
	if err := executor.Start(task); err != nil {
		t.Fatal(err)
	}
	<-started
	if err := executor.Start(NewTask("check", "same", "检查", func(*Task) error { return nil })); err == nil {
		t.Fatal("expected concurrent start to fail")
	}
	close(release)
	for i := 0; i < 20; i++ {
		status, _ := executor.Get("check", "same")
		if status.Status == StatusFailed {
			if len(status.Errors) != 1 || status.Errors[0] != "连接失败" {
				t.Fatalf("unexpected errors: %#v", status.Errors)
			}
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("task did not fail")
}
