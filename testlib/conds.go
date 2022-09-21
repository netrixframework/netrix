package testlib

import (
	"github.com/netrixframework/netrix/sm"
	"github.com/netrixframework/netrix/types"
)

func IsMessageFromPart(partLabel string) sm.Condition {
	return func(e *types.Event, c *sm.Context) bool {
		partition, ok := types.VarSetGet[*ReplicaPartition](c.Vars, partitionKey)
		if !ok {
			return false
		}
		message, ok := c.GetMessage(e)
		if !ok {
			return false
		}
		return partition.InPart(message.From, partLabel)
	}
}
