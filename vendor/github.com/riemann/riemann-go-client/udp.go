package riemanngo

import (
	"fmt"
	"net"
	"time"

	pb "github.com/golang/protobuf/proto"
	"github.com/riemann/riemann-go-client/proto"
	"gopkg.in/tomb.v2"
)

// UDPClient is a type that implements the Client interface
type UDPClient struct {
	addr         string
	conn         net.Conn
	requestQueue chan request
	timeout      time.Duration
	t            tomb.Tomb
}

// MaxUDPSize is the maximum allowed size of a UDP packet before automatically failing the send
const MaxUDPSize = 16384

// NewUDPClient - Factory
func NewUDPClient(addr string, timeout time.Duration) *UDPClient {
	t := &UDPClient{
		addr:         addr,
		requestQueue: make(chan request),
		timeout:      timeout,
	}
	return t
}

// Connect the udp client
func (c *UDPClient) Connect() error {
	c.t.Go(func() error {
		return c.runRequestQueue()
	})
	udp, err := net.DialTimeout("udp", c.addr, c.timeout)
	if err != nil {
		return err
	}
	c.conn = udp
	return nil
}

// Send queues a request to send a message to the server
func (c *UDPClient) Send(message *proto.Msg) (*proto.Msg, error) {
	responseCh := make(chan response)
	c.requestQueue <- request{message, responseCh}
	r := <-responseCh
	return r.message, r.err
}

// Close will close the UDPClient
func (c *UDPClient) Close() error {
	c.t.Kill(nil)
	_ = c.t.Wait()
	close(c.requestQueue)
	err := c.conn.Close()
	return err
}

// runRequestQueue services the UDPClient request queue
func (c *UDPClient) runRequestQueue() error {

	for {
		select {
		case <-c.t.Dying():
			return nil
		case req := <-c.requestQueue:
			message := req.message
			responseCh := req.responseCh

			msg, err := c.execRequest(message)

			responseCh <- response{msg, err}
		}

	}
}

// execRequest will send a UDP message to Riemann
func (c *UDPClient) execRequest(message *proto.Msg) (*proto.Msg, error) {
	err := c.conn.SetDeadline(time.Now().Add(c.timeout))
	if err != nil {
		return nil, err
	}
	data, err := pb.Marshal(message)
	if err != nil {
		return nil, err
	}
	if len(data) > MaxUDPSize {
		return nil, fmt.Errorf("unable to send message, too large for udp")
	}
	if _, err = c.conn.Write(data); err != nil {
		return nil, err
	}
	return nil, nil
}
