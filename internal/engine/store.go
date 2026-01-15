package engine

import "errors"

var (
	ErrPersonaNotFound = errors.New("persona not found")
	ErrAppNotFound     = errors.New("app not found")
	ErrKeyNotFound     = errors.New("key not found")
)

const SystemPersona = "_system"

// CelerixStore is the "Contract".
// Both the Server and the Embedded engine must satisfy this.
type CelerixStore interface {
	Get(personaID, appID, key string) (any, error)
	Set(personaID, appID, key string, val any) error
	Delete(personaID, appID, key string) error

	// GetApps get apps namespace
	GetApps(personaID string) ([]string, error)
	// GetPersonas get personas namespace
	GetPersonas() ([]string, error)

	// GetAppStore Bulk (Used for Migration & Backups)
	GetAppStore(personaID, appID string) (map[string]any, error)

	// --- New features ---

	// DumpApp returns data for a specific app across ALL personas.
	// Returns map[personaID]map[key]value
	DumpApp(appID string) (map[string]map[string]any, error)

	// GetGlobal searches for a key across all personas for a specific app.
	// Returns (value, personaID, error)
	GetGlobal(appID, key string) (any, string, error)

	// Move transfers a key from one persona to another within the same app.
	Move(srcPersona, dstPersona, appID, key string) error

	// App creates a scoped store for a specific persona and app.
	App(personaID, appID string) AppScope
}

// AppScope is a scoped interface for a specific persona and app.
type AppScope interface {
	Get(key string) (any, error)
	Set(key string, val any) error
	Delete(key string) error
	Vault(masterKey []byte) VaultScope
}

// VaultScope is a scoped interface for encrypted storage.
type VaultScope interface {
	Get(key string) (string, error)
	Set(key string, plaintext string) error
}
