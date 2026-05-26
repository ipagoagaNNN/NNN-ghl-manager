package handlers

// Stub handlers — each will be implemented as its module is ported.
// All follow the same pattern: validate, fetch from GHL via vault token, return JSON.

import (
	"encoding/json"
	"net/http"

	"github.com/ipagoagaNNN/nnn-ghl-manager/backend/internal/store"
)

func stub(name string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "not_yet_implemented", "handler": name})
	}
}

func ListCV(vault *store.Vault) http.HandlerFunc         { return stub("ListCV") }
func BulkUpdateCV(vault *store.Vault) http.HandlerFunc   { return stub("BulkUpdateCV") }
func ListWorkflows(vault *store.Vault) http.HandlerFunc  { return stub("ListWorkflows") }
func ListFunnels(vault *store.Vault) http.HandlerFunc    { return stub("ListFunnels") }
func UpdateFunnelPage(vault *store.Vault) http.HandlerFunc { return stub("UpdateFunnelPage") }
func DashboardContacts(vault *store.Vault) http.HandlerFunc { return stub("DashboardContacts") }
func UpdateNumbersLibrary(vault *store.Vault) http.HandlerFunc { return stub("UpdateNumbersLibrary") }
