# Celerix Store Usage Guide

This guide provides detailed examples and best practices for using the `celerix-store` library in your Go applications.

## Core Concepts

### Data Hierarchy
- **Persona:** Represents a user or a system entity. Each persona's data is stored in its own JSON file.
- **App:** A namespace within a persona. This allows multiple applications to store data for the same persona without collisions.
- **Key:** The specific identifier for a piece of data within an app namespace.

### In-Memory Sync Architecture
The `celerix-store` operates on an "In-Memory First" principle:
1.  **RAM as Primary:** All data is held in an optimized `map` structure in memory.
2.  **Filesystem as Secondary:** Each **Persona** is mapped 1:1 to a `.json` file in the data directory.
3.  **Background Flush:** When you `Set` or `Delete` a key, the change is applied immediately to RAM. A background goroutine then takes a thread-safe snapshot and writes it to the corresponding persona file.
4.  **Startup Load:** When the engine starts (either as a daemon or embedded), it scans the data directory and hydrates the memory state from all discovered `.json` files.

This design ensures that Celerix applications enjoy ultra-low latency while maintaining a human-readable and portable disk footprint.

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
The SDK supports standard KV operations. It also includes generics for type-safe operations (Go 1.18+).

```go
// Standard Set (returns error)
err := store.Set("persona1", "my-app", "theme", "dark")

// Standard Get (returns any, error)
val, err := store.Get("persona1", "my-app", "theme")

// Standard Delete (returns error)
err := store.Delete("persona1", "my-app", "theme")

// Type-Safe Set (Generics)
err := sdk.Set[string](store, "persona1", "my-app", "theme", "dark")

// Type-Safe Get (Generics)
val, err := sdk.Get[string](store, "persona1", "my-app", "theme")
```

### Discovery and Enumeration
Methods to explore the store's structure.

```go
// List all personas
personas, _ := store.GetPersonas() // []string

// List all apps for a specific persona
apps, _ := store.GetApps("persona1") // []string

// Dump all keys/values for a specific app (single persona)
// Returns map[string]any
data, _ := store.GetAppStore("persona1", "my-app")
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
Encrypt sensitive data before it ever leaves your application process. `Vault` works only with string values and uses AES-GCM encryption.

```go
masterKey := []byte("a-very-secret-32-byte-long-key!!")
vault := app.Vault(masterKey)

// Encrypted in transit and at rest
err := vault.Set("api_key", "sk-123456")

// Decrypted locally
val, err := vault.Get("api_key")
```

### Deleting Data
You can delete data at the key level.

```go
// Direct delete
err := store.Delete("persona1", "my-app", "key1")

// Via App Scope
err := app.Delete("key1")
```

---

## Environment Variables
The store and SDK can be configured using environment variables.

### Client & SDK Variables
- `CELERIX_STORE_ADDR`: Address of the remote store (e.g., `localhost:7001`). If not set, the SDK defaults to **Embedded Mode**.
- `CELERIX_DISABLE_TLS`: Set to `true` to disable TLS for network communication.

### Daemon (Server) Variables
- `CELERIX_PORT`: The port the daemon will listen on (default: `7001`).
- `CELERIX_DATA_DIR`: The path to the directory where data files are stored (default: `./data`).
- `CELERIX_DISABLE_TLS`: Set to `true` to run the server over plain TCP.
