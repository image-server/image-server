package server

import (
	"time"

	"github.com/image-server/image-server/namespaces"
)

// SetNamespaceRemote replaces the bucket the namespace admin routes use
func SetNamespaceRemote(r namespaces.Remote) (restore func()) {
	old := namespaceRemote
	namespaceRemote = r
	return func() { namespaceRemote = old }
}

// ResetRejectionLog starts a new rate limit window for rejection warnings,
// so a test that reads them is not affected by earlier tests
func ResetRejectionLog() {
	rejectionLog.Lock()
	defer rejectionLog.Unlock()
	rejectionLog.window, rejectionLog.logged, rejectionLog.suppressed = time.Time{}, 0, 0
}
