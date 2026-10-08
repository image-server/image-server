package namespaces

import "sync"

// Requests that write into a namespace hold its read lock; renames and
// deletes take the write lock. They wait for in-flight work, so a variant
// being generated cannot recreate the old directory.
var (
	locksMu sync.Mutex
	locks   = map[string]*namespaceLock{}
)

type namespaceLock struct {
	sync.RWMutex
	refs int
}

// ReadLock keeps namespace from being renamed or deleted until the returned
// func is called
func ReadLock(namespace string) func() {
	return lock(namespace, false)
}

func lock(namespace string, write bool) func() {
	locksMu.Lock()
	l, ok := locks[namespace]
	if !ok {
		l = &namespaceLock{}
		locks[namespace] = l
	}
	l.refs++
	locksMu.Unlock()

	if write {
		l.Lock()
	} else {
		l.RLock()
	}

	var once sync.Once
	return func() {
		once.Do(func() {
			if write {
				l.Unlock()
			} else {
				l.RUnlock()
			}
			locksMu.Lock()
			l.refs--
			if l.refs == 0 {
				delete(locks, namespace)
			}
			locksMu.Unlock()
		})
	}
}
