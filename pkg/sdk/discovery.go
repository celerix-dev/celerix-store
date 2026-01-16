package sdk

import (
	"errors"
	"os"
)

// Persistence is a stub to avoid importing engine,
// but we need it for the interface.
// Actually we should move engine logic or use interfaces.
type Persistence interface {
	LoadAll() (map[string]map[string]map[string]any, error)
}

type EngineProvider interface {
	NewPersistence(dir string) (Persistence, error)
	NewMemStore(initialData map[string]map[string]map[string]any, p Persistence) CelerixStore
}

var provider EngineProvider

func RegisterEngine(p EngineProvider) {
	provider = p
}

// New initializes a CelerixStore based on the environment.
// It automatically detects whether to connect to a remote server (via CELERIX_STORE_ADDR)
// or initialize a local embedded engine.
func New(dataDir string) (CelerixStore, error) {
	// 1. Check if a Remote Store is defined in Environment Variables
	remoteAddr := os.Getenv("CELERIX_STORE_ADDR")

	if remoteAddr != "" {
		// Attempt to connect to the network service
		client, err := Connect(remoteAddr)
		if err == nil {
			return client, nil
		}
		// If the connection fails, we can either log a warning or fall back to local
	}

	// 2. Fallback to Embedded Mode
	if provider == nil {
		return nil, errors.New("embedded engine not registered")
	}

	p, err := provider.NewPersistence(dataDir)
	if err != nil {
		return nil, err
	}

	allData, err := p.LoadAll()
	if err != nil {
		return nil, err
	}

	// Create a MemStore and inject the persistence
	return provider.NewMemStore(allData, p), nil
}
