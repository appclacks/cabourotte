package memorystore

import (
	"context"
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
	store.Add(context.Background(), result)
	resultList := store.List(context.Background())
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
	store.Add(context.Background(), expiredResult)
	resultList = store.List(context.Background())
	if len(resultList) != 2 {
		t.Fatalf("Invalid result list size: %d", len(resultList))
	}
	store.Purge(context.Background())
	resultList = store.List(context.Background())
	if !resultList[0].Equals(*result) {
		t.Fatalf("Invalid result content")
	}
	if len(resultList) != 1 {
		t.Fatalf("Invalid result list size: %d", len(resultList))
	}
}
