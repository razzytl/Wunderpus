package builtin

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/wunderpus/wunderpus/internal/tool"
)

type CronCallback func(jobID, message string)

type CronTool struct {
	mu             sync.RWMutex
	jobs           map[string]*cronJob
	defaultChannel string
	defaultChatID  string
	callback       CronCallback
	stopCh         chan struct{}
	wg             sync.WaitGroup
}

type cronJob struct {
	ID       string
	Message  string
	Channel  string
	ChatID   string
	NextRun  time.Time
	Schedule cronSchedule
	Enabled  bool
}

type cronSchedule struct {
	Kind     string // "once", "every", "cron"
	AtMS     int64  // for "once"
	EverySec int64  // for "every"
	Expr     string // for "cron"
}

func NewCronTool() *CronTool {
	return &CronTool{
		jobs:   make(map[string]*cronJob),
		stopCh: make(chan struct{}),
	}
}

func (t *CronTool) Name() string { return "cron" }
func (t *CronTool) Description() string {
	return "Schedule reminders, tasks, or commands. Use action='add' for one-time or recurring tasks. Use 'at_seconds' for reminders in X seconds. Use 'every_seconds' for recurring tasks."
}
func (t *CronTool) Sensitive() bool                   { return false }
func (t *CronTool) ApprovalLevel() tool.ApprovalLevel { return tool.AutoExecute }
func (t *CronTool) Version() string                   { return "1.0.0" }
func (t *CronTool) Dependencies() []string            { return nil }
func (t *CronTool) Parameters() []tool.ParameterDef {
	return []tool.ParameterDef{
		{Name: "action", Type: "string", Description: "Action: add, list, remove, enable, disable", Required: true},
		{Name: "message", Type: "string", Description: "Reminder message", Required: false},
		{Name: "at_seconds", Type: "number", Description: "One-time reminder: seconds from now (e.g., 600 for 10 minutes)", Required: false},
		{Name: "every_seconds", Type: "number", Description: "Recurring interval in seconds (e.g., 3600 for every hour)", Required: false},
		{Name: "job_id", Type: "string", Description: "Job ID for remove/enable/disable", Required: false},
	}
}

func (t *CronTool) SetContext(channel, chatID string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.defaultChannel = channel
	t.defaultChatID = chatID
}

func (t *CronTool) SetCallback(callback CronCallback) {
	t.callback = callback
}

func (t *CronTool) Execute(ctx context.Context, args map[string]any) (*tool.Result, error) {
	action, _ := args["action"].(string)
	if action == "" {
		return &tool.Result{Error: "action is required"}, nil
	}

	switch action {
	case "add":
		return t.addJob(args), nil
	case "list":
		return t.listJobs(), nil
	case "remove":
		return t.removeJob(args), nil
	case "enable":
		return t.enableJob(args, true), nil
	case "disable":
		return t.enableJob(args, false), nil
	default:
		return &tool.Result{Error: fmt.Sprintf("unknown action: %s", action)}, nil
	}
}

func (t *CronTool) addJob(args map[string]any) *tool.Result {
	t.mu.RLock()
	channel := t.defaultChannel
	chatID := t.defaultChatID
	t.mu.RUnlock()

	message, _ := args["message"].(string)
	atSeconds, _ := args["at_seconds"].(float64)
	everySeconds, _ := args["every_seconds"].(float64)

	if message == "" {
		return &tool.Result{Error: "message is required"}
	}

	now := time.Now()
	schedule := cronSchedule{}

	if atSeconds > 0 {
		schedule = cronSchedule{
			Kind: "once",
			AtMS: now.Add(time.Duration(atSeconds) * time.Second).UnixMilli(),
		}
	} else if everySeconds > 0 {
		schedule = cronSchedule{
			Kind:     "every",
			EverySec: int64(everySeconds),
		}
	} else {
		return &tool.Result{Error: "at_seconds or every_seconds is required"}
	}

	job := &cronJob{
		ID:       fmt.Sprintf("cron-%d", now.UnixNano()),
		Message:  message,
		Channel:  channel,
		ChatID:   chatID,
		Schedule: schedule,
		Enabled:  true,
	}

	if schedule.Kind == "once" {
		job.NextRun = time.UnixMilli(schedule.AtMS)
	} else {
		job.NextRun = now.Add(time.Duration(schedule.EverySec) * time.Second)
	}

	t.mu.Lock()
	t.jobs[job.ID] = job
	t.mu.Unlock()

	return &tool.Result{Output: fmt.Sprintf("Cron job added: %s (%s)", job.ID, job.Message)}
}

func (t *CronTool) listJobs() *tool.Result {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if len(t.jobs) == 0 {
		return &tool.Result{Output: "No scheduled jobs"}
	}

	var lines []string
	for id, job := range t.jobs {
		schedule := "unknown"
		if job.Schedule.Kind == "once" {
			schedule = fmt.Sprintf("at %s", job.NextRun.Format(time.RFC3339))
		} else if job.Schedule.Kind == "every" {
			schedule = fmt.Sprintf("every %ds", job.Schedule.EverySec)
		}
		status := "enabled"
		if !job.Enabled {
			status = "disabled"
		}
		lines = append(lines, fmt.Sprintf("- %s: %s (%s) [%s]", id, job.Message, schedule, status))
	}

	return &tool.Result{Output: "Scheduled jobs:\n" + join(lines, "\n")}
}

func (t *CronTool) removeJob(args map[string]any) *tool.Result {
	jobID, _ := args["job_id"].(string)
	if jobID == "" {
		return &tool.Result{Error: "job_id is required"}
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	if _, ok := t.jobs[jobID]; !ok {
		return &tool.Result{Error: fmt.Sprintf("job %s not found", jobID)}
	}

	delete(t.jobs, jobID)
	return &tool.Result{Output: fmt.Sprintf("Job removed: %s", jobID)}
}

func (t *CronTool) enableJob(args map[string]any, enable bool) *tool.Result {
	jobID, _ := args["job_id"].(string)
	if jobID == "" {
		return &tool.Result{Error: "job_id is required"}
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	job, ok := t.jobs[jobID]
	if !ok {
		return &tool.Result{Error: fmt.Sprintf("job %s not found", jobID)}
	}

	job.Enabled = enable
	status := "enabled"
	if !enable {
		status = "disabled"
	}
	return &tool.Result{Output: fmt.Sprintf("Job %s %s", jobID, status)}
}

func (t *CronTool) Start(ctx context.Context) {
	t.wg.Add(1)
	go func() {
		defer t.wg.Done()
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				t.checkAndRunJobs(ctx)
			case <-t.stopCh:
				return
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (t *CronTool) Stop() {
	close(t.stopCh)
	t.wg.Wait()
}

func (t *CronTool) checkAndRunJobs(ctx context.Context) {
	now := time.Now()

	t.mu.Lock()
	defer t.mu.Unlock()

	for id, job := range t.jobs {
		if !job.Enabled {
			continue
		}

		if now.After(job.NextRun) {
			if t.callback != nil {
				t.callback(id, job.Message)
			}

			if job.Schedule.Kind == "once" {
				delete(t.jobs, id)
			} else if job.Schedule.Kind == "every" {
				job.NextRun = now.Add(time.Duration(job.Schedule.EverySec) * time.Second)
			}
		}
	}
}

func join(elems []string, sep string) string {
	result := ""
	for i, e := range elems {
		if i > 0 {
			result += sep
		}
		result += e
	}
	return result
}
