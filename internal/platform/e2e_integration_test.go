package platform_test

import (
	"context"
	"testing"
	"time"

	"github.com/user/vhd-opener/internal/capabilities/search"
	"github.com/user/vhd-opener/internal/platform"
	"github.com/user/vhd-opener/internal/ucl"
	"github.com/user/vhd-opener/pkg/capability"
)

func TestE2EPipeline_UCLToGatewayToEventBus(t *testing.T) {
	eventBus := platform.NewEventBus()
	jobManager := platform.NewJobManager(eventBus)
	gateway := platform.NewGateway(eventBus, jobManager)

	vfsCap := search.NewVFSSearchCapability()
	gateway.RegisterCapability(vfsCap)

	startedCh := eventBus.Subscribe("JOB_STARTED")
	progressCh := eventBus.Subscribe("JOB_PROGRESS")
	completedCh := eventBus.Subscribe("COMPLETED")

	rawUCL := "FIND WHERE extension=pdf AND size>10MB LIMIT 10"
	ast, err := ucl.Parse(rawUCL)
	if err != nil {
		t.Fatalf("UCL parsing failed: %v", err)
	}

	if ast.CapabilityID != "cap.disk.search" {
		t.Fatalf("Expected CapabilityID cap.disk.search, got: %s", ast.CapabilityID)
	}

	execCtx := capability.ExecutionContext{
		SessionID:       "test_session_01",
		ActivePartition: "Partition 1 (NTFS)",
		CurrentPath:     "/Windows",
		Params:          ast.ToParamMap(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	job, err := gateway.Dispatch(ctx, ast.CapabilityID, execCtx)
	if err != nil {
		t.Fatalf("Gateway dispatch failed: %v", err)
	}

	if job.ID == "" {
		t.Fatal("Expected non-empty Job ID from Gateway dispatch")
	}

	select {
	case event := <-startedCh:
		payload, ok := event.Payload.(map[string]any)
		if !ok {
			t.Fatalf("Invalid JOB_STARTED event payload format")
		}
		if payload["job_id"] != job.ID {
			t.Errorf("Expected job_id %s, got %v", job.ID, payload["job_id"])
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Timed out waiting for JOB_STARTED event")
	}

	select {
	case event := <-progressCh:
		payload, ok := event.Payload.(map[string]any)
		if !ok {
			t.Fatalf("Invalid JOB_PROGRESS event payload format")
		}
		progressVal, isFloat := payload["progress"].(float64)
		if !isFloat || progressVal <= 0.0 {
			t.Errorf("Expected progress > 0.0, got %v", payload["progress"])
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Timed out waiting for JOB_PROGRESS event emission")
	}

	select {
	case event := <-completedCh:
		payload, ok := event.Payload.(map[string]any)
		if !ok {
			t.Fatalf("Invalid COMPLETED event payload format")
		}

		if payload["job_id"] != job.ID {
			t.Errorf("Expected job_id %s in completion event, got %v", job.ID, payload["job_id"])
		}

		results, ok := payload["result"].([]search.FileMatch)
		if !ok {
			t.Fatalf("Expected []search.FileMatch result payload, got %T", payload["result"])
		}

		if len(results) == 0 {
			t.Errorf("Expected non-empty search results from VFS execution")
		}
	case <-time.After(4 * time.Second):
		t.Fatal("Timed out waiting for COMPLETED event")
	}
}

func TestE2EPipeline_JobCancellation(t *testing.T) {
	eventBus := platform.NewEventBus()
	jobManager := platform.NewJobManager(eventBus)
	gateway := platform.NewGateway(eventBus, jobManager)

	vfsCap := search.NewVFSSearchCapability()
	gateway.RegisterCapability(vfsCap)

	canceledCh := eventBus.Subscribe("CANCELED")

	execCtx := capability.ExecutionContext{
		SessionID:       "test_session_cancel",
		ActivePartition: "Partition 1",
		CurrentPath:     "/",
		Params:          map[string]any{"extension": "pdf"},
	}

	job, err := gateway.Dispatch(context.Background(), "cap.disk.search", execCtx)
	if err != nil {
		t.Fatalf("Dispatch failed: %v", err)
	}

	time.Sleep(100 * time.Millisecond)
	success := jobManager.Cancel(job.ID)
	if !success {
		t.Fatalf("Job cancellation call returned false")
	}

	select {
	case event := <-canceledCh:
		payload, ok := event.Payload.(map[string]any)
		if !ok || payload["job_id"] != job.ID {
			t.Errorf("Expected cancellation event for job %s, got payload %v", job.ID, event.Payload)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Timed out waiting for CANCELED event")
	}
}
