package scheduler

import (
	"log"
	"sync"

	"Tracker/internal/config"
	"Tracker/internal/core"
	"Tracker/internal/runtimeflow"
	"Tracker/internal/storage"
	"github.com/robfig/cron/v3"
)

// Runner runs the default pipeline through the shared runtime executor.
type Runner struct {
	App          config.App
	PipelinePath string
	DB           *storage.DB
}

// Run executes the default pipeline once.
func (r *Runner) Run() (count int, err error) {
	if r.PipelinePath == "" {
		return 0, nil
	}
	eng := core.New()
	defer eng.Close()
	exec := runtimeflow.NewExecutor(eng, r.DB, r.App, r.PipelinePath)
	result, err := exec.RunDefault(runtimeflow.RunInput{
		Source:        "scheduler",
		RequestPath:   "scheduler/run",
		SubjectID:     r.App.Release.Instance,
		PersistOutput: true,
		UseDBDefault:  true,
	})
	if err != nil {
		return 0, err
	}
	return len(result.Items), nil
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
