package cli

import (
	"fmt"
	"os"

	"Tracker/internal/config"
	"Tracker/internal/core"
	"Tracker/internal/plugin"
	"Tracker/internal/pipeline"
	"github.com/spf13/cobra"
)

func runCmd() *cobra.Command {
	var configPath string
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run the default pipeline",
		RunE: func(cmd *cobra.Command, args []string) error {
			app, pipelinePath, err := config.Load(configPath)
			if err != nil {
				return err
			}
			pipe, err := pipeline.LoadFromFile(pipelinePath)
			if err != nil {
				return fmt.Errorf("load pipeline: %w", err)
			}
			eng := core.New()
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
			items, err := eng.Run(pipe)
			if err != nil {
				return fmt.Errorf("run pipeline: %w", err)
			}
			fmt.Printf("Pipeline finished. Items processed: %d\n", len(items))
			return nil
		},
	}
	cmd.Flags().StringVarP(&configPath, "config", "c", "", "Config file (tracker.yaml) or pipeline YAML path")
	return cmd
}
