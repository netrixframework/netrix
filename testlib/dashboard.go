package testlib

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/netrixframework/netrix/log"
)

// SetupRouter for setting up the dashboard routes implements DashboardRouter
func (srv *TestingServer) SetupRouter(router *gin.RouterGroup) {
	router.GET("/testcases", srv.handleTestCases)
	router.GET("/testcase/:name", srv.handleTestCase)
	router.POST("/logs", srv.handlerLogs)
}

// Name implements DashboardRouter
func (srv *TestingServer) Name() string {
	return "Testlib"
}

func (srv *TestingServer) handleTestCases(c *gin.Context) {
	responses := make([]map[string]string, len(srv.testCases))
	i := 0
	for _, t := range srv.testCases {
		responses[i] = map[string]string{
			"name":    t.Name,
			"timeout": t.Timeout.String(),
		}
		i++
	}
	c.JSON(http.StatusOK, gin.H{"testcases": responses})
}

func (srv *TestingServer) handleTestCase(c *gin.Context) {
	name, ok := c.Params.Get("name")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing param `name`"})
		return
	}
	response := gin.H{}
	testcase, ok := srv.testCases[name]
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "no such testcase"})
		return
	}
	response["testcase"] = map[string]string{
		"name":    testcase.Name,
		"timeout": testcase.Timeout.String(),
	}
	c.JSON(http.StatusOK, response)
}

type logRequest struct {
	Count   int               `form:"count,default=-1"`
	From    int               `form:"from,default=-1"`
	KeyVals map[string]string `form:"keyvals"`
}

func (srv *TestingServer) handlerLogs(c *gin.Context) {
	var r logRequest
	if err := c.ShouldBindJSON(&r); err != nil {
		srv.Logger.With(log.LogParams{"error": err}).Info("Bad replica request")
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to unmarshal request"})
		return
	}
	logs := srv.ctx.ReportStore.GetLogs(r.KeyVals, r.Count, r.From)
	c.JSON(http.StatusOK, gin.H{
		"logs": logs,
	})
}
