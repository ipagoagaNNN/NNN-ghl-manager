package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/ipagoagaNNN/nnn-ghl-manager/backend/internal/store"
)

type customField struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	DataType string `json:"dataType,omitempty"`
	FieldKey string `json:"fieldKey,omitempty"`
	Position int    `json:"position,omitempty"`
}

// ListCustomFields returns a location's custom-field schema
// (GET /locations/{id}/customFields, Version 2021-04-15). Read-only; used to
// resolve campaign/field names (Dashboard M6) and by Custom Values (M3).
func ListCustomFields(vault *store.Vault) http.HandlerFunc {
	client := &http.Client{Timeout: 30 * time.Second}

	return func(w http.ResponseWriter, r *http.Request) {
		locationID := r.PathValue("locationId")
		if locationID == "" {
			http.Error(w, "locationId required", http.StatusBadRequest)
			return
		}
		token, ok := vault.LocToken(locationID)
		if !ok || token == "" {
			http.Error(w, "no token for this location — save one via POST /api/tokens/{locationId}", http.StatusUnauthorized)
			return
		}

		target := fmt.Sprintf("%s/locations/%s/customFields", store.GHLBase(), url.PathEscape(locationID))
		body, status, err := ghlRequest(r.Context(), client, http.MethodGet, token, target, nil)
		if err != nil {
			http.Error(w, fmt.Sprintf("GHL error: %v", err), http.StatusBadGateway)
			return
		}
		if !isOK(status) {
			http.Error(w, fmt.Sprintf("GHL HTTP %d: %s", status, truncate(string(body), 200)), status)
			return
		}

		var data struct {
			CustomFields []customField `json:"customFields"`
			CustomField  []customField `json:"customField"`
			Data         []customField `json:"data"`
		}
		if err := json.Unmarshal(body, &data); err != nil {
			http.Error(w, fmt.Sprintf("decode error: %v", err), http.StatusBadGateway)
			return
		}
		fields := data.CustomFields
		if len(fields) == 0 {
			fields = data.CustomField
		}
		if len(fields) == 0 {
			fields = data.Data
		}
		if fields == nil {
			fields = []customField{}
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{
			"locationId":   locationID,
			"customFields": fields,
			"count":        len(fields),
		}); err != nil {
			log.Printf("customfields: encode response error: %v", err)
		}
	}
}
