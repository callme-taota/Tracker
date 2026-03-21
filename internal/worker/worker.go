package worker

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"time"

	"Tracker/internal/core"
	"Tracker/internal/pipeline"
	"Tracker/internal/plugin"
	"Tracker/internal/storage"
)

// Start runs a background loop that claims and executes pending jobs until ctx is done.
func Start(ctx context.Context, db *storage.DB, eng *core.Engine, interval time.Duration) {
	if db == nil || eng == nil {
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
			runOne(ctx, db, eng)
		}
	}
}

func runOne(ctx context.Context, db *storage.DB, eng *core.Engine) {
	instanceID := os.Getenv("TRACKER_INSTANCE_ID")
	j, err := db.ClaimNextPendingJob(instanceID)
	if err != nil || j == nil {
		return
	}
	row, err := db.GetPipelineDefinition(j.PipelineID)
	if err != nil || row == nil {
		_ = db.FailJob(j.ID, j.Attempt, j.MaxAttempt, "pipeline not found")
		return
	}
	g, err := pipeline.ParseGraphJSON([]byte(row.GraphJSON))
	if err != nil {
		_ = db.FailJob(j.ID, j.Attempt, j.MaxAttempt, err.Error())
		return
	}
	global := plugin.Config{
		"api_key":   os.Getenv("OPENAI_API_KEY"),
		"bot_token": os.Getenv("TELEGRAM_BOT_TOKEN"),
		"chat_id":   os.Getenv("TELEGRAM_CHAT_ID"),
	}
	if err := eng.Init(global); err != nil {
		_ = db.FailJob(j.ID, j.Attempt, j.MaxAttempt, err.Error())
		return
	}
	runCtx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()
	var payload map[string]interface{}
	if j.Payload != "" {
		_ = json.Unmarshal([]byte(j.Payload), &payload)
	}
	runCtx = core.WithRunPayload(runCtx, payload)
	items, err := eng.RunGraphWithContext(runCtx, j.PipelineID, g)
	if err != nil {
		log.Printf("worker: job %d failed: %v", j.ID, err)
		_ = db.FailJob(j.ID, j.Attempt, j.MaxAttempt, err.Error())
		return
	}
	if len(items) > 0 {
		for _, it := range items {
			var sid *int64
			ts := it.Timestamp
			if ts.IsZero() {
				ts = time.Now()
			}
			itemID, _ := db.SaveItem(sid, it.Title, it.URL, it.Content, it.Summary, ts, "")
			if itemID > 0 && (it.Summary != "" || len(it.KeyPoints) > 0) {
				kpJSON := storage.KeyPointsToJSON(it.KeyPoints)
				_, _ = db.SaveSummary(itemID, it.Summary, kpJSON)
			}
		}
	}
	if err := db.CompleteJob(j.ID); err != nil {
		log.Printf("worker: complete job %d: %v", j.ID, err)
	}
}
