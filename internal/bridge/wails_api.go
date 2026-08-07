package bridge

import (
	"context"
	"fmt"
	"log"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/user/vhd-opener/internal/platform"
	"github.com/user/vhd-opener/internal/ucl"
	"github.com/user/vhd-opener/pkg/capability"
)

type WailsBridge struct {
	gateway          *platform.Gateway
	eventBus         *platform.EventBus
	jobManager       *platform.JobManager
	workspaceManager *platform.WorkspaceManager
	ctx              context.Context
}

func NewWailsBridge(
	gw *platform.Gateway,
	eb *platform.EventBus,
	jm *platform.JobManager,
	wm *platform.WorkspaceManager,
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
	progressCh := b.eventBus.Subscribe("JOB_PROGRESS")
	completedCh := b.eventBus.Subscribe("COMPLETED")
	failedCh := b.eventBus.Subscribe("FAILED")
	workspaceCh := b.eventBus.Subscribe("WORKSPACE_TARGET_MOUNTED")

	for {
		select {
		case <-b.ctx.Done():
			return
		case event := <-progressCh:
			runtime.EventsEmit(b.ctx, "JOB_PROGRESS", event.Payload)
		case event := <-completedCh:
			runtime.EventsEmit(b.ctx, "JOB_COMPLETED", event.Payload)
		case event := <-failedCh:
			runtime.EventsEmit(b.ctx, "JOB_FAILED", event.Payload)
		case <-workspaceCh:
			runtime.EventsEmit(b.ctx, "WORKSPACE_UPDATED", b.workspaceManager.GetState())
		}
	}
}

func (b *WailsBridge) GetWorkspaceState() platform.WorkspaceState {
	return b.workspaceManager.GetState()
}

func (b *WailsBridge) MountDiskTarget(target platform.WorkspaceTarget) {
	b.workspaceManager.MountTarget(target)
}

func (b *WailsBridge) UnmountDiskTarget(targetID string) {
	b.workspaceManager.UnmountTarget(targetID)
}

func (b *WailsBridge) SetActiveTarget(targetID string) {
	b.workspaceManager.SetActiveTarget(targetID)
}

func (b *WailsBridge) SetActivePartition(partition string) {
	b.workspaceManager.SetActivePartition(partition)
}

func (b *WailsBridge) NavigateWorkspace(path string) {
	b.workspaceManager.Navigate(path)
}

func (b *WailsBridge) OpenTab(path string) {
	b.workspaceManager.OpenTab(path)
}

func (b *WailsBridge) CloseTab(index int) {
	b.workspaceManager.CloseTab(index)
}

func (b *WailsBridge) SetActiveTab(index int) {
	b.workspaceManager.SetActiveTab(index)
}

func (b *WailsBridge) AddBookmark(label, path string) *platform.Bookmark {
	return b.workspaceManager.AddBookmark(label, path)
}

func (b *WailsBridge) RemoveBookmark(bookmarkID string) {
	b.workspaceManager.RemoveBookmark(bookmarkID)
}

func (b *WailsBridge) GetBookmarks() []platform.Bookmark {
	return b.workspaceManager.GetBookmarks()
}

func (b *WailsBridge) ExecuteCapability(capabilityID string, params map[string]any) (string, error) {
	state := b.workspaceManager.GetState()
	execCtx := capability.ExecutionContext{
		SessionID:       state.ID,
		ActivePartition: state.ActivePartition,
		CurrentPath:     "/",
		Params:          params,
	}

	job, err := b.gateway.Dispatch(b.ctx, capabilityID, execCtx)
	if err != nil {
		return "", fmt.Errorf("dispatch error: %w", err)
	}

	return job.ID, nil
}

func (b *WailsBridge) ExecuteUCL(rawQuery string) (string, error) {
	ast, err := ucl.Parse(rawQuery)
	if err != nil {
		return "", fmt.Errorf("ucl parse error: %w", err)
	}

	state := b.workspaceManager.GetState()
	execCtx := capability.ExecutionContext{
		SessionID:       state.ID,
		ActivePartition: state.ActivePartition,
		CurrentPath:     "/",
		Params:          ast.ToParamMap(),
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
