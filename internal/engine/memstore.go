// Package engine defines the core interfaces and types for the Celerix Store.
package engine

import (
	"fmt"
	"sync"

	"github.com/celerix-dev/celerix-store/internal/vault"
	"github.com/celerix-dev/celerix-store/pkg/sdk"
)

// MemStore is a thread-safe, in-memory implementation of the CelerixStore interface.
// It supports asynchronous persistence to JSON files.
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

// Get retrieves a value for a specific persona, app, and key.
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

func (m *MemStore) DumpApp(appID string) (map[string]map[string]any, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]map[string]any)
	for personaID, apps := range m.data {
		if appData, ok := apps[appID]; ok {
			appCopy := make(map[string]any)
			for k, v := range appData {
				appCopy[k] = v
			}
			result[personaID] = appCopy
		}
	}
	return result, nil
}

func (m *MemStore) GetGlobal(appID, key string) (any, string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for personaID, apps := range m.data {
		if appData, ok := apps[appID]; ok {
			if val, ok := appData[key]; ok {
				return val, personaID, nil
			}
		}
	}
	return nil, "", ErrKeyNotFound
}

func (m *MemStore) Move(srcPersona, dstPersona, appID, key string) error {
	m.mu.Lock()
	// 1. Check if source exists
	srcP, ok := m.data[srcPersona]
	if !ok {
		m.mu.Unlock()
		return ErrPersonaNotFound
	}
	srcA, ok := srcP[appID]
	if !ok {
		m.mu.Unlock()
		return ErrAppNotFound
	}
	val, ok := srcA[key]
	if !ok {
		m.mu.Unlock()
		return ErrKeyNotFound
	}

	// 2. Perform Move
	delete(srcA, key)
	if m.data[dstPersona] == nil {
		m.data[dstPersona] = make(map[string]map[string]any)
	}
	if m.data[dstPersona][appID] == nil {
		m.data[dstPersona][appID] = make(map[string]any)
	}
	m.data[dstPersona][appID][key] = val

	// 3. Prepare background persistence for BOTH personas
	srcCopy := m.copyPersonaData(srcPersona)
	dstCopy := m.copyPersonaData(dstPersona)
	m.mu.Unlock()

	if m.persister != nil {
		m.wg.Add(2)
		go func() {
			defer m.wg.Done()
			m.persister.SavePersona(srcPersona, srcCopy)
		}()
		go func() {
			defer m.wg.Done()
			m.persister.SavePersona(dstPersona, dstCopy)
		}()
	}

	return nil
}

// --- Scoping Support ---

// App returns an AppScope that "pins" the persona and application for subsequent operations.
func (m *MemStore) App(personaID, appID string) sdk.AppScope {
	return &memAppScope{
		store:     m,
		personaID: personaID,
		appID:     appID,
	}
}

type memAppScope struct {
	store     *MemStore
	personaID string
	appID     string
}

func (a *memAppScope) Get(key string) (any, error) {
	return a.store.Get(a.personaID, a.appID, key)
}

func (a *memAppScope) Set(key string, val any) error {
	return a.store.Set(a.personaID, a.appID, key, val)
}

func (a *memAppScope) Delete(key string) error {
	return a.store.Delete(a.personaID, a.appID, key)
}

func (a *memAppScope) Vault(masterKey []byte) any {
	return &memVaultScope{
		app:       a,
		masterKey: masterKey,
	}
}

type memVaultScope struct {
	app       *memAppScope
	masterKey []byte
}

func (v *memVaultScope) Set(key string, plaintext string) error {
	ciphertext, err := vault.Encrypt(plaintext, v.masterKey)
	if err != nil {
		return err
	}
	return v.app.Set(key, ciphertext)
}

func (v *memVaultScope) Get(key string) (string, error) {
	val, err := v.app.Get(key)
	if err != nil {
		return "", err
	}
	cipherHex, ok := val.(string)
	if !ok {
		return "", fmt.Errorf("stored value is not a string")
	}
	return vault.Decrypt(cipherHex, v.masterKey)
}

func init() {
	sdk.RegisterEngine(&engineProvider{})
}

type engineProvider struct{}

func (e *engineProvider) NewPersistence(dir string) (sdk.Persistence, error) {
	return NewPersistence(dir)
}

func (e *engineProvider) NewMemStore(initialData map[string]map[string]map[string]any, p sdk.Persistence) sdk.CelerixStore {
	// We need to type assert Persistence back to our concrete type
	persister, _ := p.(*Persistence)
	return NewMemStore(initialData, persister)
}
