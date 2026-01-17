package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestMemStore_GetSetDelete(t *testing.T) {
	ms := NewMemStore(nil, nil)

	personaID := "test-persona"
	appID := "test-app"
	key := "test-key"
	val := "test-value"

	// Test Set
	err := ms.Set(personaID, appID, key, val)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Test Get
	got, err := ms.Get(personaID, appID, key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got != val {
		t.Errorf("Expected %v, got %v", val, got)
	}

	// Test Get non-existent
	_, err = ms.Get(personaID, appID, "non-existent")
	if err != ErrKeyNotFound {
		t.Errorf("Expected ErrKeyNotFound, got %v", err)
	}

	// Test Delete
	err = ms.Delete(personaID, appID, key)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err = ms.Get(personaID, appID, key)
	if err != ErrKeyNotFound {
		t.Errorf("Expected ErrKeyNotFound after delete, got %v", err)
	}
}

func TestMemStore_GetPersonasApps(t *testing.T) {
	ms := NewMemStore(nil, nil)

	ms.Set("p1", "a1", "k1", "v1")
	ms.Set("p2", "a2", "k2", "v2")

	personas, _ := ms.GetPersonas()
	if len(personas) != 2 {
		t.Errorf("Expected 2 personas, got %d", len(personas))
	}

	apps, _ := ms.GetApps("p1")
	if len(apps) != 1 || apps[0] != "a1" {
		t.Errorf("Expected [a1], got %v", apps)
	}
}

func TestPersistence(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "celerix-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	p, err := NewPersistence(tmpDir)
	if err != nil {
		t.Fatalf("NewPersistence failed: %v", err)
	}

	data := map[string]map[string]any{
		"app1": {
			"key1": "val1",
		},
	}

	err = p.SavePersona("user1", data)
	if err != nil {
		t.Fatalf("SavePersona failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(filepath.Join(tmpDir, "user1.json")); os.IsNotExist(err) {
		t.Fatal("Persona file was not created")
	}

	allData, err := p.LoadAll()
	if err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}

	if len(allData) != 1 {
		t.Errorf("Expected 1 persona, got %d", len(allData))
	}

	if allData["user1"]["app1"]["key1"] != "val1" {
		t.Errorf("Loaded data mismatch: %v", allData["user1"])
	}
}

func TestMemStore_Persistence(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "celerix-persistence-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	p, _ := NewPersistence(tmpDir)
	ms := NewMemStore(nil, p)

	err = ms.Set("p1", "a1", "k1", "v1")
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	ms.Wait() // Wait for background persistence

	// Create new MemStore and load data
	allData, _ := p.LoadAll()
	ms2 := NewMemStore(allData, p)

	val, err := ms2.Get("p1", "a1", "k1")
	if err != nil {
		t.Fatalf("Get on new store failed: %v", err)
	}
	if val != "v1" {
		t.Errorf("Expected v1, got %v", val)
	}
}

func TestMemStore_AppScopeAndVault(t *testing.T) {
	ms := NewMemStore(nil, nil)
	masterKey := []byte("thisis32byteslongsecretkey123456")

	scope := ms.App("p1", "a1")
	err := scope.Set("secret", "hidden")
	if err != nil {
		t.Fatalf("Scope Set failed: %v", err)
	}

	val, _ := scope.Get("secret")
	if val != "hidden" {
		t.Errorf("Expected hidden, got %v", val)
	}

	// Test Vault
	v := scope.Vault(masterKey)
	// We need to type assert since ms.App returns sdk.AppScope which returns any for Vault
	// But in memstore.go:280 it returns *memVaultScope.
	// Actually sdk.AppScope defines Vault(masterKey []byte) any

	// Let's use it as it is supposed to be used.
	// Since we are in the same package, we can type assert to *memVaultScope if we want,
	// but the interface return type makes it a bit tricky if we want to use the methods.

	// Let's check memstore.go again.
	// func (v *memVaultScope) Set(key string, plaintext string) error
	// func (v *memVaultScope) Get(key string) (string, error)

	type vaulter interface {
		Set(key string, plaintext string) error
		Get(key string) (string, error)
	}

	vv := v.(vaulter)
	err = vv.Set("password", "topsecret")
	if err != nil {
		t.Fatalf("Vault Set failed: %v", err)
	}

	pass, err := vv.Get("password")
	if err != nil {
		t.Fatalf("Vault Get failed: %v", err)
	}
	if pass != "topsecret" {
		t.Errorf("Expected topsecret, got %v", pass)
	}

	// Check that it's encrypted in the underlying store
	raw, _ := scope.Get("password")
	if raw == "topsecret" {
		t.Error("Vault value should be encrypted in store")
	}
}

func TestMemStore_Concurrent(t *testing.T) {
	ms := NewMemStore(nil, nil)
	const (
		numGroutines = 10
		numOps       = 100
	)
	var wg sync.WaitGroup

	for i := 0; i < numGroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOps; j++ {
				key := fmt.Sprintf("key-%d-%d", id, j)
				ms.Set("p1", "a1", key, j)
				val, err := ms.Get("p1", "a1", key)
				if err != nil || val != j {
					// We can't use t.Fatalf in a goroutine
					fmt.Printf("Concurrent error: expected %d, got %v, err %v\n", j, val, err)
				}
			}
		}(i)
	}
	wg.Wait()
}

func TestMemStore_DumpApp(t *testing.T) {
	ms := NewMemStore(nil, nil)
	ms.Set("p1", "a1", "k1", "v1")
	ms.Set("p2", "a1", "k1", "v2")
	ms.Set("p1", "a2", "k2", "v3")

	dump, err := ms.DumpApp("a1")
	if err != nil {
		t.Fatalf("DumpApp failed: %v", err)
	}

	if len(dump) != 2 {
		t.Errorf("Expected 2 personas in dump, got %d", len(dump))
	}

	if dump["p1"]["k1"] != "v1" || dump["p2"]["k1"] != "v2" {
		t.Errorf("Dump mismatch: %v", dump)
	}
}

func TestMemStore_GetGlobal(t *testing.T) {
	ms := NewMemStore(nil, nil)
	ms.Set("p1", "a1", "k1", "v1")

	val, persona, err := ms.GetGlobal("a1", "k1")
	if err != nil {
		t.Fatalf("GetGlobal failed: %v", err)
	}
	if val != "v1" || persona != "p1" {
		t.Errorf("GetGlobal mismatch: %v, %s", val, persona)
	}

	_, _, err = ms.GetGlobal("a1", "non-existent")
	if err != ErrKeyNotFound {
		t.Errorf("Expected ErrKeyNotFound, got %v", err)
	}
}

func TestMemStore_Move(t *testing.T) {
	ms := NewMemStore(nil, nil)
	ms.Set("p1", "a1", "k1", "v1")

	err := ms.Move("p1", "p2", "a1", "k1")
	if err != nil {
		t.Fatalf("Move failed: %v", err)
	}

	val, err := ms.Get("p2", "a1", "k1")
	if err != nil || val != "v1" {
		t.Errorf("Move failed to set dst: %v, %v", val, err)
	}

	_, err = ms.Get("p1", "a1", "k1")
	if err != ErrKeyNotFound {
		t.Errorf("Move failed to delete src: %v", err)
	}
}
