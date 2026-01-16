package sdk

import "errors"

var (
	// ErrPersonaNotFound is returned when a requested persona does not exist.
	ErrPersonaNotFound = errors.New("persona not found")
	// ErrAppNotFound is returned when a requested app does not exist within a persona.
	ErrAppNotFound = errors.New("app not found")
	// ErrKeyNotFound is returned when a requested key does not exist within an app.
	ErrKeyNotFound = errors.New("key not found")
)

// SystemPersona is the reserved ID for global/system-level data.
const SystemPersona = "_system"

// --- Functional Interfaces (Interface Segregation) ---

// KVReader defines the basic read operations for the store.
type KVReader interface {
	Get(personaID, appID, key string) (any, error)
}

// KVWriter defines the basic write and delete operations for the store.
type KVWriter interface {
	Set(personaID, appID, key string, val any) error
	Delete(personaID, appID, key string) error
}

// AppEnumeration allows discovering personas and apps.
type AppEnumeration interface {
	GetPersonas() ([]string, error)
	GetApps(personaID string) ([]string, error)
}

// BatchExporter allows retrieving bulk data.
type BatchExporter interface {
	GetAppStore(personaID, appID string) (map[string]any, error)
	DumpApp(appID string) (map[string]map[string]any, error)
}

// GlobalSearcher allows searching for keys across all personas.
type GlobalSearcher interface {
	GetGlobal(appID, key string) (any, string, error)
}

// Orchestrator handles higher-level data operations like moves.
type Orchestrator interface {
	Move(srcPersona, dstPersona, appID, key string) error
}

// --- Composite Interfaces ---

// CelerixStore is the primary interface for interacting with the data store.
// It combines all functional interfaces for a complete storage experience.
type CelerixStore interface {
	KVReader
	KVWriter
	AppEnumeration
	BatchExporter
	GlobalSearcher
	Orchestrator

	// App returns an AppScope that simplifies operations by "pinning" a persona and app.
	App(personaID, appID string) AppScope
}

// AppScope provides a simplified, scoped interface for a specific persona and app.
type AppScope interface {
	Get(key string) (any, error)
	Set(key string, val any) error
	Delete(key string) error
	// Vault returns a VaultScope for client-side encrypted storage.
	Vault(masterKey []byte) any
}

// VaultScope provides a scoped interface for performing client-side encryption.
type VaultScope interface {
	Get(key string) (string, error)
	Set(key string, plaintext string) error
}
