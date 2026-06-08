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

type ghlObject struct {
	ID                     string            `json:"id"`
	Key                    string            `json:"key"`
	Labels                 map[string]string `json:"labels,omitempty"`
	Description            string            `json:"description,omitempty"`
	PrimaryDisplayProperty string            `json:"primaryDisplayProperty,omitempty"`
	LocationID             string            `json:"locationId,omitempty"`
	Type                   string            `json:"type,omitempty"`
}

// ListObjects returns a location's object schema — built-in business/opportunity/
// contact plus any custom objects (GET /objects/?locationId=, Version 2023-02-21,
// which this endpoint REQUIRES). Read-only. Entry point for a future Objects/Records
// module; also a cheap location-PIT validation call.
func ListObjects(vault *store.Vault) http.HandlerFunc {
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

		target := fmt.Sprintf("%s/objects/?locationId=%s", store.GHLBase(), url.QueryEscape(locationID))
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
			Objects []ghlObject `json:"objects"`
		}
		if err := json.Unmarshal(body, &data); err != nil {
			http.Error(w, fmt.Sprintf("decode error: %v", err), http.StatusBadGateway)
			return
		}
		if data.Objects == nil {
			data.Objects = []ghlObject{}
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{
			"locationId": locationID,
			"objects":    data.Objects,
			"count":      len(data.Objects),
		}); err != nil {
			log.Printf("objects: encode response error: %v", err)
		}
	}
}
