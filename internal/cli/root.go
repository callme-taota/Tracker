package cli

import (
	"github.com/spf13/cobra"
)

// Root returns the root command.
func Root() *cobra.Command {
	root := &cobra.Command{
		Use:   "tracker",
		Short: "Tracker - AI information intelligence platform",
	}
	root.AddCommand(runCmd())
	root.AddCommand(serveCmd())
	root.AddCommand(pipelineCmd())
	root.AddCommand(pluginCmd())
	return root
}

func pipelineCmd() *cobra.Command {
	c := &cobra.Command{Use: "pipeline", Short: "Pipeline operations"}
	c.AddCommand(pipelineListCmd())
	return c
}

func pluginCmd() *cobra.Command {
	c := &cobra.Command{Use: "plugin", Short: "Plugin operations"}
	c.AddCommand(pluginListCmd())
	c.AddCommand(pluginInstallCmd())
	return c
}
