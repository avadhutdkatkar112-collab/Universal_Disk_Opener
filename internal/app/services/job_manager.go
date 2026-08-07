package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/user/vhd-opener/internal/jobs"
)

type JobState string

const (
	StateQueued    JobState = "QUEUED"
	StateRunning   JobState = "RUNNING"
	StateCompleted JobState = "COMPLETED"
	StateFailed    JobState = "FAILED"
	StateCanceled  JobState = "CANCELED"
)

type Job struct {
	ID         string                   `json:"id"`
	Capability jobs.Capability          `json:"-"`
	Context    jobs.ExecutionContext     `json:"context"`
	State      JobState                 `json:"state"`
	Progress   float64                  `json:"progress"`
	Result     any                      `json:"result,omitempty"`
	Error      error                    `json:"error,omitempty"`
	CreatedAt  time.Time                `json:"created_at"`
	cancelFunc context.CancelFunc
	mu         sync.Mutex
}

type JobManager struct {
	mu       sync.RWMutex
	jobs     map[string]*Job
	eventBus *EventBus
}

func NewJobManager(eb *EventBus) *JobManager {
	return &JobManager{
		jobs:     make(map[string]*Job),
		eventBus: eb,
	}
}

func (jm *JobManager) Submit(parentCtx context.Context, cap jobs.Capability, execCtx jobs.ExecutionContext) *Job {
	ctx, cancel := context.WithCancel(parentCtx)

	job := &Job{
		ID:         fmt.Sprintf("job_%d", time.Now().UnixNano()),
		Capability: cap,
		Context:    execCtx,
		State:      StateQueued,
		CreatedAt:  time.Now(),
		cancelFunc: cancel,
	}

	jm.mu.Lock()
	jm.jobs[job.ID] = job
	jm.mu.Unlock()

	go func() {
		job.mu.Lock()
		job.State = StateRunning
		job.mu.Unlock()

		jm.eventBus.Publish(Event{
			Type:      "JOB_STARTED",
			Source:    "JobManager",
			Payload:   map[string]any{"job_id": job.ID, "capability": cap.Metadata().ID},
			Timestamp: time.Now(),
		})

		progressChan := make(chan float64, 50)

		go func() {
			for p := range progressChan {
				job.mu.Lock()
				job.Progress = p
				job.mu.Unlock()

				jm.eventBus.Publish(Event{
					Type:      "JOB_PROGRESS",
					Source:    "JobManager",
					Payload:   map[string]any{"job_id": job.ID, "progress": p},
					Timestamp: time.Now(),
				})
			}
		}()

		res, err := cap.Execute(ctx, execCtx, progressChan)
		close(progressChan)

		job.mu.Lock()
		if err != nil {
			if ctx.Err() == context.Canceled {
				job.State = StateCanceled
			} else {
				job.State = StateFailed
				job.Error = err
			}
		} else {
			job.State = StateCompleted
			job.Progress = 100.0
			job.Result = res
		}
		job.mu.Unlock()

		jm.eventBus.Publish(Event{
			Type:      string(job.State),
			Source:    "JobManager",
			Payload:   map[string]any{"job_id": job.ID, "result": res, "error": err},
			Timestamp: time.Now(),
		})
	}()

	return job
}

func (jm *JobManager) Cancel(jobID string) bool {
	jm.mu.RLock()
	job, exists := jm.jobs[jobID]
	jm.mu.RUnlock()

	if !exists {
		return false
	}

	job.cancelFunc()
	return true
}

func (jm *JobManager) Get(jobID string) (*Job, bool) {
	jm.mu.RLock()
	defer jm.mu.RUnlock()
	job, ok := jm.jobs[jobID]
	return job, ok
}

func (jm *JobManager) GetSnapshot(jobID string) (map[string]any, bool) {
	jm.mu.RLock()
	job, ok := jm.jobs[jobID]
	jm.mu.RUnlock()
	if !ok {
		return nil, false
	}
	job.mu.Lock()
	defer job.mu.Unlock()
	return map[string]any{
		"id":         job.ID,
		"state":      string(job.State),
		"progress":   job.Progress,
		"created_at": job.CreatedAt,
	}, true
}
