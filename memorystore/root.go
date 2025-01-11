package memorystore

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"go.uber.org/zap"
	"gopkg.in/tomb.v2"

	"github.com/appclacks/cabourotte/healthcheck"
	"go.opentelemetry.io/otel"
)

// MemoryStore A store containing the latest healthchecks results
type MemoryStore struct {
	TTL     time.Duration
	Logger  *zap.Logger
	Results map[string]*healthcheck.Result
	Tick    *time.Ticker

	t    tomb.Tomb
	lock sync.RWMutex
}

// NewMemoryStore creates a new memory store
func NewMemoryStore(logger *zap.Logger) *MemoryStore {
	return &MemoryStore{
		Logger:  logger,
		TTL:     time.Second * 120,
		Results: make(map[string]*healthcheck.Result),
	}
}

// Start starts the memory store
func (m *MemoryStore) Start() {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.Tick = time.NewTicker(time.Second * 30)
	m.t.Go(func() error {
		for {
			select {
			case <-m.Tick.C:
				m.Purge(context.Background())
			case <-m.t.Dying():
				return nil
			}
		}
	})
}

// Stop stops the memory store
func (m *MemoryStore) Stop() error {
	m.Tick.Stop()
	m.t.Kill(nil)
	err := m.t.Wait()
	if err != nil {
		return err
	}
	return nil
}

// Add a new Result to the store
func (m *MemoryStore) Add(ctx context.Context, result *healthcheck.Result) {
	tracer := otel.Tracer("memorystore")
	_, span := tracer.Start(ctx, "memorystore.add")
	defer span.End()
	m.lock.Lock()
	defer m.lock.Unlock()
	m.Results[result.Name] = result
}

// Purge the expired results
func (m *MemoryStore) Purge(ctx context.Context) {
	tracer := otel.Tracer("memorystore")
	_, span := tracer.Start(ctx, "memorystore.purge")
	defer span.End()
	m.lock.Lock()
	defer m.lock.Unlock()
	now := time.Now()
	for i := range m.Results {
		result := m.Results[i]
		checkTimestamp := time.Unix(result.HealthcheckTimestamp, 0)
		if now.After(checkTimestamp.Add(m.TTL)) {
			m.Logger.Info("expire healthcheck",
				zap.String("name", result.Name))
			delete(m.Results, result.Name)
		}
	}
}

// List returns the current value of the results
func (m *MemoryStore) List(ctx context.Context) []healthcheck.Result {
	tracer := otel.Tracer("memorystore")
	_, span := tracer.Start(ctx, "memorystore.list")
	defer span.End()
	m.lock.RLock()
	defer m.lock.RUnlock()
	result := make([]healthcheck.Result, 0, len(m.Results))
	for i := range m.Results {
		value := m.Results[i]
		result = append(result, *value)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result
}

// Get returns the current value for a healthcheck
func (m *MemoryStore) Get(ctx context.Context, name string) (healthcheck.Result, error) {
	tracer := otel.Tracer("memorystore")
	_, span := tracer.Start(ctx, "memorystore.get")
	defer span.End()
	m.lock.RLock()
	defer m.lock.RUnlock()
	if result, ok := m.Results[name]; ok {
		return *result, nil
	}
	return healthcheck.Result{}, fmt.Errorf("Result not found for healthcheck %s", name)
}
