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

	// Background uploads started by requests that held the read lock
	uploadsMu sync.Mutex
	uploads   int
	idle      *sync.Cond
}

// ReadLock keeps namespace from being renamed or deleted until the returned
// func is called
func ReadLock(namespace string) func() {
	return lock(namespace, false)
}

// TrackUpload registers a background upload into namespace, started while
// holding its read lock. Renames and deletes wait for it after taking the
// write lock, so the upload cannot recreate the old namespace afterwards.
// Call the returned func when the upload is done.
func TrackUpload(namespace string) func() {
	l := acquire(namespace)
	l.uploadsMu.Lock()
	l.uploads++
	l.uploadsMu.Unlock()

	var once sync.Once
	return func() {
		once.Do(func() {
			l.uploadsMu.Lock()
			l.uploads--
			if l.uploads == 0 {
				l.idle.Broadcast()
			}
			l.uploadsMu.Unlock()
			release(namespace, l)
		})
	}
}

func lock(namespace string, write bool) func() {
	l := acquire(namespace)
	if write {
		l.Lock()
		l.uploadsMu.Lock()
		for l.uploads > 0 {
			l.idle.Wait()
		}
		l.uploadsMu.Unlock()
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
			release(namespace, l)
		})
	}
}

func acquire(namespace string) *namespaceLock {
	locksMu.Lock()
	defer locksMu.Unlock()
	l, ok := locks[namespace]
	if !ok {
		l = &namespaceLock{}
		l.idle = sync.NewCond(&l.uploadsMu)
		locks[namespace] = l
	}
	l.refs++
	return l
}

func release(namespace string, l *namespaceLock) {
	locksMu.Lock()
	defer locksMu.Unlock()
	l.refs--
	if l.refs == 0 {
		delete(locks, namespace)
	}
}
