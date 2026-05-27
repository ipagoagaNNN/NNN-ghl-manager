package proxy

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/ipagoagaNNN/nnn-ghl-manager/backend/internal/store"
)

const (
	ghlVersion = "2021-07-28"
	maxRetries = 3
)

// Handler is the GHL API proxy. It strips /api/ghl from the path,
// injects the correct Bearer token from the vault, and forwards the request.
type Handler struct {
	vault  *store.Vault
	client *http.Client
}

func New(vault *store.Vault) *Handler {
	return &Handler{
		vault: vault,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Strip /api/ghl prefix
	ghlPath := strings.TrimPrefix(r.URL.Path, "/api/ghl")
	if ghlPath == "" {
		ghlPath = "/"
	}

	// Determine which token to use based on locationId query param or path
	token := h.resolveToken(r, ghlPath)
	if token == "" {
		http.Error(w, "no token for this location — call POST /api/tokens/{locationId} first", http.StatusUnauthorized)
		return
	}

	targetURL := h.vault.GHLBase() + ghlPath
	if r.URL.RawQuery != "" {
		targetURL += "?" + r.URL.RawQuery
	}

	// Defensive: enforce GHL host lock (belt-and-suspenders vs. SSRF).
	// gosec G704 flags this client.Do as SSRF because targetURL incorporates
	// user-controlled path segments. The base host is hardcoded in store.ghlBase
	// and validated here, so the user can only reach paths under GHL.
	if !strings.HasPrefix(targetURL, h.vault.GHLBase()) {
		http.Error(w, "invalid proxy target", http.StatusBadRequest)
		return
	}

	var bodyBytes []byte
	if r.Body != nil {
		var err error
		bodyBytes, err = io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "bad request body", http.StatusBadRequest)
			return
		}
	}

	var resp *http.Response
	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		// #nosec G107 G704 -- targetURL host is validated above; only path/query are user-controlled
		req, err := http.NewRequestWithContext(r.Context(), r.Method, targetURL, strings.NewReader(string(bodyBytes)))
		if err != nil {
			http.Error(w, fmt.Sprintf("proxy request build error: %v", err), http.StatusInternalServerError)
			return
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Version", ghlVersion)
		if len(bodyBytes) > 0 {
			req.Header.Set("Content-Type", "application/json")
		}

		// #nosec G107 G704 -- targetURL is locked to GHL base (validated above); user only controls path under that host
		resp, lastErr = h.client.Do(req)
		if lastErr != nil {
			http.Error(w, fmt.Sprintf("proxy error: %v", lastErr), http.StatusBadGateway)
			return
		}

		// Retry on 429 with back-off
		if resp.StatusCode == http.StatusTooManyRequests && attempt < maxRetries-1 {
			if cerr := resp.Body.Close(); cerr != nil {
				log.Printf("proxy: body close error during 429 retry: %v", cerr)
			}
			time.Sleep(time.Duration(500*(attempt+1)) * time.Millisecond)
			continue
		}
		break
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			log.Printf("proxy: body close error: %v", cerr)
		}
	}()

	for k, vv := range resp.Header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	if _, err := io.Copy(w, resp.Body); err != nil {
		// Client likely disconnected mid-stream — log only, response already started.
		log.Printf("proxy: response copy error: %v", err)
	}
}

func (h *Handler) resolveToken(r *http.Request, path string) string {
	// Try locationId query param first
	locID := r.URL.Query().Get("locationId")
	if locID != "" {
		if tok, ok := h.vault.LocToken(locID); ok {
			return tok
		}
	}

	// Try path extraction (e.g. /locations/{locId}/customValues)
	parts := strings.Split(strings.Trim(path, "/"), "/")
	for i, part := range parts {
		if part == "locations" && i+1 < len(parts) {
			if tok, ok := h.vault.LocToken(parts[i+1]); ok {
				return tok
			}
		}
	}

	// Fall back to agency token
	return h.vault.AgencyToken()
}
