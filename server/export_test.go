package server

import "github.com/image-server/image-server/namespaces"

// SetNamespaceRemote replaces the bucket the namespace admin routes use
func SetNamespaceRemote(r namespaces.Remote) (restore func()) {
	old := namespaceRemote
	namespaceRemote = r
	return func() { namespaceRemote = old }
}
