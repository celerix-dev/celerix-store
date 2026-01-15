package engine

import (
	"sync"
)

// MemStore is our thread-safe "Liquid Data" engine.
type MemStore struct {
	mu sync.RWMutex
	// Structure: [personaID][appID][key]value
	data      map[string]map[string]map[string]any
	persister *Persistence
	wg        sync.WaitGroup
}

// NewMemStore initializes a store.
// It accepts existing data (from LoadAll) and a persister.
func NewMemStore(initialData map[string]map[string]map[string]any, p *Persistence) *MemStore {
	if initialData == nil {
		initialData = make(map[string]map[string]map[string]any)
	}
	return &MemStore{
		data:      initialData,
		persister: p,
		wg:        sync.WaitGroup{},
	}
}

// Wait waits for all background persistence tasks to complete.
func (m *MemStore) Wait() {
	m.wg.Wait()
}

// --- Interface Implementation ---

func (m *MemStore) Get(personaID, appID, key string) (any, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	persona, ok := m.data[personaID]
	if !ok {
		return nil, ErrPersonaNotFound
	}

	app, ok := persona[appID]
	if !ok {
		return nil, ErrAppNotFound
	}

	val, ok := app[key]
	if !ok {
		return nil, ErrKeyNotFound
	}

	return val, nil
}

func (m *MemStore) Set(personaID, appID, key string, val any) error {
	m.mu.Lock()
	if m.data[personaID] == nil {
		m.data[personaID] = make(map[string]map[string]any)
	}
	if m.data[personaID][appID] == nil {
		m.data[personaID][appID] = make(map[string]any)
	}

	m.data[personaID][appID][key] = val

	// Deep copy the persona's state to save safely in background
	currentPersonaData := m.copyPersonaData(personaID)
	m.mu.Unlock()

	// Persist in background
	if m.persister != nil {
		m.wg.Add(1)
		go func(pID string, data map[string]map[string]any) {
			defer m.wg.Done()
			m.persister.SavePersona(pID, data)
		}(personaID, currentPersonaData)
	}
	return nil
}

func (m *MemStore) Delete(personaID, appID, key string) error {
	m.mu.Lock()
	if p, ok := m.data[personaID]; ok {
		if a, ok := p[appID]; ok {
			delete(a, key)
		}
	}
	// Deep copy the persona's state to save safely in background
	currentPersonaData := m.copyPersonaData(personaID)
	m.mu.Unlock()

	if m.persister != nil {
		m.wg.Add(1)
		go func(pID string, data map[string]map[string]any) {
			defer m.wg.Done()
			m.persister.SavePersona(pID, data)
		}(personaID, currentPersonaData)
	}
	return nil
}

// copyPersonaData creates a deep copy of a persona's data.
// It MUST be called while holding m.mu.Lock or m.mu.RLock.
func (m *MemStore) copyPersonaData(personaID string) map[string]map[string]any {
	original, ok := m.data[personaID]
	if !ok {
		return nil
	}

	personaCopy := make(map[string]map[string]any)
	for appID, appData := range original {
		appCopy := make(map[string]any)
		for k, v := range appData {
			appCopy[k] = v
		}
		personaCopy[appID] = appCopy
	}
	return personaCopy
}

func (m *MemStore) GetPersonas() ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var list []string
	for id := range m.data {
		list = append(list, id)
	}
	return list, nil
}

func (m *MemStore) GetApps(personaID string) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var list []string
	if apps, ok := m.data[personaID]; ok {
		for appID := range apps {
			list = append(list, appID)
		}
	}
	return list, nil
}

func (m *MemStore) GetAppStore(personaID, appID string) (map[string]any, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if p, ok := m.data[personaID]; ok {
		if a, ok := p[appID]; ok {
			// Return a copy to prevent external mutation of the internal map
			copy := make(map[string]any)
			for k, v := range a {
				copy[k] = v
			}
			return copy, nil
		}
	}
	return nil, ErrAppNotFound
}
