package worker

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"Tracker/internal/runtimeflow"
	"Tracker/internal/storage"
)

// Start runs a background loop that claims and executes pending jobs until ctx is done.
func Start(ctx context.Context, db *storage.DB, exec *runtimeflow.Executor, interval time.Duration) {
	if db == nil || exec == nil {
		return
	}
	if interval <= 0 {
		interval = 3 * time.Second
	}
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			runOne(ctx, db, exec)
		}
	}
}

func runOne(ctx context.Context, db *storage.DB, exec *runtimeflow.Executor) {
	instanceID := exec.App.Release.Instance
	j, err := db.ClaimNextPendingJob(instanceID)
	if err != nil || j == nil {
		return
	}
	row, err := db.GetPipelineDefinition(j.PipelineID)
	if err != nil || row == nil {
		_ = db.FailJob(j.ID, j.Attempt, j.MaxAttempt, "pipeline not found")
		return
	}
	runCtx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()
	var payload map[string]interface{}
	if j.Payload != "" {
		_ = json.Unmarshal([]byte(j.Payload), &payload)
	}
	runtimePayload := payload
	if runtimePayload == nil {
		runtimePayload = map[string]interface{}{}
	}
	if _, ok := runtimePayload["__tracker_runtime"]; !ok {
		runtimePayload["__tracker_runtime"] = map[string]interface{}{}
	}
	subjectID := instanceID
	flagOverrides := map[string]string{}
	if meta, ok := runtimePayload["__tracker_runtime"].(map[string]interface{}); ok {
		if s, ok := meta["subject_id"].(string); ok && s != "" {
			subjectID = s
		}
		if raw, ok := meta["overrides"].(map[string]interface{}); ok {
			for k, v := range raw {
				if s, ok := v.(string); ok && s != "" {
					flagOverrides[k] = s
				}
			}
		}
	}
	result, err := exec.RunPipelineByID(runtimeflow.RunInput{
		Ctx:           runCtx,
		Source:        "worker",
		RequestPath:   "worker/job",
		SubjectID:     subjectID,
		PipelineID:    j.PipelineID,
		JobID:         j.ID,
		Payload:       runtimePayload,
		FlagOverrides: flagOverrides,
		PersistOutput: true,
	})
	if err != nil {
		log.Printf("worker: job %d failed: %v", j.ID, err)
		_ = db.FailJob(j.ID, j.Attempt, j.MaxAttempt, err.Error())
		return
	}
	if err := db.CompleteJob(j.ID); err != nil {
		log.Printf("worker: complete job %d: %v", j.ID, err)
	}
	_ = row
	_ = result
}
