// Package sdk provides the client-side library for interacting with the Celerix Store.
// It supports both remote connections via TCP/TLS and local embedded mode.
package sdk

import (
	"bufio"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/celerix-dev/celerix-store/internal/vault"
)

// Client is a remote client for the Celerix Store.
// It implements the CelerixStore interface.
type Client struct {
	addr   string
	conn   net.Conn
	reader *bufio.Reader
	mu     sync.Mutex // Protects concurrent access to the connection
}

// Connect establishes a TLS-encrypted connection to a remote Celerix Store daemon.
// If CELERIX_DISABLE_TLS is set to "true", it falls back to plain TCP.
func Connect(addr string) (*Client, error) {
	c := &Client{addr: addr}
	if err := c.reconnect(); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Client) reconnect() error {
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}

	var conn net.Conn
	var err error

	dialer := &net.Dialer{
		Timeout:   10 * time.Second,
		KeepAlive: 60 * time.Second, // Increased keep-alive
	}

	if os.Getenv("CELERIX_DISABLE_TLS") == "true" {
		conn, err = dialer.Dial("tcp", c.addr)
	} else {
		config := &tls.Config{
			InsecureSkipVerify: true, // We use self-signed certs for internal traffic
		}
		conn, err = tls.DialWithDialer(dialer, "tcp", c.addr, config)
	}

	if err != nil {
		return err
	}

	c.conn = conn
	c.reader = bufio.NewReader(conn)
	return nil
}

// Internal helper for TCP communication
func (c *Client) sendAndReceive(cmd string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var err error
	var resp string

	// Try up to 3 times with exponential backoff
	for i := 0; i < 3; i++ {
		// Ensure we have a connection
		if c.conn == nil {
			if reconnectErr := c.reconnect(); reconnectErr != nil {
				err = fmt.Errorf("reconnect failed: %w", reconnectErr)
				time.Sleep(time.Duration(i*100) * time.Millisecond)
				continue
			}
		}

		// Set deadlines for the operation
		c.conn.SetDeadline(time.Now().Add(30 * time.Second))

		_, err = fmt.Fprint(c.conn, cmd+"\n")
		if err == nil {
			resp, err = c.reader.ReadString('\n')
			if err == nil {
				resp = strings.TrimSpace(resp)
				if strings.HasPrefix(resp, "ERR") {
					return "", fmt.Errorf("%s", strings.TrimPrefix(resp, "ERR "))
				}
				return resp, nil
			}
		}

		// If we got here, there was an error communicating.
		fmt.Fprintf(os.Stderr, "[Celerix SDK] Attempt %d failed: %v. Reconnecting...\n", i+1, err)

		// Force a reconnect on the next iteration
		if closeErr := c.reconnect(); closeErr != nil {
			fmt.Fprintf(os.Stderr, "[Celerix SDK] Reconnect attempt failed: %v\n", closeErr)
		}

		// Wait before retrying (exponential backoff)
		time.Sleep(time.Duration((i+1)*200) * time.Millisecond)
	}

	return "", fmt.Errorf("failed after 3 attempts. last error: %v", err)
}

func (c *Client) Get(personaID, appID, key string) (any, error) {
	resp, err := c.sendAndReceive(fmt.Sprintf("GET %s %s %s", personaID, appID, key))
	if err != nil {
		return nil, err
	}
	jsonData := strings.TrimPrefix(resp, "OK ")
	var val any
	err = json.Unmarshal([]byte(jsonData), &val)
	return val, err
}

func (c *Client) Set(personaID, appID, key string, val any) error {
	jsonData, _ := json.Marshal(val)
	_, err := c.sendAndReceive(fmt.Sprintf("SET %s %s %s %s", personaID, appID, key, string(jsonData)))
	return err
}

func (c *Client) Delete(personaID, appID, key string) error {
	_, err := c.sendAndReceive(fmt.Sprintf("DEL %s %s %s", personaID, appID, key))
	return err
}

func (c *Client) GetPersonas() ([]string, error) {
	resp, err := c.sendAndReceive("LIST_PERSONAS")
	if err != nil {
		return nil, err
	}
	jsonData := strings.TrimPrefix(resp, "OK ")
	var list []string
	err = json.Unmarshal([]byte(jsonData), &list)
	return list, err
}

func (c *Client) GetApps(personaID string) ([]string, error) {
	resp, err := c.sendAndReceive(fmt.Sprintf("LIST_APPS %s", personaID))
	if err != nil {
		return nil, err
	}
	jsonData := strings.TrimPrefix(resp, "OK ")
	var list []string
	err = json.Unmarshal([]byte(jsonData), &list)
	return list, err
}

func (c *Client) GetAppStore(personaID, appID string) (map[string]any, error) {
	resp, err := c.sendAndReceive(fmt.Sprintf("DUMP %s %s", personaID, appID))
	if err != nil {
		return nil, err
	}
	jsonData := strings.TrimPrefix(resp, "OK ")
	var store map[string]any
	err = json.Unmarshal([]byte(jsonData), &store)
	return store, err
}

func (c *Client) DumpApp(appID string) (map[string]map[string]any, error) {
	resp, err := c.sendAndReceive(fmt.Sprintf("DUMP_APP %s", appID))
	if err != nil {
		return nil, err
	}
	jsonData := strings.TrimPrefix(resp, "OK ")
	var store map[string]map[string]any
	err = json.Unmarshal([]byte(jsonData), &store)
	return store, err
}

func (c *Client) GetGlobal(appID, key string) (any, string, error) {
	resp, err := c.sendAndReceive(fmt.Sprintf("GET_GLOBAL %s %s", appID, key))
	if err != nil {
		return nil, "", err
	}
	jsonData := strings.TrimPrefix(resp, "OK ")
	var out struct {
		Persona string `json:"persona"`
		Value   any    `json:"value"`
	}
	err = json.Unmarshal([]byte(jsonData), &out)
	return out.Value, out.Persona, err
}

func (c *Client) Move(srcPersona, dstPersona, appID, key string) error {
	_, err := c.sendAndReceive(fmt.Sprintf("MOVE %s %s %s %s", srcPersona, dstPersona, appID, key))
	return err
}

func (c *Client) Close() error {
	fmt.Fprintln(c.conn, "QUIT")
	return c.conn.Close()
}

// --- Generics Support (Go 1.18+) ---

// Get retrieves a type-safe value using Go generics.
// It handles JSON unmarshaling into the target type automatically.
func Get[T any](s KVReader, personaID, appID, key string) (T, error) {
	var target T
	val, err := s.Get(personaID, appID, key)
	if err != nil {
		return target, err
	}

	// If it's already the right type (e.g. from MemStore), just return it
	if v, ok := val.(T); ok {
		return v, nil
	}

	// Otherwise, it might be a map/slice from JSON, so we re-marshal/unmarshal
	// This is a bit slow but ensures type safety for the caller.
	bytes, err := json.Marshal(val)
	if err != nil {
		return target, err
	}
	err = json.Unmarshal(bytes, &target)
	return target, err
}

// Set stores a type-safe value using Go generics.
func Set[T any](s KVWriter, personaID, appID, key string, val T) error {
	return s.Set(personaID, appID, key, val)
}

// --- App and Vault Scopes ---
// App returns a scoped interface for a specific persona and application.
func (c *Client) App(personaID, appID string) AppScope {
	return &RemoteAppScope{
		client:    c,
		personaID: personaID,
		appID:     appID,
	}
}

// RemoteAppScope is a scoped client that "remembers" its persona and application IDs.
type RemoteAppScope struct {
	client    *Client
	personaID string
	appID     string
}

// Set stores a value using the scoped persona and app.
func (a *RemoteAppScope) Set(key string, val any) error {
	return a.client.Set(a.personaID, a.appID, key, val)
}

// Get retrieves a value using the scoped persona and app.
func (a *RemoteAppScope) Get(key string) (any, error) {
	return a.client.Get(a.personaID, a.appID, key)
}

// Delete removes a key using the scoped persona and app.
func (a *RemoteAppScope) Delete(key string) error {
	return a.client.Delete(a.personaID, a.appID, key)
}

// Vault returns a scope that automatically encrypts/decrypts data.
// It returns any to satisfy the AppScope interface.
func (a *RemoteAppScope) Vault(masterKey []byte) any {
	return &RemoteVaultScope{
		app:       a,
		masterKey: masterKey,
	}
}

// RemoteVaultScope provides client-side encryption for sensitive data.
type RemoteVaultScope struct {
	app       *RemoteAppScope
	masterKey []byte
}

// Set encrypts the plaintext and stores it in the scoped app.
func (v *RemoteVaultScope) Set(key string, plaintext string) error {
	// 1. Encrypt locally before sending
	ciphertext, err := vault.Encrypt(plaintext, v.masterKey)
	if err != nil {
		return err
	}
	// 2. Save the encrypted hex string to the store
	return v.app.Set(key, ciphertext)
}

// Get retrieves and decrypts a value from the scoped app.
func (v *RemoteVaultScope) Get(key string) (string, error) {
	// 1. Get the encrypted hex string from the store
	val, err := v.app.Get(key)
	if err != nil {
		return "", err
	}

	ciphertext, ok := val.(string)
	if !ok {
		return "", fmt.Errorf("vault data is not a string")
	}

	// 2. Decrypt locally
	return vault.Decrypt(ciphertext, v.masterKey)
}
