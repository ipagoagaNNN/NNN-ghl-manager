package handlers

// Stub handlers — each will be implemented as its module is ported.
// All follow the same pattern: validate, fetch from GHL via vault token, return JSON.

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/ipagoagaNNN/nnn-ghl-manager/backend/internal/store"
)

func stub(name string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]string{"status": "not_yet_implemented", "handler": name}); err != nil {
			log.Printf("stub[%s]: encode error: %v", name, err)
		}
	}
}

// ListFunnels is implemented in funnels.go (Phase 2e-1, read + audit).
// UpdateFunnelPage stays a stub: GHL's public API v2 has no funnel/page write
// endpoint, so pixel/head-code injection via a location PIT is not possible.
// Phase 2e-2 (deferred, pending user decision) covers the write story — see
// docs/funnels/verified-implementation-path.md §2e-2.
func UpdateFunnelPage(vault *store.Vault) http.HandlerFunc     { return stub("UpdateFunnelPage") }
func UpdateNumbersLibrary(vault *store.Vault) http.HandlerFunc { return stub("UpdateNumbersLibrary") }
