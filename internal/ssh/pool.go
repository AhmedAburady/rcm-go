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
//
// The lock is released while dialing (NewClient does blocking network I/O and,
// in agent mode, may wait on a 1Password biometric prompt), so concurrent
// connects to *other* hosts aren't serialized behind it. A second lock-and-check
// after dialing handles the rare race where two goroutines open the same
// host+auth at once — the loser closes its duplicate and reuses the winner's.
func GetClient(host, user string, auth AuthConfig) (*Client, error) {
	key := poolKey(host, user, auth)

	poolMu.Lock()
	if client, ok := pool[key]; ok {
		poolMu.Unlock()
		return client, nil
	}
	poolMu.Unlock()

	client, err := NewClient(host, user, auth)
	if err != nil {
		return nil, err
	}

	poolMu.Lock()
	defer poolMu.Unlock()
	if existing, ok := pool[key]; ok {
		client.Close() // lost the race; discard our duplicate
		return existing, nil
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
