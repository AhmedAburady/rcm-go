package ssh

import (
	"sync"
)

// pool manages SSH connections - one per host, reused for all operations
var (
	pool   = make(map[string]*Client)
	poolMu sync.Mutex
)

// GetClient returns a cached connection or creates a new one.
// This mimics the Python Fabric pattern - one connection per host, reused.
func GetClient(host, user, keyPath string) (*Client, error) {
	poolMu.Lock()
	defer poolMu.Unlock()

	key := user + "@" + host

	// Return cached connection if it exists
	if client, ok := pool[key]; ok {
		return client, nil
	}

	// Create new connection
	client, err := NewClient(host, user, keyPath)
	if err != nil {
		return nil, err
	}

	pool[key] = client
	return client, nil
}

// RemoveClient removes a client from the pool (call when connection fails)
func RemoveClient(host, user string) {
	poolMu.Lock()
	defer poolMu.Unlock()

	key := user + "@" + host
	if client, ok := pool[key]; ok {
		client.Close()
		delete(pool, key)
	}
}

// CloseAll closes all cached connections. Call this when the app exits.
func CloseAll() {
	poolMu.Lock()
	defer poolMu.Unlock()

	for key, client := range pool {
		client.Close()
		delete(pool, key)
	}
}
