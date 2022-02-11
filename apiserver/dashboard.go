package apiserver

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// DashboardRouter for handling routes that are specific to the dashboard.
// The APIServer will be initialized with an instance of DashboardRouter.
// The dashboard routes depends on the mode in which the scheduler is run.
type DashboardRouter interface {
	// Name should return the key for the dashboard type
	Name() string
	// SetupRouter should set up the routes for the dashboard
	SetupRouter(*gin.RouterGroup)
}

// HandleDashboardName is the handler for `/dashboard/name` route of the APIServer
func (srv *APIServer) HandleDashboardName(c *gin.Context) {
	if srv.dashboard == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "no dashboard router set",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"name": srv.dashboard.Name(),
	})
}

// HandleDashboard is the handler for the router `/dashboard`
func (srv *APIServer) HandleDashboard(c *gin.Context) {

}
