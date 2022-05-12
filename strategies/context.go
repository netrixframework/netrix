package strategies

import (
	"github.com/netrixframework/netrix/config"
	"github.com/netrixframework/netrix/types"
)

type Context struct {
	Config   *config.StrategyConfig
	Replicas *types.ReplicaStore
	Messages *types.MessageStore
}
