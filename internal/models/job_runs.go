package models

import "time"

// JobRun records a single invocation of a background job (e.g. scheduled snapshot).
type JobRun struct {
	ID         int64      `json:"id"`
	JobName    string     `json:"job_name"`
	Status     string     `json:"status"` // "running", "completed", "failed"
	Outcome    string     `json:"outcome"`
	Log        string     `json:"log"`
	StartedAt  time.Time  `json:"started_at"`
	FinishedAt *time.Time `json:"finished_at"`
}
