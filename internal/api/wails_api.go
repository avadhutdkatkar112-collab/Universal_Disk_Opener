package api

import (
	"context"
	"fmt"
	"log"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	appservices "github.com/user/vhd-opener/internal/app/services"
	"github.com/user/vhd-opener/internal/jobs"
)

type WailsBridge struct {
	gateway          *appservices.Gateway
	eventBus         *appservices.EventBus
	jobManager       *appservices.JobManager
	workspaceManager *appservices.Workspace
	ctx              context.Context
}

func NewWailsBridge(
	gw *appservices.Gateway,
	eb *appservices.EventBus,
	jm *appservices.JobManager,
	wm *appservices.Workspace,
) *WailsBridge {
	return &WailsBridge{
		gateway:          gw,
		eventBus:         eb,
		jobManager:       jm,
		workspaceManager: wm,
	}
}

func (b *WailsBridge) Startup(ctx context.Context) {
	b.ctx = ctx
	go b.startEventBridge()
	log.Println("[WailsBridge] IPC Bridge online with Workspace & Event streaming.")
}

func (b *WailsBridge) startEventBridge() {
	startedCh := b.eventBus.Subscribe("JOB_STARTED")
	progressCh := b.eventBus.Subscribe("JOB_PROGRESS")
	completedCh := b.eventBus.Subscribe("JOB_COMPLETED")
	failedCh := b.eventBus.Subscribe("JOB_FAILED")

	for {
		select {
		case event := <-startedCh:
			runtime.EventsEmit(b.ctx, "JOB_STARTED", event.Payload)
		case event := <-progressCh:
			runtime.EventsEmit(b.ctx, "JOB_PROGRESS", event.Payload)
		case event := <-completedCh:
			runtime.EventsEmit(b.ctx, "JOB_COMPLETED", event.Payload)
		case event := <-failedCh:
			runtime.EventsEmit(b.ctx, "JOB_FAILED", event.Payload)
		}
	}
}

func (b *WailsBridge) GetWorkspaceState() map[string]any {
	return b.workspaceManager.GetState()
}

func (b *WailsBridge) MountDiskTarget(target map[string]any) {
	b.workspaceManager.MountTarget(target)
}

func (b *WailsBridge) UnmountDiskTarget(targetID string) {
	b.workspaceManager.UnmountTarget(targetID)
}

func (b *WailsBridge) SetActiveTarget(targetID string) {
	b.workspaceManager.SetActiveTarget(targetID)
}

func (b *WailsBridge) SetActivePartition(partition string) {}

func (b *WailsBridge) NavigateWorkspace(path string) {}

func (b *WailsBridge) OpenTab(path string) {}

func (b *WailsBridge) CloseTab(index int) {}

func (b *WailsBridge) SetActiveTab(index int) {}

func (b *WailsBridge) AddBookmark(label, path string) *appservices.Bookmark {
	return b.workspaceManager.AddBookmark("", label, path, "")
}

func (b *WailsBridge) RemoveBookmark(bookmarkID string) {}

func (b *WailsBridge) GetBookmarks() []appservices.Bookmark {
	return nil
}

func (b *WailsBridge) ExecuteCapability(capabilityID string, params map[string]any) (string, error) {
	execCtx := jobs.ExecutionContext{
		Params: params,
	}

	job, err := b.gateway.Dispatch(b.ctx, capabilityID, execCtx)
	if err != nil {
		return "", fmt.Errorf("dispatch error: %w", err)
	}

	return job.ID, nil
}

func (b *WailsBridge) ExecuteUCL(rawQuery string) (string, error) {
	ast, err := appservices.Parse(rawQuery)
	if err != nil {
		return "", fmt.Errorf("ucl parse error: %w", err)
	}

	execCtx := jobs.ExecutionContext{
		Params: ast.ToParamMap(),
	}

	job, err := b.gateway.Dispatch(b.ctx, ast.CapabilityID, execCtx)
	if err != nil {
		return "", fmt.Errorf("ucl dispatch error: %w", err)
	}

	return job.ID, nil
}

func (b *WailsBridge) CancelJob(jobID string) bool {
	return b.jobManager.Cancel(jobID)
}

func (b *WailsBridge) GetJobStatus(jobID string) (map[string]any, bool) {
	return b.jobManager.GetSnapshot(jobID)
}
