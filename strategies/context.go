package strategies

import "github.com/netrixframework/netrix/types"

type Context struct {
	MessagePool *types.MessageStore
	// Replicas reference to the replica store
	Replicas *types.ReplicaStore
	// EventDAG is the directed acyclic graph all prior events
	EventDAG *types.EventDAG
	// Vars is a generic key value store to facilate maintaining auxilliary information
	// during the execution of a testcase
	Vars *types.VarSet

	PendingMessages *types.MessageStore
}
