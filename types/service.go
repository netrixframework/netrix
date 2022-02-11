package types

import (
	"sync"

	"github.com/netrixframework/netrix/log"
)

// Service is any entity which runs on a separate thread
type Service interface {
	// Name of the service
	Name() string
	// Start to start the service
	Start() error
	// Running to indicate if the service is running
	Running() bool
	// Stop to stop the service
	Stop() error
	// Quit returns a channel which will be closed once the service stops running
	QuitCh() <-chan struct{}
}

// RestartableService is a service which can be restarted
type RestartableService interface {
	Service
	// Restart restarts the service
	Restart() error
}

// BaseService provides the basic nuts an bolts needed to implement a service
type BaseService struct {
	running bool
	o       *sync.Once
	lock    *sync.Mutex
	name    string
	quit    chan struct{}
	Logger  *log.Logger
}

// NewBaseService instantiates BaseService
func NewBaseService(name string, parentLogger *log.Logger) *BaseService {
	return &BaseService{
		running: false,
		lock:    new(sync.Mutex),
		name:    name,
		o:       new(sync.Once),
		quit:    make(chan struct{}),
		Logger:  parentLogger.With(log.LogParams{"service": name}),
	}
}

// StartRunning is called to set the running flag
func (b *BaseService) StartRunning() {
	b.Logger.Debug("Starting service")
	b.lock.Lock()
	defer b.lock.Unlock()
	b.running = true
}

// StopRunning is called to unset the running flag
func (b *BaseService) StopRunning() {
	b.Logger.Debug("Stopping service")
	b.lock.Lock()
	defer b.lock.Unlock()
	b.running = false
	b.o.Do(func() {
		b.Logger.Debug("Closing quit channel")
		close(b.quit)
	})
}

// Name returns the name of the service
func (b *BaseService) Name() string {
	return b.name
}

// Running returns the flag
func (b *BaseService) Running() bool {
	b.lock.Lock()
	defer b.lock.Unlock()
	return b.running
}

// QuitCh returns the quit channel which will be closed when the service stops running
func (b *BaseService) QuitCh() <-chan struct{} {
	return b.quit
}
