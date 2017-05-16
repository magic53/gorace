// The purpose of this package is to prevent race conditions that cause goroutine
// panics on long running api calls by encapsulating goroutine logic in a shielded
// and cancelable implementation. This package is designed specifically for use
// with contexts.
package gorace

import (
	"context"
)

// GoCancelable contract
type GoCancelable interface {
	// Cancel closes the internal channel and returns true. If the
	// cancelable is already canceled this returns false
	Cancel() bool
	// Send a result to channel listeners. This method requires calling
	// Cancel() manually when done to free up resources
	Send(result interface{})
	// Receive returns the internal communication channel
	Receive() <-chan interface{}
	// Start runs the userdefined handler func and returns the internal
	// channel. The specified context is passed through to the handler func
	Start(ctx context.Context) GoCancelable
	// StartBackground starts the canceled on a goroutine. Equivalent to
	// go cancelable.Start(ctx)
	StartBackground(ctx context.Context) GoCancelable
	// LastResult returns the last value sent successfully on the channel.
	// Note this value isn't updated after Cancel() is called
	LastResult() interface{}
	// IsCanceled returns true if the cancelable is canceled otherwise
	// returns false
	IsCanceled() bool
}

// GoRace creates and returns a cancelable instance. The specified handler
// will be called in Start
func GoRace(handler func(ctx context.Context, cancelable GoCancelable)) GoCancelable {
	send := make(chan interface{}, 1)
	return &goCancelable{handler: handler, send: send}
}

// Implementation for the gorace framework
type goCancelable struct {
	handler    func(ctx context.Context, cancelable GoCancelable)
	send       chan interface{}
	canceled   bool
	started    bool
	lastResult interface{}
}

// Cancel closes the send channel and sets the state to canceled
func (gc *goCancelable) Cancel() bool {
	if !gc.IsCanceled() {
		gc.canceled = true
		close(gc.send)
		return true
	} else {
		return false
	}
}

// IsCanceled returns true if the cancelable is already canceled otherwise returns false
func (gc *goCancelable) IsCanceled() bool {
	return gc.canceled
}

// Send stores the last result and sends the result on the cancelable's channel
func (gc *goCancelable) Send(result interface{}) {
	if !gc.IsCanceled() {
		gc.lastResult = result
		gc.send <- result
	}
}

// Receive returns the receive channel
func (gc *goCancelable) Receive() <-chan interface{} {
	return gc.send
}

// Start calls the associated gorace handler if the cancelable has not been canceled or started. If the cancelable
// is canceled or has already started this call does nothing
func (gc *goCancelable) Start(ctx context.Context) GoCancelable {
	if !gc.IsCanceled() && !gc.started {
		defer gc.Cancel() // Clean up resources after handler is called
		gc.started = true
		// Call the handler
		gc.handler(ctx, gc)
	}
	return gc
}

// StartBackground calls Start on a goroutine with the specified context
func (gc *goCancelable) StartBackground(ctx context.Context) GoCancelable {
	go gc.Start(ctx)
	return gc
}

// LastResult returns the last successful result sent on the cancelable's channel. This does not return
// values attempted to be sent after the cancelable is canceled
func (gc *goCancelable) LastResult() interface{} {
	return gc.lastResult
}
