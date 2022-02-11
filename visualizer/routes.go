package visualizer

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/netrixframework/netrix/log"
	"github.com/netrixframework/netrix/types"
)

func (v *Visualizer) SetupRouter(r *gin.RouterGroup) {
	group := r.Group("/visualize")
	group.GET("/messages", v.HandleMessagesGET)
	group.POST("/message/:id", v.HandleMessagesPOST)
}

type messagesGetParams struct {
	From      string `form:"from"`
	To        string `form:"to"`
	Delivered bool   `form:"delivered"`
	Pending   bool   `form:"pending,default=true"`
}

func (v *Visualizer) HandleMessagesGET(c *gin.Context) {
	var params messagesGetParams
	if err := c.ShouldBindQuery(&params); err != nil {
		v.Logger.With(log.LogParams{"error": err}).Debug("bad messages request")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Error parsing query"})
		return
	}
	if params.Delivered && params.Pending {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Use only one of pending,delivered"})
		return
	}
	if params.From != "" && params.To != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Use only one of from,to"})
		return
	}
	var messages map[string]*types.Message
	var ok bool
	if params.From != "" {
		if params.Delivered {
			messages, ok = v.messagePool.GetDeliveredFrom(types.ReplicaID(params.From))
		} else {
			messages, ok = v.messagePool.GetPendingFrom(types.ReplicaID(params.From))
		}
	} else {
		if params.Delivered {
			messages, ok = v.messagePool.GetDeliveredTo(types.ReplicaID(params.To))
		} else {
			messages, ok = v.messagePool.GetPendingTo(types.ReplicaID(params.To))
		}
	}
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid replica ID"})
		return
	}
	messagesL := make([]*types.Message, len(messages))
	i := 0
	for _, m := range messages {
		messagesL[i] = m
		i = i + 1
	}
	c.JSON(http.StatusOK, gin.H{"messages": messages})
}

func (v *Visualizer) HandleMessagesPOST(c *gin.Context) {
	messageID, ok := c.Params.Get("id")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing parameters"})
		return
	}
	message, ok := v.messagePool.Get(messageID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "message does not exist"})
		return
	}
	err := v.dispatcher.DispatchMessage(message)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to dispatch message"})
	}

	v.messagePool.MarkDelivered(messageID)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
