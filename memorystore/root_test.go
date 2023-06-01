package memorystore

import (
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/appclacks/cabourotte/healthcheck"
)

func TestMemoryExporter(t *testing.T) {
	store := NewMemoryStore(zap.NewExample())
	ts := time.Now()
	result := &healthcheck.Result{
		Name:                 "foo",
		Success:              true,
		HealthcheckTimestamp: ts.Unix(),
		Message:              "message",
	}
	store.Add(result)
	resultList := store.List()
	if !resultList[0].Equals(*result) {
		t.Fatalf("Invalid result content")
	}
	if len(resultList) != 1 {
		t.Fatalf("Invalid result list size: %d", len(resultList))
	}
	ts = time.Now().Add(time.Minute * time.Duration(-5))
	expiredResult := &healthcheck.Result{
		Name:                 "bar",
		Success:              true,
		HealthcheckTimestamp: ts.Unix(),
		Message:              "message",
	}
	store.Add(expiredResult)
	resultList = store.List()
	if len(resultList) != 2 {
		t.Fatalf("Invalid result list size: %d", len(resultList))
	}
	store.Purge()
	resultList = store.List()
	if !resultList[0].Equals(*result) {
		t.Fatalf("Invalid result content")
	}
	if len(resultList) != 1 {
		t.Fatalf("Invalid result list size: %d", len(resultList))
	}
}
