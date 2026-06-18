package server

import (
	"net/http"

	"github.com/Waffleophagus/tailor/internal/authz"
)

// BootstrapMiddleware attaches a short browser-only bootstrap authorization when
// the caller presents a valid bootstrap cookie tied to their tailnet identity.
func BootstrapMiddleware(server *Server, next http.Handler) http.Handler {
	if server == nil || server.bootstrap == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		identity, ok := authz.IdentityFromContext(ctx)
		if ok {
			token := bootstrapTokenFromRequest(r)
			if token != "" {
				if valid, _ := server.bootstrap.Valid(token, identity.LoginName, identity.NodeName); valid {
					ctx = authz.WithBootstrap(ctx)
				}
			}
		}
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
