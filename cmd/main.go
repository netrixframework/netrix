package cmd

import (
	"github.com/netrixframework/netrix/cmd/strategies"
	"github.com/netrixframework/netrix/cmd/visualizer"
	"github.com/netrixframework/netrix/config"
	"github.com/spf13/cobra"
)

// RootCmd returns the root cobra command of the scheduler tool
func RootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "netrix",
		Short: "Tool to test/understand consensus algorithms",
	}
	cmd.CompletionOptions.DisableDefaultCmd = true
	cmd.PersistentFlags().StringVarP(&config.ConfigPath, "config", "c", "config.json", "Config file path")
	cmd.AddCommand(visualizer.VisualizerCmd())
	cmd.AddCommand(strategies.StrategiesCmd())
	return cmd
}
