package middleware

import "net/http"

// Auth is a no-op middleware stub.
// Phase 2: replace with JWT validation — check httpOnly cookie, validate signature,
// set user context. All route handlers already receive the request through this
// middleware, so Phase 2 requires no handler changes.
func Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// TODO Phase 2: validate JWT cookie, reject with 401 if invalid
		next.ServeHTTP(w, r)
	})
}

// Chain composes middleware in order: first applied is outermost.
func Chain(h http.Handler, middlewares ...func(http.Handler) http.Handler) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		h = middlewares[i](h)
	}
	return h
}
