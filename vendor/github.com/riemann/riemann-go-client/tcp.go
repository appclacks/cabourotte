package riemanngo

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"io"
	"io/ioutil"
	"net"
	"time"

	pb "github.com/golang/protobuf/proto"
	"github.com/riemann/riemann-go-client/proto"
	"gopkg.in/tomb.v2"
)

// TCPClient is a type that implements the Client interface
type TCPClient struct {
	tls          bool
	addr         string
	conn         net.Conn
	requestQueue chan request
	timeout      time.Duration
	tlsConfig    *tls.Config
	t            tomb.Tomb
}

// NewTLSClient - Factory
func NewTLSClient(addr string, tlsConfig *tls.Config, timeout time.Duration) (*TCPClient, error) {
	t := &TCPClient{
		tls:          true,
		addr:         addr,
		tlsConfig:    tlsConfig,
		requestQueue: make(chan request),
		timeout:      timeout,
	}
	return t, nil
}

// GetTLSConfig returns a *tls.Config
func GetTLSConfig(serverName string, certPath string, keyPath string, insecure bool) (*tls.Config, error) {
	certFile, err := ioutil.ReadFile(certPath)
	if err != nil {
		return nil, err
	}

	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return nil, err
	}

	clientCertPool := x509.NewCertPool()
	clientCertPool.AppendCertsFromPEM(certFile)
	config := tls.Config{
		Certificates:       []tls.Certificate{cert},
		RootCAs:            clientCertPool,
		InsecureSkipVerify: insecure}
	if !insecure {
		config.ServerName = serverName
	}
	return &config, nil

}

// NewTCPClient - Factory
func NewTCPClient(addr string, timeout time.Duration) *TCPClient {
	t := &TCPClient{
		tls:          false,
		addr:         addr,
		requestQueue: make(chan request),
		timeout:      timeout,
	}
	return t
}

// Connect the tcp client
func (c *TCPClient) Connect() error {
	c.t.Go(func() error {
		return c.runRequestQueue()
	})
	connection, err := net.DialTimeout("tcp", c.addr, c.timeout)
	if err != nil {
		return err
	}
	if c.tls {
		tlsConn := tls.Client(connection, c.tlsConfig)
		err = tlsConn.Handshake()
		if err != nil {
			return err
		}
		c.conn = tlsConn
	} else {
		c.conn = connection
	}
	return nil
}

// Close will close the TCPClient
func (c *TCPClient) Close() error {
	c.t.Kill(nil)
	_ = c.t.Wait()
	close(c.requestQueue)
	err := c.conn.Close()
	return err
}

// Send queues a request to send a message to the server
func (c *TCPClient) Send(message *proto.Msg) (*proto.Msg, error) {
	responseCh := make(chan response)
	c.requestQueue <- request{message, responseCh}
	r := <-responseCh
	return r.message, r.err
}

// runRequestQueue services the TCPClient request queue
func (c *TCPClient) runRequestQueue() error {
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

// execRequest will send a TCP message to Riemann
func (c *TCPClient) execRequest(message *proto.Msg) (*proto.Msg, error) {
	err := c.conn.SetDeadline(time.Now().Add(c.timeout))
	if err != nil {
		return nil, err
	}
	msg := &proto.Msg{}
	data, err := pb.Marshal(message)
	if err != nil {
		return msg, err
	}
	b := new(bytes.Buffer)
	if err = binary.Write(b, binary.BigEndian, uint32(len(data))); err != nil {
		return msg, err
	}
	// send the msg length
	if _, err = c.conn.Write(b.Bytes()); err != nil {
		return msg, err
	}
	// send the msg
	if _, err = c.conn.Write(data); err != nil {
		return msg, err
	}
	var header uint32
	if err = binary.Read(c.conn, binary.BigEndian, &header); err != nil {
		return msg, err
	}
	response := make([]byte, header)
	if err = readMessages(c.conn, response); err != nil {
		return msg, err
	}
	if err = pb.Unmarshal(response, msg); err != nil {
		return msg, err
	}
	return msg, nil
}

// readMessages will read Riemann messages from the TCP connection
func readMessages(r io.Reader, p []byte) error {
	for len(p) > 0 {
		n, err := r.Read(p)
		p = p[n:]
		if err != nil {
			return err
		}
	}
	return nil
}

// QueryIndex query the server for events using the client
func (c *TCPClient) QueryIndex(q string) ([]Event, error) {
	err := c.conn.SetDeadline(time.Now().Add(c.timeout))
	if err != nil {
		return nil, err
	}
	query := &proto.Query{}
	query.String_ = pb.String(q)

	message := &proto.Msg{}
	message.Query = query

	response, err := c.Send(message)
	if err != nil {
		return nil, err
	}

	return ProtocolBuffersToEvents(response.GetEvents()), nil
}
