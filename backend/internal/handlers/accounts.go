package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/ipagoagaNNN/nnn-ghl-manager/backend/internal/store"
)

// ListAccounts returns the list of location IDs that have tokens in the vault.
// The actual location metadata was returned at connect time; this just shows
// which accounts are configured with tokens.
func ListAccounts(vault *store.Vault) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tokens := vault.AllLocTokens()
		ids := make([]string, 0, len(tokens))
		for id := range tokens {
			ids = append(ids, id)
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{
			"configuredLocations": ids,
			"count":               len(ids),
		}); err != nil {
			log.Printf("accounts: encode response error: %v", err)
		}
	}
}

// UpdateLocMeta upserts per-location metadata (domain, acuity, calendar IDs, active flag).
// Token is NOT accepted here — that path is POST /api/tokens/:locationId.
func UpdateLocMeta(vault *store.Vault) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		locationID := r.PathValue("locationId")
		if locationID == "" {
			http.Error(w, "locationId required", http.StatusBadRequest)
			return
		}
		var req store.LocMeta
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json: "+err.Error(), http.StatusBadRequest)
			return
		}
		vault.SetLocMeta(locationID, req)
		w.WriteHeader(http.StatusNoContent)
	}
}

// ListLibrary returns all configured location metadata + which IDs have tokens.
// Used by the Accounts page to pre-fill the per-location editor.
// Token VALUES are never returned — only the boolean hasToken flag per ID.
func ListLibrary(vault *store.Vault) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		meta := vault.AllLocMeta()
		tokens := vault.AllLocTokens()

		type entry struct {
			LocationID  string `json:"locationId"`
			Name        string `json:"name"`
			Domain      string `json:"domain"`
			AcuityField string `json:"acuityField"`
			CalendarIDs string `json:"calendarIds"`
			Active      bool   `json:"active"`
			HasToken    bool   `json:"hasToken"`
		}

		// Union of all keys: any location with meta OR a token gets a row.
		ids := map[string]struct{}{}
		for id := range meta {
			ids[id] = struct{}{}
		}
		for id := range tokens {
			ids[id] = struct{}{}
		}

		out := make([]entry, 0, len(ids))
		for id := range ids {
			m := meta[id]
			_, hasTok := tokens[id]
			out = append(out, entry{
				LocationID:  id,
				Name:        m.Name,
				Domain:      m.Domain,
				AcuityField: m.AcuityField,
				CalendarIDs: m.CalendarIDs,
				Active:      m.Active,
				HasToken:    hasTok,
			})
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{
			"library": out,
			"count":   len(out),
		}); err != nil {
			log.Printf("accounts library: encode response error: %v", err)
		}
	}
}
