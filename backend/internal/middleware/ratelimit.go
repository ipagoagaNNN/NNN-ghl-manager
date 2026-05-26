package middleware

import (
	"net/http"
	"sync"
	"time"
)

// RateLimit enforces a simple per-IP sliding window on /api/ghl/* routes.
// GHL's own rate limits are handled inside the proxy with per-token back-off.
func RateLimit(next http.Handler) http.Handler {
	type window struct {
		count    int
		resetAt  time.Time
	}
	var mu sync.Mutex
	windows := make(map[string]*window)

	const maxReq = 60
	const period = time.Minute

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := r.RemoteAddr
		mu.Lock()
		win, ok := windows[ip]
		if !ok || time.Now().After(win.resetAt) {
			win = &window{count: 0, resetAt: time.Now().Add(period)}
			windows[ip] = win
		}
		win.count++
		over := win.count > maxReq
		mu.Unlock()

		if over {
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}
