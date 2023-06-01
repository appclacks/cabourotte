package http

import (
	"crypto/tls"
	"crypto/x509"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"go.uber.org/zap"

	"github.com/appclacks/cabourotte/healthcheck"
	"github.com/appclacks/cabourotte/memorystore"
	"github.com/appclacks/cabourotte/prometheus"
)

func TestStartStop(t *testing.T) {
	prom, err := prometheus.New()
	if err != nil {
		t.Fatalf("Error creating prometheus component :\n%v", err)
	}
	logger := zap.NewExample()
	healthcheck, err := healthcheck.New(logger, make(chan *healthcheck.Result, 10), prom, []string{})
	if err != nil {
		t.Fatalf("Fail to create the healthcheck component\n%v", err)
	}
	component, err := New(logger, memorystore.NewMemoryStore(logger), prom, &Configuration{Host: "127.0.0.1", Port: 2000}, healthcheck)
	if err != nil {
		t.Fatalf("Fail to create the component\n%v", err)
	}
	err = component.Start()
	if err != nil {
		t.Fatalf("Fail to start the component\n%v", err)
	}
	resp, err := http.Get("http://localhost:2000/metrics")
	if err != nil {
		t.Fatalf("HTTP error\n%v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("Was expected a 200 status")
	}
	err = component.Stop()
	if err != nil {
		t.Fatalf("Fail to stop the component\n%v", err)
	}
}

func TestStartStopTLS(t *testing.T) {
	logger := zap.NewExample()
	prom, err := prometheus.New()
	if err != nil {
		t.Fatalf("Error creating prometheus component :\n%v", err)
	}
	healthcheck, err := healthcheck.New(logger, make(chan *healthcheck.Result, 10), prom, []string{})
	if err != nil {
		t.Fatalf("Fail to create the healthcheck component\n%v", err)
	}
	component, err := New(
		logger, memorystore.NewMemoryStore(logger),
		prom,
		&Configuration{
			Host:   "127.0.0.1",
			Port:   2000,
			Key:    "../test/key.pem",
			Cert:   "../test/cert.pem",
			Cacert: "../test/cert.pem",
		},
		healthcheck,
	)
	if err != nil {
		t.Fatalf("Fail to create the component\n%v", err)
	}
	err = component.Start()
	if err != nil {
		t.Fatalf("Fait to start component\n%v", err)
	}
	// http req
	resp, err := http.Get("http://localhost:2000/metrics")
	if err != nil {
		t.Fatalf("HTTP error\n%v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Fail reading response body\n%v", err)
	}
	if !strings.Contains(string(body), "Client sent an HTTP request to an HTTPS server.") {
		t.Fatalf("HTTP should not work")
	}
	if resp.StatusCode != 400 {
		t.Fatalf("Was expected a 400 status")
	}
	// https req
	cert, err := tls.LoadX509KeyPair("../test/cert.pem", "../test/key.pem")
	if err != nil {
		t.Fatalf("Fail to load certificates\n%v", err)
	}
	if err != nil {
		t.Fatalf("Fail to start the component\n%v", err)
	}
	caCert, err := os.ReadFile("../test/cert.pem")
	if err != nil {
		t.Fatalf("Fail to load the certificate\n%v", err)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			RootCAs:      caCertPool,
			Certificates: []tls.Certificate{cert},
		},
	}
	client := http.Client{
		Transport: transport,
	}
	resp, err = client.Get("https://localhost:2000/metrics")
	if err != nil {
		t.Fatalf("HTTP error\n%v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("Was expected a 200 status")
	}
	// insecure
	transport = &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	client = http.Client{
		Transport: transport,
	}
	_, err = client.Get("https://localhost:2000/metrics")
	if err == nil {
		t.Fatalf("Was expecting an error")
	}
	err = component.Stop()
	if err != nil {
		t.Fatalf("Fail to stop the component\n%v", err)
	}
}
