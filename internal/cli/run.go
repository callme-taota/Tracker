package cli

import (
	"fmt"

	"Tracker/internal/config"
	"Tracker/internal/core"
	"Tracker/internal/runtimeflow"
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
			eng := core.New()
			defer eng.Close()
			exec := runtimeflow.NewExecutor(eng, nil, app, pipelinePath)
			result, err := exec.RunPipelineFile(runtimeflow.RunInput{
				Source:      "cli",
				RequestPath: "cli/run",
				SubjectID:   app.Release.Instance,
			})
			if err != nil {
				return fmt.Errorf("run pipeline: %w", err)
			}
			fmt.Printf("Pipeline finished. Items processed: %d\n", len(result.Items))
			return nil
		},
	}
	cmd.Flags().StringVarP(&configPath, "config", "c", "", "Config file (tracker.yaml) or pipeline YAML path")
	return cmd
}
