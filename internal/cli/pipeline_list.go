package cli

import (
	"fmt"
	"os"

	"Tracker/internal/pipeline"
	"github.com/spf13/cobra"
)

func pipelineListCmd() *cobra.Command {
	var configPath string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List pipeline stages (from config)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if configPath == "" {
				configPath = "config.yaml"
				if _, err := os.Stat(configPath); os.IsNotExist(err) {
					configPath = "configs/default.yaml"
				}
			}
			pipe, err := pipeline.LoadFromFile(configPath)
			if err != nil {
				return fmt.Errorf("load pipeline: %w", err)
			}
			fmt.Println("Pipeline:", pipe.Name)
			for i, st := range pipe.Stages {
				fmt.Printf("  %d. %s (%s) plugin=%s\n", i+1, st.Name, st.PluginType, st.PluginID)
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&configPath, "config", "c", "", "Pipeline config YAML path")
	return cmd
}
