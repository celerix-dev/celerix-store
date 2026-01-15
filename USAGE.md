# Celerix Store Usage Guide

This guide provides detailed examples and best practices for using the `celerix-store` library in your Go applications.

## Core Concepts

### Data Hierarchy
- **Persona:** Represents a user or a system entity. Each persona's data is stored in its own JSON file.
- **App:** A namespace within a persona. This allows multiple applications to store data for the same persona without collisions.
- **Key:** The specific identifier for a piece of data within an app namespace.

### The `_system` Persona
The `_system` persona is a reserved namespace for global application metadata, registry of users, or any data that isn't tied to a specific human user. It is treated as a first-class citizen and optimized for discovery.

---

## SDK Basics

### Initializing the Store
The SDK automatically switches between **Embedded** (local JSON files) and **Remote** (connecting to a `celerix-stored` daemon) based on the `CELERIX_STORE_ADDR` environment variable.

```go
import "github.com/celerix-dev/celerix-store/pkg/sdk"

func main() {
    // dataDir is used only if in Embedded mode
    store, err := sdk.New("./data")
    if err != nil {
        log.Fatal(err)
    }
}
```

### Basic CRUD
The SDK now supports generics for type-safe operations (Go 1.18+).

```go
// Set a value
err := sdk.Set[string](store, "persona1", "my-app", "theme", "dark")

// Get a value
val, err := sdk.Get[string](store, "persona1", "my-app", "theme")
```

---

## Advanced Features

### App Scopes
Scopes make your code cleaner by "remembering" the Persona and App IDs.

```go
app := store.App("persona1", "my-app")

app.Set("volume", 80)
vol, _ := app.Get("volume")
```

### Global Indexing & Lookups
If you need to find which persona owns a specific key within an app (e.g., finding a file by its unique ID across all users):

```go
// In the future: store.SetGlobal(...)
// For now, look up where a key exists globally for an app:
val, personaID, err := store.GetGlobal("my-app", "unique-file-id")
```

### Batch Operations
To retrieve data across all personas for a specific application (useful for admin dashboards):

```go
// Returns map[PersonaID]map[Key]Value
allAppData, err := store.DumpApp("my-app")
```

### Atomic Moves
Transfer data from one persona to another safely.

```go
err := store.Move("old-owner", "new-owner", "my-app", "document-123")
```

### The Vault (Client-Side Encryption)
Encrypt sensitive data before it ever leaves your application process.

```go
masterKey := []byte("a-very-secret-32-byte-long-key!!")
vault := app.Vault(masterKey)

// Encrypted in transit and at rest
vault.Set("api_key", "sk-123456")

// Decrypted locally
key, _ := vault.Get("api_key")
```

---

## Migration and Backups

You can easily move data between embedded and remote stores using the `Migrate` utility.

```go
localStore, _ := sdk.New("./data")
remoteStore, _ := sdk.Connect("localhost:7001")

// Upgrade local data to the shared server
err := sdk.Migrate(localStore, remoteStore)
```
