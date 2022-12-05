package testlib

import (
	"github.com/netrixframework/netrix/sm"
	"github.com/netrixframework/netrix/types"
)

// IsMessageFromPart condition returns true when message is from a replica that belongs to the specified part.
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

// IsMessageAcrossPartition condition returns true when the event represents a message between replicas of different partitions.
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

// IsMessageWithinPartition condition returns true when the event represents a message between two replicas of the same partition.
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
