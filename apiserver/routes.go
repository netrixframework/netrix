package apiserver

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/netrixframework/netrix/log"
	"github.com/netrixframework/netrix/types"
)

var (
	messageSendEventType    = "MessageSend"
	messageReceiveEventType = "MessageReceive"
	timeoutStartEventType   = "TimeoutStart"
	timeoutEndEventType     = "TimeoutEnd"
)

// handleMessage is the handler for the route `/message`
// which is used by replicas to send messages
func (srv *APIServer) handleMessage(c *gin.Context) {
	srv.Logger.Debug("Handling message")
	var msg types.Message
	if err := c.ShouldBindJSON(&msg); err != nil {
		srv.Logger.With(log.LogParams{"error": err}).Debug("Bad message")
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to unmarshal request"})
		return
	}

	srv.lock.Lock()
	_, block := srv.resetReplicas[msg.From]
	srv.lock.Unlock()
	if block {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
		return
	}

	err := msg.Parse(srv.messageParser)
	srv.Logger.With(log.LogParams{
		"message": msg.Data,
		"type":    msg.Type,
		"from":    msg.From,
		"to":      msg.To,
		"id":      msg.ID,
	}).Debug("Received message")
	if err == nil {
		srv.Logger.With(log.LogParams{
			"parsed_message": msg.ParsedMessage.String(),
		}).Debug("Parsed message")
	}

	srv.ctx.MessageStore.Add(msg.ID, &msg)
	srv.ctx.MessageQueue.Add(&msg)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// handleReplicaPost is the handler for the route `/replica` for a POST request.
// The route is used by replicas to register and start communicating with the scheduler
func (srv *APIServer) handleReplicaPost(c *gin.Context) {
	var replica types.Replica
	if err := c.ShouldBindJSON(&replica); err != nil {
		srv.Logger.With(log.LogParams{"error": err}).Debug("Bad replica request")
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to unmarshal request"})
		return
	}
	if replica.Ready {
		srv.lock.Lock()
		delete(srv.resetReplicas, replica.ID)
		srv.lock.Unlock()
	}

	srv.Logger.With(log.LogParams{
		"replica_id": replica.ID,
		"ready":      replica.Ready,
		"info":       replica.Info,
	}).Debug("Received replica information")

	srv.ctx.Replicas.Add(&replica)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// handleEvent is the handler for the router `/event` .
// The route is used by replicas to send events to the scheduler
func (srv *APIServer) handleEvent(c *gin.Context) {
	var e = &types.Event{}
	if err := c.ShouldBindJSON(e); err != nil {
		srv.Logger.With(log.LogParams{"error": err}).Debug("Bad event request")
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to unmarshal request"})
		return
	}
	srv.lock.Lock()
	_, block := srv.resetReplicas[e.Replica]
	srv.lock.Unlock()
	if block {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
		return
	}

	var eventType types.EventType
	switch e.TypeS {
	case messageSendEventType:
		messageID := e.Params["message_id"]
		eventType = types.NewMessageSendEventType(messageID)
	case messageReceiveEventType:
		messageID := e.Params["message_id"]
		eventType = types.NewMessageReceiveEventType(messageID)
	case timeoutStartEventType:
		timeout, ok := types.TimeoutFromParams(e.Replica, e.Params)
		if ok {
			eventType = types.NewTimeoutStartEventType(timeout)
		} else {
			eventType = types.NewGenericEventType(e.Params, e.TypeS)
		}
	case timeoutEndEventType:
		timeout, ok := types.TimeoutFromParams(e.Replica, e.Params)
		if ok {
			eventType = types.NewTimeoutEndEventType(timeout)
		} else {
			eventType = types.NewGenericEventType(e.Params, e.TypeS)
		}
	default:
		eventType = types.NewGenericEventType(e.Params, e.TypeS)
	}
	event := types.NewEventWithParams(
		e.Replica,
		eventType,
		eventType.String(),
		types.EventID((srv.gen.Next())),
		e.Timestamp,
		e.Params,
	)

	srv.Logger.With(log.LogParams{
		"replica": e.Replica,
		"type":    e.TypeS,
		"params":  e.Params,
		"id":      event.ID,
	}).Debug("Received event")
	srv.ctx.EventQueue.Add(event)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (srv *APIServer) handleReplicas(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"replicas": srv.ctx.Replicas.Iter(),
	})
}

func (srv *APIServer) handleReplicaGet(c *gin.Context) {
	replicaID, ok := c.Params.Get("replica")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing replica param"})
		return
	}
	replica, ok := srv.ctx.Replicas.Get(types.ReplicaID(replicaID))
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "replica id does not exist"})
		return
	}
	c.JSON(http.StatusOK, replica)
}
