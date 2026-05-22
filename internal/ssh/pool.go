package ssh

import (
	"sync"
)

// pool manages SSH connections - one per host, reused for all operations
var (
	pool   = make(map[string]*Client)
	poolMu sync.Mutex
)

// poolKey identifies a pooled connection. It includes the auth identity so two
// callers with the same user@host but different credentials (e.g. agent vs. a
// key file) don't share a connection and silently authenticate as each other.
func poolKey(host, user string, auth AuthConfig) string {
	return user + "@" + host + "#" + auth.cacheKey()
}

// GetClient returns a cached connection or creates a new one.
// This mimics the Python Fabric pattern - one connection per host+auth, reused.
func GetClient(host, user string, auth AuthConfig) (*Client, error) {
	poolMu.Lock()
	defer poolMu.Unlock()

	key := poolKey(host, user, auth)

	// Return cached connection if it exists
	if client, ok := pool[key]; ok {
		return client, nil
	}

	// Create new connection
	client, err := NewClient(host, user, auth)
	if err != nil {
		return nil, err
	}

	pool[key] = client
	return client, nil
}

// RemoveClient removes a client from the pool (call when connection fails)
func RemoveClient(host, user string, auth AuthConfig) {
	poolMu.Lock()
	defer poolMu.Unlock()

	key := poolKey(host, user, auth)
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
