package memorystore

import (
	"testing"
	"time"

	"go.uber.org/zap"

	"cabourotte/healthcheck"
)

func TestMemoryExporter(t *testing.T) {
	store := NewMemoryStore(zap.NewExample())
	result := &healthcheck.Result{
		Name:      "foo",
		Success:   true,
		Timestamp: time.Now(),
		Message:   "message",
	}
	store.Add(result)
	resultList := store.List()
	if resultList[0] != *result {
		t.Fatalf("Invalid result content")
	}
	if len(resultList) != 1 {
		t.Fatalf("Invalid result list size: %d", len(resultList))
	}
	expiredResult := &healthcheck.Result{
		Name:      "bar",
		Success:   true,
		Timestamp: time.Now().Add(time.Minute * time.Duration(-5)),
		Message:   "message",
	}
	store.Add(expiredResult)
	resultList = store.List()
	if len(resultList) != 2 {
		t.Fatalf("Invalid result list size: %d", len(resultList))
	}
	store.Purge()
	resultList = store.List()
	if resultList[0] != *result {
		t.Fatalf("Invalid result content")
	}
	if len(resultList) != 1 {
		t.Fatalf("Invalid result list size: %d", len(resultList))
	}
}
