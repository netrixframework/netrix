package visualizer

import (
	"fmt"

	"github.com/netrixframework/netrix/config"
	"github.com/netrixframework/netrix/context"
	"github.com/netrixframework/netrix/log"
	"github.com/netrixframework/netrix/util"
	"github.com/netrixframework/netrix/visualizer"
	"github.com/spf13/cobra"
)

// VisualizerCmd returns the command for running a visualizer
func VisualizerCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "viz",
		Short: "Visualize the execution of a consensus algorithm",
		RunE: func(cmd *cobra.Command, args []string) error {
			termCh := util.Term()

			conf, err := config.ParseConfig(config.ConfigPath)
			if err != nil {
				return fmt.Errorf("failed to parse config: %s", err)
			}
			log.Init(conf.LogConfig)
			ctx := context.NewRootContext(conf, log.DefaultLogger)

			viz := visualizer.NewVisualizer(ctx)
			ctx.Start()
			viz.Start()

			<-termCh
			viz.Stop()
			ctx.Stop()
			return nil
		},
	}
}
