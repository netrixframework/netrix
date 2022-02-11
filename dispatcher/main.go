package dispatcher

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"sync"

	"github.com/netrixframework/netrix/context"
	"github.com/netrixframework/netrix/log"
	"github.com/netrixframework/netrix/types"
)

var (
	// ErrDestUnknown is returned when a message is asked to be dispatched to an unknown replicas
	ErrDestUnknown = errors.New("destination unknown")
	// ErrFailedMarshal is returned when the message could not be marshalled
	ErrFailedMarshal = errors.New("failed to marshal data")
	// ErrDuplicateDispatch is returned when a message already sent is being dispatched again
	ErrDuplicateDispatch = errors.New("message already dispatched")
	// ErrSendFaied is returned when the request could not be created
	ErrSendFailed = errors.New("sending failed")
	// ErrResponseReadFail is returned when the response to the request could not be read
	ErrResponseReadFail = errors.New("failed to read response")
	// ErrBadResponse is returned when the request did not receive a 2** response
	ErrBadResponse = errors.New("bad response")

	// Directive action to start the replica
	startAction = &directiveMessage{
		Action: "START",
	}

	// Directive action to stop the replica
	stopAction = &directiveMessage{
		Action: "STOP",
	}

	// Directive action to restart the replica
	restartAction = &directiveMessage{
		Action: "RESTART",
	}
)

type directiveMessage struct {
	Action string `json:"action"`
}

// Dispatcher should be used to send messages to the replicas and handles the marshalling of messages
type Dispatcher struct {
	// Replicas an instance of the replica store
	Replicas *types.ReplicaStore
	logger   *log.Logger

	clients            map[types.ReplicaID]*http.Client
	dispatchedMessages map[string]bool
	lock               *sync.Mutex
}

// NewDispatcher instantiates a new instance of Dispatcher
func NewDispatcher(ctx *context.RootContext) *Dispatcher {
	return &Dispatcher{
		Replicas:           ctx.Replicas,
		logger:             ctx.Logger.With(log.LogParams{"service": "dispatcher"}),
		clients:            make(map[types.ReplicaID]*http.Client),
		dispatchedMessages: make(map[string]bool),
		lock:               new(sync.Mutex),
	}
}

// DispatchMessage should be called to send an _intercepted_ message to a replica
// DispatchMessage sends a message with a particular ID only once,
// will return ErrDuplicateDispatch on successive calls
func (d *Dispatcher) DispatchMessage(msg *types.Message) error {
	d.lock.Lock()
	_, ok := d.dispatchedMessages[msg.ID]
	if ok {
		d.lock.Unlock()
		return ErrDuplicateDispatch
	}
	d.dispatchedMessages[msg.ID] = true
	d.lock.Unlock()

	d.logger.With(log.LogParams{
		"message_id": msg.ID,
	}).Debug("Dispatching message")
	replica, ok := d.Replicas.Get(msg.To)
	if !ok {
		return ErrDestUnknown
	}
	bytes, err := json.Marshal(msg)
	if err != nil {
		return ErrFailedMarshal
	}
	_, err = d.sendReq(replica, "/message", bytes)
	if err != nil {
		return err
	}
	return nil
}

// Send a timeout message to the replica
// This is the equivalent of ending a timeout at the replica
func (d *Dispatcher) DispatchTimeout(t *types.ReplicaTimeout) error {
	d.logger.With(log.LogParams{
		"timeout_type": t.Type,
		"duration":     t.Duration.String(),
	}).Debug("Dispatching timeout")
	replica, ok := d.Replicas.Get(t.Replica)
	if !ok {
		return ErrDestUnknown
	}
	bytes, err := json.Marshal(t)
	if err != nil {
		return ErrFailedMarshal
	}
	_, err = d.sendReq(replica, "/timeout", bytes)
	if err != nil {
		return err
	}
	return nil
}

// StopReplica should be called to direct the replica to stop running
func (d *Dispatcher) StopReplica(replica types.ReplicaID) error {
	replicaS, ok := d.Replicas.Get(replica)
	if !ok {
		return ErrDestUnknown
	}
	return d.sendDirective(stopAction, replicaS)
}

// StartReplica should be called to direct the replica to start running
func (d *Dispatcher) StartReplica(replica types.ReplicaID) error {
	replicaS, ok := d.Replicas.Get(replica)
	if !ok {
		return ErrDestUnknown
	}
	return d.sendDirective(startAction, replicaS)
}

// RestartReplica should be called to direct the replica to restart
func (d *Dispatcher) RestartReplica(replica types.ReplicaID) error {
	replicaS, ok := d.Replicas.Get(replica)
	if !ok {
		return ErrDestUnknown
	}
	return d.sendDirective(restartAction, replicaS)
}

// RestartAll restarts all the replicas
func (d *Dispatcher) RestartAll() error {
	errCh := make(chan error, d.Replicas.Cap())
	for _, r := range d.Replicas.Iter() {
		go func(errCh chan error, replica *types.Replica) {
			errCh <- d.sendDirective(restartAction, replica)
		}(errCh, r)
	}
	for i := 0; i < d.Replicas.Cap(); i++ {
		err := <-errCh
		if err != nil {
			return err
		}
	}
	return nil
}

func (d *Dispatcher) sendDirective(directive *directiveMessage, to *types.Replica) error {
	d.logger.With(log.LogParams{
		"action":  directive.Action,
		"replica": to.ID,
	}).Info("Dispatching directive!")
	bytes, err := json.Marshal(directive)
	if err != nil {
		return ErrFailedMarshal
	}
	_, err = d.sendReq(to, "/directive", bytes)
	if err != nil {
		return err
	}
	return nil
}

func (d *Dispatcher) sendReq(to *types.Replica, path string, msg []byte) (string, error) {
	d.lock.Lock()
	client, ok := d.clients[to.ID]
	if !ok {
		// Creating a keep-alive client
		client = &http.Client{
			Transport: &http.Transport{
				MaxIdleConnsPerHost: 2,
				MaxConnsPerHost:     1,
			},
		}
		d.clients[to.ID] = client
	}
	d.lock.Unlock()

	req, err := http.NewRequest(http.MethodPost, "http://"+to.Addr+path, bytes.NewBuffer([]byte(msg)))
	if err != nil {
		return "", ErrSendFailed
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", ErrSendFailed
	}
	defer resp.Body.Close()
	statusOK := resp.StatusCode >= 200 && resp.StatusCode < 300
	if statusOK {
		bodyB, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return "", ErrResponseReadFail
		}
		return string(bodyB), nil
	}
	return "", ErrBadResponse
}
