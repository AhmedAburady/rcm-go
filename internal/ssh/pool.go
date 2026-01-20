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

	// Return cached connection if it exists and is alive
	if client, ok := pool[key]; ok {
		// Test if connection is still alive
		if _, err := client.Run("true"); err == nil {
			return client, nil
		}
		// Connection dead, clean it up
		client.Close()
		delete(pool, key)
	}

	// Create new connection
	client, err := NewClient(host, user, keyPath)
	if err != nil {
		return nil, err
	}

	pool[key] = client
	return client, nil
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
