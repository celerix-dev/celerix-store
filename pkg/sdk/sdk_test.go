package sdk_test

import (
	"fmt"
	"net"
	"os"
	"testing"

	"github.com/celerix-dev/celerix-store/internal/server"
	"github.com/celerix-dev/celerix-store/pkg/engine"
	"github.com/celerix-dev/celerix-store/pkg/sdk"
)

// MockStore implements CelerixStore for testing SDK helpers
type MockStore struct {
	data map[string]any
}

func (m *MockStore) Get(personaID, appID, key string) (any, error) {
	return m.data[key], nil
}
func (m *MockStore) Set(personaID, appID, key string, val any) error {
	m.data[key] = val
	return nil
}
func (m *MockStore) Delete(personaID, appID, key string) error  { return nil }
func (m *MockStore) GetPersonas() ([]string, error)             { return nil, nil }
func (m *MockStore) GetApps(personaID string) ([]string, error) { return nil, nil }
func (m *MockStore) GetAppStore(personaID, appID string) (map[string]any, error) {
	return nil, nil
}
func (m *MockStore) DumpApp(appID string) (map[string]map[string]any, error) { return nil, nil }
func (m *MockStore) GetGlobal(appID, key string) (any, string, error)        { return nil, "", nil }
func (m *MockStore) Move(srcPersona, dstPersona, appID, key string) error    { return nil }
func (m *MockStore) App(personaID, appID string) sdk.AppScope                { return nil }

func TestGenericGetSet(t *testing.T) {
	ms := &MockStore{data: make(map[string]any)}

	type User struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	user := User{Name: "Alice", Age: 30}

	// Test Generic Set
	err := sdk.Set(ms, "p1", "a1", "user1", user)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Test Generic Get
	gotUser, err := sdk.Get[User](ms, "p1", "a1", "user1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if gotUser.Name != user.Name || gotUser.Age != user.Age {
		t.Errorf("Expected %v, got %v", user, gotUser)
	}
}

func TestGenericGetWithJsonConversion(t *testing.T) {
	// Simulate data coming from JSON (where it's map[string]any)
	ms := &MockStore{data: map[string]any{
		"user1": map[string]any{
			"name": "Bob",
			"age":  float64(25), // JSON unmarshals numbers as float64
		},
	}}

	type User struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	gotUser, err := sdk.Get[User](ms, "p1", "a1", "user1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if gotUser.Name != "Bob" || gotUser.Age != 25 {
		t.Errorf("Expected Bob/25, got %v", gotUser)
	}
}

func TestClient_Integration(t *testing.T) {
	// Start a real server on a random port
	store := engine.NewMemStore(nil, nil)
	router := server.NewRouter(store)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to listen: %v", err)
	}
	port := fmt.Sprintf("%d", listener.Addr().(*net.TCPAddr).Port)
	addr := "127.0.0.1:" + port

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go router.HandleConnection(conn)
		}
	}()
	defer listener.Close()

	os.Setenv("CELERIX_DISABLE_TLS", "true")
	defer os.Unsetenv("CELERIX_DISABLE_TLS")

	client, err := sdk.Connect(addr)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	// Test basic operations
	err = client.Set("p1", "a1", "k1", "v1")
	if err != nil {
		t.Fatalf("Client Set failed: %v", err)
	}

	val, err := client.Get("p1", "a1", "k1")
	if err != nil || val != "v1" {
		t.Errorf("Client Get failed: %v, %v", val, err)
	}

	// Test App Scope
	app := client.App("p1", "a1")
	err = app.Set("k2", "v2")
	if err != nil {
		t.Fatalf("App Set failed: %v", err)
	}

	val, _ = app.Get("k2")
	if val != "v2" {
		t.Errorf("App Get failed: %v", val)
	}

	// Test Vault Scope
	masterKey := []byte("thisis32byteslongsecretkey123456")
	vault := app.Vault(masterKey).(interface {
		Set(key string, plaintext string) error
		Get(key string) (string, error)
	})

	err = vault.Set("secret", "mypassword")
	if err != nil {
		t.Fatalf("Vault Set failed: %v", err)
	}

	got, err := vault.Get("secret")
	if err != nil || got != "mypassword" {
		t.Errorf("Vault Get failed: %v, %v", got, err)
	}
}

func TestClient_RetryLogic(t *testing.T) {
	// This test is harder because it depends on the server dying and coming back,
	// or the connection being dropped.
	// But we can at least verify that it tries to reconnect if we close the server.

	store := engine.NewMemStore(nil, nil)
	router := server.NewRouter(store)

	listener, _ := net.Listen("tcp", "127.0.0.1:0")
	addr := listener.Addr().String()

	go func() {
		conn, _ := listener.Accept()
		if conn != nil {
			go router.HandleConnection(conn)
		}
	}()

	os.Setenv("CELERIX_DISABLE_TLS", "true")
	client, _ := sdk.Connect(addr)

	// Close the listener so NO MORE connections can be accepted
	listener.Close()

	// The existing connection might still work for one command if it was already accepted.
	client.Set("p1", "a1", "k1", "v1")

	// Now if we try again, it might fail and try to reconnect, which will fail.
	// We just want to see it doesn't panic.
	client.Get("p1", "a1", "k1")
}
