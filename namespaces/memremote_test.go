package namespaces

import (
	"errors"
	"sort"
	"strings"
	"sync"
)

// memRemote is an in-memory bucket. failCopyAfter and failDelete simulate a
// store that fails part way.
type memRemote struct {
	mu            sync.Mutex
	objects       map[string]string
	copies        int
	failCopyAfter int // fail every copy once this many succeeded; 0 = never
	failDelete    bool
	deleteBatches [][]string
}

var errStore = errors.New("store unavailable")

func newMemRemote(keys ...string) *memRemote {
	m := &memRemote{objects: map[string]string{}}
	for _, k := range keys {
		m.objects[k] = "data:" + k
	}
	return m
}

func (m *memRemote) ListKeys(prefix string) ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var keys []string
	for k := range m.objects {
		if strings.HasPrefix(k, prefix) {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	return keys, nil
}

func (m *memRemote) Copy(src, dst string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.failCopyAfter > 0 && m.copies >= m.failCopyAfter {
		return errStore
	}
	data, ok := m.objects[src]
	if !ok {
		return errors.New("no such key " + src)
	}
	m.objects[dst] = data
	m.copies++
	return nil
}

func (m *memRemote) DeleteKeys(keys []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.failDelete {
		return errStore
	}
	m.deleteBatches = append(m.deleteBatches, keys)
	for _, k := range keys {
		delete(m.objects, k)
	}
	return nil
}

func (m *memRemote) PutBytes(key string, data []byte, contentType string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.objects[key] = string(data)
	return nil
}

func (m *memRemote) GetBytes(key string) ([]byte, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	data, ok := m.objects[key]
	return []byte(data), ok, nil
}

func (m *memRemote) keys() []string {
	keys, _ := m.ListKeys("")
	return keys
}
