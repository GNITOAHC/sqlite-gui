package app

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"

	"sqlite-gui/pkg/database"
)

var (
	ErrConnectionExists = errors.New("connection already exists")
	ErrConnectionMiss   = errors.New("connection not found")
)

// databaseFactory creates fresh database instances (e.g. sqlite.New).
type databaseFactory func() database.Database

type connectionEntry struct {
	name       string
	connString string
	db         database.Database
}

type ConnectionManager struct {
	mu          sync.RWMutex
	connections map[string]*connectionEntry
	defaultName string
	factory     databaseFactory
}

type ConnectionInfo struct {
	Name       string `json:"name"`
	ConnString string `json:"connString"`
	Default    bool   `json:"default"`
}

func NewConnectionManager(factory databaseFactory) *ConnectionManager {
	return &ConnectionManager{
		connections: make(map[string]*connectionEntry),
		factory:     factory,
	}
}

func (m *ConnectionManager) Add(ctx context.Context, name, connString string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.connections[name]; exists {
		return fmt.Errorf("%w: %s", ErrConnectionExists, name)
	}
	db := m.factory()
	if err := db.Connect(ctx, connString); err != nil {
		return err
	}
	m.connections[name] = &connectionEntry{name: name, connString: connString, db: db}
	if m.defaultName == "" {
		m.defaultName = name
	}
	return nil
}

func (m *ConnectionManager) Get(name string) (database.Database, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if name == "" {
		name = m.defaultName
	}
	entry, ok := m.connections[name]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrConnectionMiss, name)
	}
	return entry.db, nil
}

func (m *ConnectionManager) Default() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.defaultName
}

func (m *ConnectionManager) List() []ConnectionInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	results := make([]ConnectionInfo, 0, len(m.connections))
	for name, entry := range m.connections {
		results = append(results, ConnectionInfo{
			Name:       name,
			ConnString: entry.connString,
			Default:    name == m.defaultName,
		})
	}
	sort.Slice(results, func(i, j int) bool { return results[i].Name < results[j].Name })
	return results
}

func (m *ConnectionManager) CloseAll() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var firstErr error
	for name, entry := range m.connections {
		if err := entry.db.Close(); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("close %s: %w", name, err)
		}
	}
	m.connections = make(map[string]*connectionEntry)
	m.defaultName = ""
	return firstErr
}
