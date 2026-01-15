package sdk

import (
	"os"

	"github.com/celerix-dev/celerix-store/internal/engine"
)

// New initializes the store based on the environment.
// It returns the Interface, so the app doesn't care if it's local or remote.
func New(dataDir string) (engine.CelerixStore, error) {
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
	// This uses the same engine the server uses, but inside the app process.
	p, err := engine.NewPersistence(dataDir)
	if err != nil {
		return nil, err
	}

	allData, err := p.LoadAll()
	if err != nil {
		return nil, err
	}

	// Create a MemStore and inject the persistence
	return engine.NewMemStore(allData, p), nil
}
