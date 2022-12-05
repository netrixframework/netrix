// Package apiserver implements the API interface of Netrix.
//
// The package is used internally by Netrix to instantiate an API interface.
// The package also contains the interface to invoke directives and send messages to processes/replicas.
package apiserver

import (
	goctx "context"
	"errors"
	"net/http"
	"path"
	"runtime"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/netrixframework/netrix/context"
	"github.com/netrixframework/netrix/log"
	"github.com/netrixframework/netrix/types"
	"github.com/netrixframework/netrix/util"
)

// DefaultAddr is the default address of the APIServer
const DefaultAddr = "0.0.0.0:7074"

// APIServer runs a HTTP server to receive messages from
// the replicas. APIServer is used to send messages and
// directives to the replicas.

// APIServer implements [types.Service] interface.
type APIServer struct {
	router        *gin.Engine
	ctx           *context.RootContext
	gen           *util.Counter
	dashboard     DashboardRouter
	messageParser types.MessageParser

	server             *http.Server
	addr               string
	clients            map[types.ReplicaID]*http.Client
	dispatchedMessages map[types.MessageID]bool
	resetReplicas      map[types.ReplicaID]bool
	lock               *sync.Mutex

	*types.BaseService
}

// NewAPIServer instantiates APIServer
func NewAPIServer(ctx *context.RootContext, messageParser types.MessageParser, dashboard DashboardRouter) *APIServer {

	server := &APIServer{
		gen:           ctx.EventIDGen,
		ctx:           ctx,
		addr:          ctx.Config.APIServerAddr,
		dashboard:     dashboard,
		messageParser: messageParser,

		clients:            make(map[types.ReplicaID]*http.Client),
		dispatchedMessages: make(map[types.MessageID]bool),
		resetReplicas:      make(map[types.ReplicaID]bool),
		lock:               new(sync.Mutex),
		BaseService:        types.NewBaseService("APIServer", ctx.Logger),
	}
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(server.logMiddleware)

	router.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/dashboard/app")
	})
	router.POST("/message", server.handleMessage)
	router.POST("/event", server.handleEvent)
	router.POST("/replica", server.handleReplicaPost)

	router.GET("/replicas", server.handleReplicas)
	router.GET("/replicas/:replica", server.handleReplicaGet)
	router.GET("/dashboard/name", server.handleDashboardName)

	_, file, _, _ := runtime.Caller(0)
	router.StaticFS("/dashboard/app", gin.Dir(path.Join(path.Dir(file), "dist"), false))

	dashboard.SetupRouter(router.Group("/dashboard/api"))

	server.router = router
	server.server = &http.Server{
		Addr:    server.addr,
		Handler: router,
	}

	return server
}

func (a *APIServer) logMiddleware(c *gin.Context) {
	start := time.Now()
	path := c.Request.URL.Path
	raw := c.Request.URL.RawQuery

	// Process request
	c.Next()

	end := time.Now()
	if raw != "" {
		path = path + "?" + raw
	}
	a.Logger.With(log.LogParams{
		"timestamp":   end,
		"latency":     end.Sub(start).String(),
		"client_ip":   c.ClientIP(),
		"method":      c.Request.Method,
		"status_code": c.Writer.Status(),
		"error":       c.Errors.ByType(gin.ErrorTypePrivate).String(),
		"body_size":   c.Writer.Size(),
		"path":        path,
	}).Debug("Handled request")
}

// Start runs the HTTP server and starts listening to the API requests.
func (a *APIServer) Start() {
	a.StartRunning()
	go func() {
		a.Logger.With(log.LogParams{
			"addr": a.addr,
		}).Info("API server starting!")
		if err := a.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			a.Logger.With(log.LogParams{
				"addr": a.addr,
				"err":  err,
			}).Fatal("API server closed!")
		}
	}()
}

// Stop terminates the HTTP server.
func (a *APIServer) Stop() {
	a.StopRunning()
	ctx, cancel := goctx.WithTimeout(goctx.Background(), 5*time.Second)
	defer cancel()
	if err := a.server.Shutdown(ctx); err != nil {
		a.Logger.Error("API server focefully shutdown")
	}
	a.Logger.Info("API server stopped!")
}
