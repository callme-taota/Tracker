package scheduler

import (
	"log"
	"os"
	"sync"
	"time"

	"Tracker/internal/core"
	"Tracker/internal/pipeline"
	"Tracker/internal/plugin"
	"Tracker/internal/storage"
	"github.com/robfig/cron/v3"
)

// Runner runs a pipeline once (load config, init engine, run, optionally save to DB).
type Runner struct {
	PipelinePath string
	DB           *storage.DB
	Env          map[string]string
}

// Run executes the pipeline once. Env overrides are applied to os env for this run only (not persisted).
func (r *Runner) Run() (count int, err error) {
	pipe, err := pipeline.LoadFromFile(r.PipelinePath)
	if err != nil {
		return 0, err
	}
	eng := core.New()
	global := plugin.Config{
		"api_key":   os.Getenv("OPENAI_API_KEY"),
		"bot_token": os.Getenv("TELEGRAM_BOT_TOKEN"),
		"chat_id":   os.Getenv("TELEGRAM_CHAT_ID"),
	}
	for k, v := range r.Env {
		global[k] = v
	}
	if err := eng.Init(global); err != nil {
		return 0, err
	}
	items, err := eng.Run(pipe)
	if err != nil {
		return 0, err
	}
	if r.DB != nil && len(items) > 0 {
		for _, it := range items {
			var sid *int64
			ts := it.Timestamp
			if ts.IsZero() {
				ts = time.Now()
			}
			itemID, _ := r.DB.SaveItem(sid, it.Title, it.URL, it.Content, it.Summary, ts, "")
			if itemID > 0 && (it.Summary != "" || len(it.KeyPoints) > 0) {
				kpJSON := storage.KeyPointsToJSON(it.KeyPoints)
				_, _ = r.DB.SaveSummary(itemID, it.Summary, kpJSON)
			}
		}
	}
	return len(items), nil
}

// Scheduler runs the pipeline on a cron schedule.
type Scheduler struct {
	runner *Runner
	cron   *cron.Cron
	mu     sync.Mutex
}

// New creates a scheduler. Schedule is a cron expression (e.g. "0 */2 * * *" for every 2 hours). Empty = no schedule.
func New(runner *Runner, schedule string) *Scheduler {
	s := &Scheduler{runner: runner}
	if schedule == "" {
		return s
	}
	s.cron = cron.New()
	_, err := s.cron.AddFunc(schedule, func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		n, err := s.runner.Run()
		if err != nil {
			log.Printf("[scheduler] run failed: %v", err)
			return
		}
		log.Printf("[scheduler] pipeline run ok, items=%d", n)
	})
	if err != nil {
		log.Printf("[scheduler] invalid cron %q: %v", schedule, err)
		return &Scheduler{runner: runner}
	}
	return s
}

// Start starts the cron scheduler (non-blocking).
func (s *Scheduler) Start() {
	if s.cron != nil {
		s.cron.Start()
		log.Printf("[scheduler] started")
	}
}

// Stop stops the scheduler.
func (s *Scheduler) Stop() {
	if s.cron != nil {
		s.cron.Stop()
	}
}
