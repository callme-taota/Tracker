package cli

import (
	"fmt"

	"Tracker/internal/core"
	"github.com/spf13/cobra"
)

func pluginListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List registered plugins",
		RunE: func(cmd *cobra.Command, args []string) error {
			eng := core.New()
			for _, p := range eng.PM.ListAll() {
				fmt.Printf("%s\t%s\t%s\n", p.Name(), p.Version(), p.Type())
			}
			return nil
		},
	}
}
