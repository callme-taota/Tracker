package cli

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"Tracker/internal/api"
	"Tracker/internal/config"
	"Tracker/internal/core"
	"Tracker/internal/pluginhub"
	"Tracker/internal/scheduler"
	"Tracker/internal/plugin"
	"Tracker/internal/storage"
	"Tracker/internal/worker"

	"github.com/spf13/cobra"
)

func serveCmd() *cobra.Command {
	var configPath string
	var flagPort int
	var flagDB string
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the web interface and API server (with optional pipeline schedule)",
		RunE: func(cmd *cobra.Command, args []string) error {
			app, pipelinePath, err := config.Load(configPath)
			if err != nil {
				return err
			}
			if flagPort > 0 {
				app.ServePort = flagPort
			}
			if flagDB != "" {
				app.DBPath = flagDB
			}
			eng := core.New()
			defer eng.Close()
			global := plugin.Config{
				"api_key":   os.Getenv("OPENAI_API_KEY"),
				"bot_token": os.Getenv("TELEGRAM_BOT_TOKEN"),
				"chat_id":   os.Getenv("TELEGRAM_CHAT_ID"),
			}
			for k, v := range app.Env {
				global[k] = v
			}
			if err := eng.Init(global); err != nil {
				return fmt.Errorf("init engine: %w", err)
			}
			eng.Hub.Subscribe(func(rt *pluginhub.RuntimeContext, ev pluginhub.Event) {
				if ev.Type == pluginhub.ErrorEvent && ev.Err != nil {
					log.Printf("[pipeline] err node=%s plugin=%s: %v", ev.NodeID, ev.PluginID, ev.Err)
				}
			})
			var db *storage.DB
			if app.DBPath != "" {
				db, err = storage.Open(app.DBPath)
				if err != nil {
					return fmt.Errorf("open db: %w", err)
				}
				defer db.Close()
			}
			var staticFS fs.FS
			if d, err := os.Stat("web/dist"); err == nil && d.IsDir() {
				staticFS = os.DirFS("web/dist")
			}
			srv := api.NewServer(eng, db, pipelinePath, staticFS)
			if db != nil {
				wctx, wcancel := context.WithCancel(context.Background())
				defer wcancel()
				go worker.Start(wctx, db, eng, 3*time.Second)
			}
			addr := fmt.Sprintf(":%d", app.ServePort)
			log.Printf("Tracker web UI: http://localhost%s", addr)
			if app.Schedule != "" {
				runner := &scheduler.Runner{PipelinePath: pipelinePath, DB: db, Env: app.Env}
				sched := scheduler.New(runner, app.Schedule)
				sched.Start()
				defer sched.Stop()
				log.Printf("Pipeline schedule: %s", app.Schedule)
			}
			go func() {
				if err := http.ListenAndServe(addr, srv.Handler()); err != nil && err != http.ErrServerClosed {
					log.Fatal(err)
				}
			}()
			quit := make(chan os.Signal, 1)
			signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
			<-quit
			return nil
		},
	}
	cmd.Flags().IntVarP(&flagPort, "port", "p", 0, "HTTP port (overrides config)")
	cmd.Flags().StringVar(&flagDB, "db", "", "SQLite path (overrides config; empty = use config)")
	cmd.Flags().StringVarP(&configPath, "config", "c", "", "Config file (tracker.yaml / config.yaml)")
	return cmd
}
