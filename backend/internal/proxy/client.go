package proxy

import (
	"fmt"
	"io"
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

	var bodyBytes []byte
	if r.Body != nil {
		bodyBytes, _ = io.ReadAll(r.Body)
	}

	var resp *http.Response
	var err error
	for attempt := 0; attempt < maxRetries; attempt++ {
		req, _ := http.NewRequestWithContext(r.Context(), r.Method, targetURL, strings.NewReader(string(bodyBytes)))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Version", ghlVersion)
		if len(bodyBytes) > 0 {
			req.Header.Set("Content-Type", "application/json")
		}

		resp, err = h.client.Do(req)
		if err != nil {
			http.Error(w, fmt.Sprintf("proxy error: %v", err), http.StatusBadGateway)
			return
		}

		// Retry on 429 with back-off
		if resp.StatusCode == http.StatusTooManyRequests && attempt < maxRetries-1 {
			resp.Body.Close()
			time.Sleep(time.Duration(500*(attempt+1)) * time.Millisecond)
			continue
		}
		break
	}
	defer resp.Body.Close()

	for k, vv := range resp.Header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
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
