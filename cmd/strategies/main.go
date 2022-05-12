package strategies

import (
	"github.com/spf13/cobra"
)

func StrategiesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "strategy [strategy_name]",
		Short: "Run Netrix with the specified strategy",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// termCh := util.Term()

			// conf, err := config.ParseConfig(config.ConfigPath)
			// if err != nil {
			// 	return fmt.Errorf("failed to parse config: %s", err)
			// }
			// log.Init(conf.LogConfig)
			// ctx := context.NewRootContext(conf, log.DefaultLogger)

			// strategy, err := strategies.GetStrategyDriver(ctx, args[0])
			// if err != nil {
			// 	return fmt.Errorf("failed to initialize strategy: %s", err)
			// }

			// ctx.Start()
			// strategy.Start()

			// <-termCh
			// strategy.Stop()
			// ctx.Stop()
			return nil
		},
	}
}
