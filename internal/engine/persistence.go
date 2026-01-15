// Package engine defines the core interfaces and types for the Celerix Store.
package engine

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
)

// Persistence handles the disk I/O for the MemStore
type Persistence struct {
	DataDir string
	mu      sync.Mutex // Protects concurrent writes to the filesystem
}

// NewPersistence initializes a persistence handler.
func NewPersistence(dir string) (*Persistence, error) {
	// Ensure the data directory exists
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	return &Persistence{DataDir: dir}, nil
}

// SavePersona writes a single persona's data to a JSON file atomically.
func (p *Persistence) SavePersona(personaID string, data map[string]map[string]any) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	filePath := filepath.Join(p.DataDir, fmt.Sprintf("%s.json", personaID))
	tempPath := filePath + ".tmp"

	// 1. Convert map to JSON bytes
	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	// 2. Write to a temporary file first
	if err := os.WriteFile(tempPath, bytes, 0644); err != nil {
		return err
	}

	// 3. Atomic Rename (The "Blink" swap)
	// On Linux/Unix, this replaces the file instantly.
	// If the power fails, you have either the old file or the new one, never a corrupt one.
	return os.Rename(tempPath, filePath)
}

// LoadAll returns all persona data found in the data directory.
func (p *Persistence) LoadAll() (map[string]map[string]map[string]any, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	allData := make(map[string]map[string]map[string]any)

	files, err := os.ReadDir(p.DataDir)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		if filepath.Ext(file.Name()) == ".json" {
			personaID := file.Name()[:len(file.Name())-5] // Strip .json

			content, err := os.ReadFile(filepath.Join(p.DataDir, file.Name()))
			if err != nil {
				log.Printf("Warning: Could not read persona file %s: %v", file.Name(), err)
				continue // Skip corrupted/unreadable files
			}

			var personaData map[string]map[string]any
			if err := json.Unmarshal(content, &personaData); err != nil {
				log.Printf("Warning: Could not unmarshal persona data from %s: %v", file.Name(), err)
				continue
			}
			allData[personaID] = personaData
		}
	}
	return allData, nil
}
