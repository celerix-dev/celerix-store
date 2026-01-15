// Package engine defines the core interfaces and types for the Celerix Store.
package engine

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

// CelerixStore is the primary interface for interacting with the data store.
// Both the local embedded engine and the remote network client implement this contract.
type CelerixStore interface {
	// Get retrieves a value for a specific persona, app, and key.
	Get(personaID, appID, key string) (any, error)
	// Set stores a value for a specific persona, app, and key.
	Set(personaID, appID, key string, val any) error
	// Delete removes a key and its value from a specific persona and app.
	Delete(personaID, appID, key string) error

	// GetApps returns a list of all app IDs belonging to a specific persona.
	GetApps(personaID string) ([]string, error)
	// GetPersonas returns a list of all persona IDs in the store.
	GetPersonas() ([]string, error)

	// GetAppStore returns all keys and values for a specific persona and app.
	// Useful for migrations, exports, or batch processing.
	GetAppStore(personaID, appID string) (map[string]any, error)

	// DumpApp retrieves data for a specific application ID across ALL personas.
	// Returns a map keyed by personaID.
	DumpApp(appID string) (map[string]map[string]any, error)

	// GetGlobal searches for a key across all personas for a specific app ID.
	// Returns the value and the personaID of the owner.
	GetGlobal(appID, key string) (any, string, error)

	// Move transfers a key and its value from a source persona to a destination persona.
	Move(srcPersona, dstPersona, appID, key string) error

	// App returns an AppScope that simplifies operations by "pinning" a persona and app.
	App(personaID, appID string) AppScope
}

// AppScope provides a simplified, scoped interface for a specific persona and app.
type AppScope interface {
	// Get retrieves a value using the pinned persona and app.
	Get(key string) (any, error)
	// Set stores a value using the pinned persona and app.
	Set(key string, val any) error
	// Delete removes a key using the pinned persona and app.
	Delete(key string) error
	// Vault returns a VaultScope for client-side encrypted storage.
	// We use any here to avoid hard-coding a specific return type in the shared interface,
	// allowing implementations (like the SDK) to return their own types.
	Vault(masterKey []byte) any
}

// VaultScope provides a scoped interface for performing client-side encryption.
// Data is encrypted before being sent to the store and decrypted upon retrieval.
type VaultScope interface {
	// Get retrieves and decrypts a value using the provided master key.
	Get(key string) (string, error)
	// Set encrypts and stores a value using the provided master key.
	Set(key string, plaintext string) error
}
