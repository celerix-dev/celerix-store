package engine

import "errors"

var (
	ErrPersonaNotFound = errors.New("persona not found")
	ErrAppNotFound     = errors.New("app not found")
	ErrKeyNotFound     = errors.New("key not found")
)

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
}
