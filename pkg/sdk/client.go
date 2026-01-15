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

type Client struct {
	conn   net.Conn
	reader *bufio.Reader
	mu     sync.Mutex // Protects concurrent access to the connection
}

func Connect(addr string) (*Client, error) {
	var conn net.Conn
	var err error

	dialer := &net.Dialer{Timeout: 10 * time.Second}

	if os.Getenv("CELERIX_DISABLE_TLS") == "true" {
		conn, err = dialer.Dial("tcp", addr)
	} else {
		config := &tls.Config{
			InsecureSkipVerify: true, // We use self-signed certs for internal traffic
		}
		conn, err = tls.DialWithDialer(dialer, "tcp", addr, config)
	}

	if err != nil {
		return nil, err
	}
	return &Client{
		conn:   conn,
		reader: bufio.NewReader(conn),
	}, nil
}

// Internal helper for TCP communication
func (c *Client) sendAndReceive(cmd string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Set deadlines for the operation
	c.conn.SetDeadline(time.Now().Add(30 * time.Second))

	_, err := fmt.Fprint(c.conn, cmd+"\n")
	if err != nil {
		return "", err
	}
	resp, err := c.reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	resp = strings.TrimSpace(resp)
	if strings.HasPrefix(resp, "ERR") {
		return "", fmt.Errorf("%s", strings.TrimPrefix(resp, "ERR "))
	}
	return resp, nil
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

func (c *Client) Close() error {
	fmt.Fprintln(c.conn, "QUIT")
	return c.conn.Close()
}

// App returns a helper that "remembers" the Persona and App IDs
func (c *Client) App(personaID, appID string) *AppScope {
	return &AppScope{
		client:    c,
		personaID: personaID,
		appID:     appID,
	}
}

type AppScope struct {
	client    *Client
	personaID string
	appID     string
}

func (a *AppScope) Set(key string, val any) error {
	return a.client.Set(a.personaID, a.appID, key, val)
}

func (a *AppScope) Get(key string) (any, error) {
	return a.client.Get(a.personaID, a.appID, key)
}

func (a *AppScope) Delete(key string) error {
	return a.client.Delete(a.personaID, a.appID, key)
}

// Vault returns a scope that automatically encrypts/decrypts data
func (a *AppScope) Vault(masterKey []byte) *VaultScope {
	return &VaultScope{
		app:       a,
		masterKey: masterKey,
	}
}

type VaultScope struct {
	app       *AppScope
	masterKey []byte
}

func (v *VaultScope) Set(key string, plaintext string) error {
	// 1. Encrypt locally before sending
	ciphertext, err := vault.Encrypt(plaintext, v.masterKey)
	if err != nil {
		return err
	}
	// 2. Save the encrypted hex string to the store
	return v.app.Set(key, ciphertext)
}

func (v *VaultScope) Get(key string) (string, error) {
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
