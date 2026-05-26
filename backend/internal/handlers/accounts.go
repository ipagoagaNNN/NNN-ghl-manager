package handlers

import (
	"encoding/json"
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
		json.NewEncoder(w).Encode(map[string]any{
			"configuredLocations": ids,
			"count":               len(ids),
		})
	}
}
