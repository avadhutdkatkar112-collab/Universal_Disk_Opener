package jobs

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type JobStatus string

const (
	StatusPending   JobStatus = "PENDING"
	StatusRunning   JobStatus = "RUNNING"
	StatusCompleted JobStatus = "COMPLETED"
	StatusFailed    JobStatus = "FAILED"
)

type HashJobProgress struct {
	JobID          string    `json:"job_id"`
	TargetPath     string    `json:"target_path"`
	Status         JobStatus `json:"status"`
	BytesProcessed int64     `json:"bytes_processed"`
	TotalBytes     int64     `json:"total_bytes"`
	Percentage     float64   `json:"percentage"`
	ThroughputMBps float64   `json:"throughput_mbps"`
	ETASeconds     float64   `json:"eta_seconds"`
	Error          string    `json:"error,omitempty"`
	Result         any       `json:"result,omitempty"`
}

type HashJobManager struct {
	ctx       context.Context
	mu        sync.RWMutex
	jobs      map[string]*HashJobProgress
	cancelFns map[string]context.CancelFunc
	maxConc   int
	sem       chan struct{}
}

func NewHashJobManager(ctx context.Context, maxConcurrent int) *HashJobManager {
	if maxConcurrent <= 0 {
		maxConcurrent = 3
	}
	return &HashJobManager{
		ctx:       ctx,
		jobs:      make(map[string]*HashJobProgress),
		cancelFns: make(map[string]context.CancelFunc),
		maxConc:   maxConcurrent,
		sem:       make(chan struct{}, maxConcurrent),
	}
}

func (jm *HashJobManager) StartHashJob(
	jobID string,
	targetPath string,
	totalBytes int64,
	execFunc func(ctx context.Context, progressChan chan<- int64) (any, error),
) {
	job := &HashJobProgress{
		JobID:          jobID,
		TargetPath:     targetPath,
		Status:         StatusRunning,
		TotalBytes:     totalBytes,
		Percentage:     0.0,
		ThroughputMBps: 0.0,
		ETASeconds:     0.0,
		BytesProcessed: 0,
	}

	jm.mu.Lock()
	jm.jobs[jobID] = job
	jm.mu.Unlock()

	jobCtx, cancel := context.WithCancel(jm.ctx)
	jm.mu.Lock()
	jm.cancelFns[jobID] = cancel
	jm.mu.Unlock()

	go func() {
		jm.sem <- struct{}{}        // acquire concurrency slot
		defer func() { <-jm.sem }() // release concurrency slot

		progressChan := make(chan int64, 100)
		startTime := time.Now()

		// Progress emitter goroutine
		done := make(chan struct{})
		go func() {
			for {
				select {
				case bytesRead, ok := <-progressChan:
					if !ok {
						close(done)
						return
					}
					jm.mu.Lock()
					elapsed := time.Since(startTime).Seconds()
					if elapsed < 0.001 {
						elapsed = 0.001
					}

					job.BytesProcessed = bytesRead
					if job.TotalBytes > 0 {
						job.Percentage = (float64(bytesRead) / float64(job.TotalBytes)) * 100.0
					}
					job.ThroughputMBps = (float64(bytesRead) / (1024 * 1024)) / elapsed

					remainingBytes := job.TotalBytes - bytesRead
					if job.ThroughputMBps > 0 && remainingBytes > 0 {
						job.ETASeconds = (float64(remainingBytes) / (1024 * 1024)) / job.ThroughputMBps
					} else {
						job.ETASeconds = 0
					}

					runtime.EventsEmit(jm.ctx, "hash:progress", job)
					jm.mu.Unlock()
				case <-jobCtx.Done():
					// Drain remaining progress
					for {
						select {
						case _, ok := <-progressChan:
							if !ok {
								close(done)
								return
							}
						default:
							close(done)
							return
						}
					}
				}
			}
		}()

		result, err := execFunc(jobCtx, progressChan)
		close(progressChan)
		<-done // wait for progress emitter to finish

		jm.mu.Lock()
		delete(jm.cancelFns, jobID)

		if err != nil {
			job.Status = StatusFailed
			job.Error = err.Error()
			runtime.EventsEmit(jm.ctx, "hash:complete", job)
		} else {
			job.Status = StatusCompleted
			job.Percentage = 100.0
			job.Result = result
			runtime.EventsEmit(jm.ctx, "hash:complete", job)
		}
		jm.mu.Unlock()
	}()
}

func (jm *HashJobManager) CancelJob(jobID string) bool {
	jm.mu.Lock()
	cancel, ok := jm.cancelFns[jobID]
	jm.mu.Unlock()

	if ok {
		cancel()
		jm.mu.Lock()
		job, exists := jm.jobs[jobID]
		if exists && job.Status == StatusRunning {
			job.Status = StatusFailed
			job.Error = "cancelled by user"
		}
		jm.mu.Unlock()
		return true
	}
	return false
}

func (jm *HashJobManager) GetJob(jobID string) (*HashJobProgress, bool) {
	jm.mu.RLock()
	defer jm.mu.RUnlock()
	job, ok := jm.jobs[jobID]
	return job, ok
}

func (jm *HashJobManager) ListJobs() []*HashJobProgress {
	jm.mu.RLock()
	defer jm.mu.RUnlock()
	result := make([]*HashJobProgress, 0, len(jm.jobs))
	for _, job := range jm.jobs {
		result = append(result, job)
	}
	return result
}

func (jm *HashJobManager) GenerateJobID(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}
