package cli

import (
	"fmt"

	"Tracker/internal/plugin"
	"github.com/spf13/cobra"
)

func pluginInstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "install [name]",
		Short: "Install a plugin (MVP: lists built-in plugins only)",
		RunE: func(cmd *cobra.Command, args []string) error {
			builtin := plugin.BuiltinPluginIDs()
			if len(args) == 0 {
				fmt.Println("Built-in plugins (already available):")
				for _, id := range builtin {
					fmt.Println("  -", id)
				}
				return nil
			}
			name := args[0]
			for _, id := range builtin {
				if id == name {
					fmt.Printf("Plugin %q is built-in and already available.\n", name)
					return nil
				}
			}
			fmt.Printf("Plugin %q not found in built-in list. Marketplace not implemented yet.\n", name)
			return nil
		},
	}
}
