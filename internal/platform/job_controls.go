package platform

import (
	"time"
)

// Pause suspends a running job.
func (jm *JobManager) Pause(jobID string) bool {
	jm.mu.RLock()
	job, exists := jm.jobs[jobID]
	jm.mu.RUnlock()

	if !exists || job.State != StateRunning {
		return false
	}

	job.mu.Lock()
	job.State = "PAUSED"
	job.mu.Unlock()

	jm.eventBus.Publish(Event{
		Type:      "JOB_PAUSED",
		Source:    "JobManager",
		Payload:   map[string]any{"job_id": jobID},
		Timestamp: time.Now(),
	})
	return true
}

// Resume restarts a paused job.
func (jm *JobManager) Resume(jobID string) bool {
	jm.mu.RLock()
	job, exists := jm.jobs[jobID]
	jm.mu.RUnlock()

	if !exists || job.State != "PAUSED" {
		return false
	}

	job.mu.Lock()
	job.State = StateRunning
	job.mu.Unlock()

	jm.eventBus.Publish(Event{
		Type:      "JOB_RESUMED",
		Source:    "JobManager",
		Payload:   map[string]any{"job_id": jobID},
		Timestamp: time.Now(),
	})
	return true
}

// List returns all tracked jobs.
func (jm *JobManager) List() []*Job {
	jm.mu.RLock()
	defer jm.mu.RUnlock()
	jobs := make([]*Job, 0, len(jm.jobs))
	for _, j := range jm.jobs {
		jobs = append(jobs, j)
	}
	return jobs
}
