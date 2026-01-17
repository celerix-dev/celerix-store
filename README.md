# celerix-store

[![Go Reference](https://pkg.go.dev/badge/github.com/celerix-dev/celerix-store.svg)](https://pkg.go.dev/github.com/celerix-dev/celerix-store)

`celerix-store` is a lightweight, low-latency Key-Value (KV) data store designed for the Celerix suite of applications. It provides a **"Liquid Data"** experience, allowing applications to seamlessly transition between local-first embedded storage and a networked, shared service without changing application logic.

### What it does
- **Unified State Management:** Stores configuration and state at the **Persona -> App -> Key** level.
- **Dual-Mode Operation:** Works as an embedded Go library (zero-dependency) or a standalone TLS-encrypted daemon.
- **Transparent Discovery:** The SDK automatically detects whether to run locally or connect to a remote server.
- **End-to-End Security:** Includes a "Vault" layer for client-side AES-GCM encryption and automatic TLS for network traffic.
- **Thread-Safe & Crash-Resilient:** Uses deep-copy persistence and atomic file renames to ensure data integrity.
- **Advanced Querying:** Supports cross-persona moves, global indexing, and batch app dumps.

### Architecture: In-Memory with File-System Sync
`celerix-store` is an **in-memory first** data store. 
- **Performance:** All read operations are served directly from RAM, providing microsecond latency.
- **Persistence:** Every write operation is synchronously applied to the in-memory state and asynchronously flushed to a 1:1 JSON file on disk.
- **Reliability:** Uses atomic "write-then-rename" operations for the filesystem to prevent data corruption during crashes or power failures.

## Documentation
- **[Usage Guide](USAGE.md):** Detailed guide on library usage, patterns, and best practices.

## Installation

```bash
go get github.com/celerix-dev/celerix-store
```

## Data Hierarchy
Data is organized in a three-tier hierarchy:
1. **Persona:** The top-level owner (e.g., a user or system identity). Data is persisted in `persona.json` files.
2. **App:** A namespace for a specific application or service.
3. **Key:** The specific configuration or state key.

## Usage Modes

### 1. Embedded Mode (Library)
Best for local-first applications or when running without infrastructure. Data is stored in local JSON files.

```go
import "github.com/celerix-dev/celerix-store/pkg/sdk"

func main() {
    // If CELERIX_STORE_ADDR is NOT set, it defaults to Embedded mode.
    store, _ := sdk.New("./data") 

    // Basic usage
    store.Set("persona1", "my-app", "theme", "dark")
    val, _ := store.Get("persona1", "my-app", "theme")
}
```

### 2. Remote Mode (Shared Service)
Multiple services can share state via the `celerix-stored` daemon.

#### Deploy with Docker Compose
```yaml
services:
  celerix-store:
    image: ghcr.io/celerix-dev/celerix-store:latest
    ports: ["7001:7001"]
    volumes: ["./data:/app/data"]

  my-service:
    image: my-service-image
    environment:
      - CELERIX_STORE_ADDR=celerix-store:7001
```

In your code, `sdk.New("./data")` will automatically detect the address and connect via TLS.

## SDK Advanced Features

### App Scopes
Instead of passing IDs every time, use a scope:
```go
app := store.App("persona1", "my-app")
app.Set("volume", 80)
```

### The Vault (Encrypted Storage)
For sensitive data (API keys, tokens), use the `Vault` scope which performs **client-side encryption**:
```go
masterKey := []byte("a-very-secret-32-byte-long-key!!") // Must be 32 bytes
vault := app.Vault(masterKey)

// Data is encrypted BEFORE being sent to the store
vault.Set("api_token", "super-secret-value")

// Data is decrypted locally
token, _ := vault.Get("api_token")
```

## Developer Reference

### Core Interfaces & Types (`pkg/sdk`)
Celerix Store follows the **Interface Segregation Principle**. Use the smallest interface your application needs:

- **`KVReader`**: Basic `Get` operations.
- **`KVWriter`**: `Set` and `Delete` operations.
- **`AppEnumeration`**: Discovering personas and apps.
- **`BatchExporter`**: Bulk data retrieval (`DumpApp`, `GetAppStore`).
- **`GlobalSearcher`**: Finding keys across all personas (`GetGlobal`).
- **`Orchestrator`**: High-level operations (`Move`).
- **`CelerixStore`**: The full composite interface.

### The `CelerixStore` Interface
Any storage implementation (Embedded or Client) satisfies this:
```go
type CelerixStore interface {
    KVReader
    KVWriter
    AppEnumeration
    BatchExporter
    GlobalSearcher
    Orchestrator

    // Scoping
    App(personaID, appID string) AppScope
}
```

### Shared Schemas (`pkg/schema`)
Common data structures used across the Celerix ecosystem (e.g., `UserRecord`, `AuditLog`) are available in `pkg/schema` to ensure data consistency between different services.

## CLI & Tooling

### Celerix CLI
```bash
go run cmd/celerix/main.go LIST_PERSONAS
go run cmd/celerix/main.go SET mypersona myapp mykey '{"foo": "bar"}'
```

### Standard Tools
TLS is enabled by default. Use `openssl` for raw testing:
```bash
echo "LIST_PERSONAS" | openssl s_client -connect localhost:7001 -quiet
```

## Environment Variables
- `CELERIX_STORE_ADDR`: Remote daemon address (e.g., `localhost:7001`). Used by the SDK and CLI.
- `CELERIX_PORT`: Port the daemon listens on (default: `7001`).
- `CELERIX_DATA_DIR`: Directory where JSON files are stored (default: `./data`).
- `CELERIX_DISABLE_TLS`: Set to `true` to revert to plain TCP.

## License
This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
