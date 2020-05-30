package exporter

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"

	"cabourotte/healthcheck"
	"cabourotte/memorystore"
)

func TestStartStop(t *testing.T) {
	mutex := &sync.RWMutex{}
	count := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mutex.Lock()
		count++
		mutex.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	port, err := strconv.ParseUint(strings.Split(ts.URL, ":")[2], 10, 16)
	if err != nil {
		t.Errorf("Error getting HTTP server port :\n%v", err)
	}
	chanResult := make(chan *healthcheck.Result, 10)
	logger := zap.NewExample()
	component := New(
		logger,
		memorystore.NewMemoryStore(logger),
		chanResult,
		&Configuration{
			HTTP: []HTTPConfiguration{
				HTTPConfiguration{
					Host:     "",
					Port:     uint32(port),
					Protocol: healthcheck.HTTP,
				},
			}})
	err = component.Start()
	if err != nil {
		t.Errorf("Error starting the component :\n%v", err)
	}
	chanResult <- &healthcheck.Result{
		Name:      "foo",
		Success:   true,
		Timestamp: time.Now(),
		Message:   "message",
	}
	success := false
	for i := 0; i < 10; i++ {
		time.Sleep(time.Millisecond * 100)
		mutex.RLock()
		if count == 1 {
			success = true
			break
		}
		mutex.RUnlock()
	}
	if !success {
		t.Errorf("The request counter is invalid")
	}
	err = component.Stop()
	if err != nil {
		t.Errorf("Error stopping the component :\n%v", err)
	}
}
