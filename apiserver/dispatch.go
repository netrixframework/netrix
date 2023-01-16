package apiserver

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

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
	// ErrSendFailed is returned when the request could not be created
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

// SendMessage should be invoked to send the message to the intended replica.
// The MessageID is cached and subsequent invocations for the same message will result in a no-op
func (a *APIServer) SendMessage(msg *types.Message) error {
	a.lock.Lock()
	_, ok := a.dispatchedMessages[msg.ID]
	if ok {
		a.lock.Unlock()
		return ErrDuplicateDispatch
	}
	a.dispatchedMessages[msg.ID] = true
	_, block := a.resetReplicas[msg.To]
	a.lock.Unlock()
	if block {
		return nil
	}

	a.Logger.With(log.LogParams{
		// "message": msg.ParsedMessage.String(),
		"to": msg.To,
		"id": msg.ID,
	}).Debug("Sending message")

	replica, ok := a.ctx.Replicas.Get(msg.To)
	if !ok {
		return ErrDestUnknown
	}
	bytes, err := json.Marshal(msg)
	if err != nil {
		return ErrFailedMarshal
	}
	resp, err := a.sendReq(replica, "/message", bytes)
	a.Logger.With(log.LogParams{
		"resp": resp,
		"to":   replica.ID,
		"id":   msg.ID,
		"addr": replica.Addr,
		"data": string(bytes),
	}).Debug("Dispatched message to replica")
	if err != nil {
		a.Logger.With(log.LogParams{
			"error": err,
		}).Debug("Failed to dispatch message")
	}
	return err
}

// ForgetSendMessages clears the delivered message cache.
func (a *APIServer) ForgetSentMessages() {
	a.lock.Lock()
	a.dispatchedMessages = make(map[types.MessageID]bool)
	a.lock.Unlock()
}

// SendTimeout sends a timeout message to the replica.
// This is the equivalent of ending a timeout at the replica.
func (d *APIServer) SendTimeout(t *types.ReplicaTimeout) error {
	d.Logger.With(log.LogParams{
		"timeout_type": t.Type,
		"duration":     t.Duration.String(),
	}).Debug("Dispatching timeout")
	replica, ok := d.ctx.Replicas.Get(t.Replica)
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

// StopReplica sends the Stop directive to the replica
func (d *APIServer) StopReplica(replica types.ReplicaID) error {
	replicaS, ok := d.ctx.Replicas.Get(replica)
	if !ok {
		return ErrDestUnknown
	}
	return d.sendDirective(stopAction, replicaS)
}

// StartReplica sends the Start directive to the replica
func (d *APIServer) StartReplica(replica types.ReplicaID) error {
	replicaS, ok := d.ctx.Replicas.Get(replica)
	if !ok {
		return ErrDestUnknown
	}
	return d.sendDirective(startAction, replicaS)
}

// RestartReplica sends the Restart directive to the replica
func (d *APIServer) RestartReplica(replica types.ReplicaID) error {
	replicaS, ok := d.ctx.Replicas.Get(replica)
	if !ok {
		return ErrDestUnknown
	}
	d.lock.Lock()
	d.resetReplicas[replica] = true
	d.lock.Unlock()
	return d.sendDirective(restartAction, replicaS)
}

// RestartAll sends a Restart directive to all replicas
func (d *APIServer) RestartAll() error {
	errCh := make(chan error, d.ctx.Replicas.Cap())
	for _, r := range d.ctx.Replicas.Iter() {
		d.lock.Lock()
		d.resetReplicas[r.ID] = true
		d.lock.Unlock()
	}
	for _, r := range d.ctx.Replicas.Iter() {
		go func(errCh chan error, replica *types.Replica) {
			errCh <- d.sendDirective(restartAction, replica)
		}(errCh, r)
	}
	for i := 0; i < d.ctx.Replicas.Cap(); i++ {
		err := <-errCh
		if err != nil {
			return err
		}
	}
	return nil
}

func (d *APIServer) sendDirective(directive *directiveMessage, to *types.Replica) error {
	d.Logger.With(log.LogParams{
		"action":  directive.Action,
		"replica": to.ID,
	}).Debug("Dispatching directive!")

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

func (a *APIServer) sendReq(to *types.Replica, path string, msg []byte) (string, error) {
	a.lock.Lock()
	client, ok := a.clients[to.ID]
	if !ok {
		// Creating a keep-alive client
		client = &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				MaxIdleConnsPerHost: 1024,
			},
		}
		a.clients[to.ID] = client
	}
	a.lock.Unlock()

	resp, err := client.Post("http://"+to.Addr+path, "application/json", bytes.NewBuffer([]byte(msg)))
	if err != nil {
		return "", fmt.Errorf("send failed: %s", err)
	}
	bodyB, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()

	if err != nil {
		return "", fmt.Errorf("failed to read response: %s", err)
	}
	statusOK := resp.StatusCode >= 200 && resp.StatusCode < 300
	if statusOK {
		return string(bodyB), nil
	}
	return "", fmt.Errorf("bad response: %d", resp.StatusCode)
}
