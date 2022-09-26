package testlib

import (
	"github.com/netrixframework/netrix/sm"
	"github.com/netrixframework/netrix/types"
)

func IsMessageFromPart(partLabel string) sm.Condition {
	return func(e *types.Event, c *sm.Context) bool {
		partition, ok := types.VarSetGet[*Partition](c.Vars, partitionKey)
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

func IsMessageAcrossPartition() sm.Condition {
	return func(e *types.Event, c *sm.Context) bool {
		message, ok := c.GetMessage(e)
		if !ok {
			return false
		}
		partition, ok := types.VarSetGet[*Partition](c.Vars, partitionKey)
		if !ok {
			return false
		}
		fromPartition, ok := partition.GetPartLabel(message.From)
		if !ok {
			return false
		}
		toPartition, ok := partition.GetPartLabel(message.To)
		if !ok {
			return false
		}
		return fromPartition != toPartition
	}
}

func IsMessageWithinPartition() sm.Condition {
	return func(e *types.Event, c *sm.Context) bool {
		message, ok := c.GetMessage(e)
		if !ok {
			return false
		}
		partition, ok := types.VarSetGet[*Partition](c.Vars, partitionKey)
		if !ok {
			return false
		}
		fromPartition, ok := partition.GetPartLabel(message.From)
		if !ok {
			return false
		}
		toPartition, ok := partition.GetPartLabel(message.To)
		if !ok {
			return false
		}
		return fromPartition == toPartition
	}
}
