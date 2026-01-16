// Package engine defines the core storage engine for the Celerix Store.
package engine

import "errors"

// Standard errors for the engine.
// Note: We keep these here for internal engine use, but SDK will have its own
// or we can alias them if we want exact matching.
var (
	ErrPersonaNotFound = errors.New("persona not found")
	ErrAppNotFound     = errors.New("app not found")
	ErrKeyNotFound     = errors.New("key not found")
)

// SystemPersona is the reserved ID for global/system-level data.
const SystemPersona = "_system"

// AppScope and VaultScope interfaces are now defined in pkg/sdk.
// We use 'any' or specific types if needed, but the engine implementations
// will satisfy the sdk interfaces.
