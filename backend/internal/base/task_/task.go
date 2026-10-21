package task_

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
	"handfree-work/octo-backup/internal/base/log_"
)

// Status 描述内存任务的生命周期状态。
type Status string

const (
	StatusPending  Status = "pending"
	StatusRunning  Status = "running"
	StatusDone     Status = "done"
	StatusFailed   Status = "failed"
	StatusCanceled Status = "canceled"
)

// TaskFunc 执行一项后台任务，并通过 Task 更新进度。
type TaskFunc func(task *Task) error

// Task 保存任务状态和可查询的进度信息。
type Task struct {
	TaskId       string        `json:"taskId"`
	Type         string        `json:"type"`
	Key          string        `json:"key"`
	Title        string        `json:"title"`
	Status       Status        `json:"status"`
	Total        int           `json:"total"`
	Current      int           `json:"current"`
	SkipCount    int           `json:"skipCount"`
	Progress     float64       `json:"progress"`
	SuccessCount int           `json:"successCount"`
	ErrorCount   int           `json:"errorCount"`
	Errors       []string      `json:"errors,omitempty"`
	StartTime    time.Time     `json:"startTime,omitempty"`
	EndTime      time.Time     `json:"endTime,omitempty"`
	Timeout      time.Duration `json:"-"`

	mu     sync.RWMutex
	run    TaskFunc
	ctx    context.Context
	cancel context.CancelFunc
}

var taskSequence uint64

// NewTask 创建一项处于待执行状态的任务。
func NewTask(taskType, key, title string, run TaskFunc) *Task {
	sequence := atomic.AddUint64(&taskSequence, 1)
	taskId := fmt.Sprintf("%s-%s-%d", taskType, key, sequence)
	return &Task{TaskId: taskId, Type: taskType, Key: key, Title: title, Status: StatusPending, Timeout: 30 * time.Minute, run: run}
}

// Context 返回任务的可取消执行上下文。
func (t *Task) Context() context.Context {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if t.ctx == nil {
		return context.Background()
	}
	return t.ctx
}

// Cancel 请求取消任务；底层操作需使用 Context 响应取消。
func (t *Task) Cancel() {
	t.mu.Lock()
	if t.cancel != nil {
		t.cancel()
	}
	t.mu.Unlock()
}

// SetTotal 设置任务总数并刷新进度。
func (t *Task) SetTotal(total int) { t.mu.Lock(); defer t.mu.Unlock(); t.Total = total; t.refresh() }

// IncrementCurrent 标记一项任务处理完成并刷新进度。
func (t *Task) IncrementCurrent() { t.mu.Lock(); defer t.mu.Unlock(); t.Current++; t.refresh() }

// IncrementSkipCount 标记一项任务被跳过。
func (t *Task) IncrementSkipCount() { t.mu.Lock(); defer t.mu.Unlock(); t.SkipCount++ }

// AddError 记录一项可展示的任务错误。
func (t *Task) AddError(message string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.Errors = append(t.Errors, message)
	t.ErrorCount = len(t.Errors)
}

func (t *Task) refresh() {
	if t.Total > 0 {
		t.Progress = float64(t.Current) * 100 / float64(t.Total)
		if t.Progress > 100 {
			t.Progress = 100
		}
	}
	t.SuccessCount = t.Current
}

func (t *Task) snapshot() *Task {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return &Task{
		TaskId: t.TaskId, Type: t.Type, Key: t.Key, Title: t.Title, Status: t.Status,
		Total: t.Total, Current: t.Current, SkipCount: t.SkipCount, Progress: t.Progress,
		SuccessCount: t.SuccessCount, ErrorCount: t.ErrorCount,
		Errors: append([]string(nil), t.Errors...), StartTime: t.StartTime, EndTime: t.EndTime,
	}
}

// Executor 管理按类型和键互斥的内存后台任务。
type Executor struct {
	mu        sync.RWMutex
	tasks     map[string]*Task
	byId      map[string]*Task
	retention time.Duration
}

// NewExecutor 创建任务执行器，完成任务将在 retention 后自动清理。
func NewExecutor(retention time.Duration) *Executor {
	if retention <= 0 {
		retention = 24 * time.Hour
	}
	return &Executor{tasks: make(map[string]*Task), byId: make(map[string]*Task), retention: retention}
}

func taskKey(taskType, key string) string { return taskType + "\x00" + key }

// Start 注册并异步执行任务；同一类型和键的运行中任务不能重复启动。
func (e *Executor) Start(task *Task) error {
	if task == nil || task.run == nil {
		return fmt.Errorf("任务不能为空")
	}
	k := taskKey(task.Type, task.Key)
	e.mu.Lock()
	if old := e.tasks[k]; old != nil {
		old.mu.RLock()
		running := old.Status == StatusRunning
		old.mu.RUnlock()
		if running {
			e.mu.Unlock()
			return fmt.Errorf("任务 %s[%s] 正在运行中", task.Type, task.Key)
		}
		if old == task {
			e.mu.Unlock()
			return fmt.Errorf("任务不能重复启动")
		}
	}
	task.mu.RLock()
	pending := task.Status == StatusPending
	task.mu.RUnlock()
	if !pending {
		e.mu.Unlock()
		return fmt.Errorf("任务不能重复启动")
	}
	task.mu.Lock()
	task.Status = StatusRunning
	task.StartTime = time.Now()
	task.ctx, task.cancel = context.WithTimeout(context.Background(), task.Timeout)
	task.mu.Unlock()
	e.tasks[k] = task
	e.byId[task.TaskId] = task
	e.mu.Unlock()
	go e.run(k, task)
	return nil
}

func (e *Executor) run(k string, task *Task) {
	var err error
	func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				err = fmt.Errorf("任务执行异常: %v", recovered)
				log_.Logger.Error("后台任务异常", zap.String("type", task.Type), zap.String("key", task.Key), zap.Any("panic", recovered))
			}
		}()
		err = task.run(task)
	}()
	ctxErr := task.Context().Err()
	task.mu.Lock()
	if err != nil {
		task.Status = StatusFailed
		task.Errors = append(task.Errors, err.Error())
		task.ErrorCount = len(task.Errors)
	}
	if ctxErr != nil && err == nil {
		task.Status = StatusCanceled
		task.Errors = append(task.Errors, ctxErr.Error())
		task.ErrorCount = len(task.Errors)
	}
	if err == nil && ctxErr == nil {
		task.Status = StatusDone
	}
	task.EndTime = time.Now()
	task.refresh()
	task.mu.Unlock()
	time.AfterFunc(e.retention, func() { e.clearIfSame(task.Type, task.Key, task) })
}

func (e *Executor) clearIfSame(taskType, key string, expected *Task) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.tasks[taskKey(taskType, key)] == expected {
		delete(e.tasks, taskKey(taskType, key))
		delete(e.byId, expected.TaskId)
	}
}

// Get 返回任务状态快照。
func (e *Executor) Get(taskType, key string) (*Task, bool) {
	e.mu.RLock()
	task := e.tasks[taskKey(taskType, key)]
	e.mu.RUnlock()
	if task == nil {
		return nil, false
	}
	return task.snapshot(), true
}

// GetById 返回任务标识对应的状态快照。
func (e *Executor) GetById(taskId string) (*Task, bool) {
	e.mu.RLock()
	task := e.byId[taskId]
	e.mu.RUnlock()
	if task == nil {
		return nil, false
	}
	return task.snapshot(), true
}

// CancelById 请求取消指定任务。
func (e *Executor) CancelById(taskId string) bool {
	e.mu.RLock()
	task := e.byId[taskId]
	e.mu.RUnlock()
	if task == nil {
		return false
	}
	task.Cancel()
	return true
}

// Clear 删除指定任务及其延迟清理计时器之外的内存记录。
func (e *Executor) Clear(taskType, key string) {
	e.mu.Lock()
	delete(e.tasks, taskKey(taskType, key))
	for id, task := range e.byId {
		if task.Type == taskType && task.Key == key {
			delete(e.byId, id)
		}
	}
	e.mu.Unlock()
}

// RemoveIfEnded 删除已结束任务，运行中任务保持不变。
func (e *Executor) RemoveIfEnded(taskType, key string) {
	if task, ok := e.Get(taskType, key); ok && task.Status != StatusRunning {
		e.Clear(taskType, key)
	}
}

// DefaultExecutor 是进程级共享任务执行器。
var DefaultExecutor = NewExecutor(24 * time.Hour)
