package exporter

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
	store.add(result)
	resultList := store.list()
	if resultList[0] != *result {
		t.Errorf("Invalid result content")
	}
	if len(resultList) != 1 {
		t.Errorf("Invalid result list size: %d", len(resultList))
	}
	expiredResult := &healthcheck.Result{
		Name:      "bar",
		Success:   true,
		Timestamp: time.Now().Add(time.Minute * time.Duration(-5)),
		Message:   "message",
	}
	store.add(expiredResult)
	resultList = store.list()
	if len(resultList) != 2 {
		t.Errorf("Invalid result list size: %d", len(resultList))
	}
	store.purge()
	resultList = store.list()
	if resultList[0] != *result {
		t.Errorf("Invalid result content")
	}
	if len(resultList) != 1 {
		t.Errorf("Invalid result list size: %d", len(resultList))
	}
}
