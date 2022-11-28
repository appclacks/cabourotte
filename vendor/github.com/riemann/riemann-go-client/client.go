// A Riemann client for Go, featuring concurrency, sending events and state updates, queries
//
// Copyright (C) 2014 by Christopher Gilbert <christopher.john.gilbert@gmail.com>
package riemanngo

import (
	"github.com/riemann/riemann-go-client/proto"
)

// Client is an interface to a generic client
type Client interface {
	Send(message *proto.Msg) (*proto.Msg, error)
	Connect() error
	Close() error
}

// IndexClient is an interface to a generic Client for index queries
type IndexClient interface {
	QueryIndex(q string) ([]Event, error)
}

// request encapsulates a request to send to the Riemann server
type request struct {
	message    *proto.Msg
	responseCh chan response
}

// response encapsulates a response from the Riemann server
type response struct {
	message *proto.Msg
	err     error
}

// SendEvent send an event using a client
func SendEvent(c Client, e *Event) (*proto.Msg, error) {
	return SendEvents(
		c, &([]Event{*e}),
	)
}

// SendEvents send multiple events using a client
func SendEvents(c Client, e *[]Event) (*proto.Msg, error) {
	buff := make(
		[]*proto.Event, len(*e),
	)

	for i, elem := range *e {
		epb, err := EventToProtocolBuffer(
			&elem,
		)

		if err != nil {
			return nil, err
		}

		buff[i] = epb
	}

	message := new(proto.Msg)
	message.Events = buff

	return c.Send(message)
}
