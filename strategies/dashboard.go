package strategies

import "github.com/gin-gonic/gin"

// SetupRouter for setting up the dashboard routes implements DashboardRouter
func (srv *Driver) SetupRouter(router *gin.RouterGroup) {
}

// Name implements DashboardRouter
func (srv *Driver) Name() string {
	return "StrategyDriver"
}
