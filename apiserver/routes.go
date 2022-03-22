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

// HandleMessage is the handler for the route `/message`
// which is used by replicas to send messages
func (srv *APIServer) HandleMessage(c *gin.Context) {
	srv.Logger.Debug("Handling message")
	var msg types.Message
	if err := c.ShouldBindJSON(&msg); err != nil {
		srv.Logger.With(log.LogParams{"error": err}).Info("Bad message")
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to unmarshal request"})
		return
	}
	// TODO: dispatch message when it should not be intercepted

	msg.Parse(srv.messageParser)

	srv.ctx.MessageStore.Add(&msg)
	srv.ctx.MessageQueue.Add(&msg)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// HandleReplicaPost is the handler for the route `/replica` for a POST request.
// The route is used by replicas to register and start communicating with the scheduler
func (srv *APIServer) HandleReplicaPost(c *gin.Context) {
	var replica types.Replica
	if err := c.ShouldBindJSON(&replica); err != nil {
		srv.Logger.With(log.LogParams{"error": err}).Info("Bad replica request")
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to unmarshal request"})
		return
	}

	srv.Logger.With(log.LogParams{
		"replica_id": replica.ID,
		"ready":      replica.Ready,
		"info":       replica.Info,
	}).Info("Received replica information")

	srv.ctx.Replicas.Add(&replica)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

type eventS struct {
	types.Event `json:",inline"`
	Params      map[string]string `json:"params"`
}

// HandleEvent is the handler for the router `/event` .
// The route is used by replicas to send events to the scheduler
func (srv *APIServer) HandleEvent(c *gin.Context) {
	var e eventS
	if err := c.ShouldBindJSON(&e); err != nil {
		srv.Logger.With(log.LogParams{"error": err}).Info("Bad event request")
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to unmarshal request"})
		return
	}
	srv.Logger.With(log.LogParams{
		"replica": e.Replica,
		"type":    e.TypeS,
		"params":  e.Params,
	}).Debug("Received event")

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

	srv.ctx.EventQueue.Add(types.NewEvent(
		e.Replica,
		eventType,
		eventType.String(),
		types.EventID((srv.gen.Next())),
		e.Timestamp,
	))
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

// HandleLog is the handler for the route `/log`
// The route is used by replicas to send log messages
func (srv *APIServer) HandleLog(c *gin.Context) {
	var l types.ReplicaLog
	if err := c.ShouldBindJSON(&l); err != nil {
		srv.Logger.With(log.LogParams{"error": err}).Debug("Bad replica log request")
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to unmarshall request"})
		return
	}

	srv.Logger.With(log.LogParams{
		"replica": l.Replica,
		"message": l.Message,
	}).Debug("Received replica log")

	srv.ctx.LogStore.Add(&l)
	srv.ctx.LogQueue.Add(&l)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
