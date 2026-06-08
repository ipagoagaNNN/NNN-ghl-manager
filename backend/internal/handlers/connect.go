package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/ipagoagaNNN/nnn-ghl-manager/backend/internal/store"
)

type connectRequest struct {
	AgencyToken string `json:"agencyToken"`
	CompanyID   string `json:"companyId"`
}

type connectResponse struct {
	LocationCount int        `json:"locationCount"`
	Locations     []location `json:"locations"`
}

type location struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	BusinessName string `json:"businessName,omitempty"`
}

// Connect validates the agency token against GHL and stores it in the vault.
// The token is never returned to the frontend.
func Connect(vault *store.Vault) http.HandlerFunc {
	client := &http.Client{Timeout: 30 * time.Second}

	return func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		var req connectRequest
		if err := json.Unmarshal(body, &req); err != nil || req.AgencyToken == "" || req.CompanyID == "" {
			http.Error(w, "agencyToken and companyId required", http.StatusBadRequest)
			return
		}

		// Validate token against GHL
		locs, err := fetchAllLocations(client, req.AgencyToken, req.CompanyID)
		if err != nil {
			http.Error(w, fmt.Sprintf("GHL error: %v", err), http.StatusBadGateway)
			return
		}
		if len(locs) == 0 {
			http.Error(w, "no sub-accounts found for this company ID", http.StatusUnprocessableEntity)
			return
		}

		// Store server-side — never returned to frontend
		vault.SetAgency(req.AgencyToken, req.CompanyID)

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(connectResponse{
			LocationCount: len(locs),
			Locations:     locs,
		}); err != nil {
			log.Printf("connect: encode response error: %v", err)
		}
	}
}

func fetchAllLocations(client *http.Client, agencyToken, companyID string) ([]location, error) {
	var all []location
	skip := 0
	for {
		url := fmt.Sprintf("%s/locations/search?companyId=%s&skip=%d&limit=100", store.GHLBase(), companyID, skip)
		req, _ := http.NewRequest(http.MethodGet, url, nil)
		req.Header.Set("Authorization", "Bearer "+agencyToken)
		req.Header.Set("Version", "2021-07-28")

		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if !isOK(resp.StatusCode) {
			b, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, truncate(string(b), 120))
		}

		var data struct {
			Locations []struct {
				ID           string `json:"id"`
				Name         string `json:"name"`
				BusinessName string `json:"businessName"`
			} `json:"locations"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
			return nil, err
		}
		for _, l := range data.Locations {
			all = append(all, location{ID: l.ID, Name: l.Name, BusinessName: l.BusinessName})
		}
		if len(data.Locations) < 100 {
			break
		}
		skip += 100
	}
	return all, nil
}

type connectLocationRequest struct {
	LocationID string `json:"locationId"`
	Token      string `json:"token"`
}

type connectLocationResponse struct {
	LocationID string `json:"locationId"`
	Name       string `json:"name"`
	Valid      bool   `json:"valid"`
}

// ConnectLocation validates a sub-account (location-scoped) Private Integration
// Token directly, bypassing agency discovery (/locations/search). This supports
// users who only hold location-scoped PITs — which return 403 on the agency
// search endpoint but 200 on location endpoints. On success the token is stored
// in the vault and the location becomes usable by every module.
func ConnectLocation(vault *store.Vault) http.HandlerFunc {
	client := &http.Client{Timeout: 30 * time.Second}

	return func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		var req connectLocationRequest
		if err := json.Unmarshal(body, &req); err != nil || req.LocationID == "" || req.Token == "" {
			http.Error(w, "locationId and token required", http.StatusBadRequest)
			return
		}

		// Validate the token AND fetch the custom values in one call: a location PIT
		// returns 200 here, and the CVs let us resolve a friendly account name (GHL
		// exposes it as the "Location Name" custom value) instead of showing the raw id.
		cvs, err := fetchCustomValues(r, client, req.Token, req.LocationID)
		if err != nil {
			http.Error(w, fmt.Sprintf("token validation failed: %v", err), http.StatusUnauthorized)
			return
		}

		vault.SetLocToken(req.LocationID, req.Token)

		name := req.LocationID
		if n := extractLocationName(cvs); n != "" {
			name = n
		}
		// Store the resolved name, upgrading a placeholder (id) name but never
		// clobbering a name the user set on the Accounts page.
		if meta, ok := vault.LocMetaFor(req.LocationID); !ok {
			vault.SetLocMeta(req.LocationID, store.LocMeta{Name: name, Active: true})
		} else if meta.Name == "" || meta.Name == req.LocationID {
			meta.Name = name
			vault.SetLocMeta(req.LocationID, meta)
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(connectLocationResponse{
			LocationID: req.LocationID,
			Name:       name,
			Valid:      true,
		}); err != nil {
			log.Printf("connect location: encode error: %v", err)
		}
	}
}

// extractLocationName returns a friendly account name from the location's custom
// values — GHL exposes it as the "Location Name" custom value (key
// custom_values.location_name). Empty if none is set; the caller falls back to
// the locationId.
func extractLocationName(cvs []cvItem) string {
	for _, cv := range cvs {
		switch strings.ToLower(strings.TrimSpace(cv.Name)) {
		case "location name", "location_name", "account name", "business name":
			if v := strings.TrimSpace(cv.Value); v != "" {
				return v
			}
		}
	}
	return ""
}

// SaveToken stores a sub-account token server-side.
func SaveToken(vault *store.Vault) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		locationID := r.PathValue("locationId")
		if locationID == "" {
			http.Error(w, "locationId required", http.StatusBadRequest)
			return
		}
		var req struct {
			Token string `json:"token"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Token == "" {
			http.Error(w, "token required", http.StatusBadRequest)
			return
		}
		vault.SetLocToken(locationID, req.Token)
		w.WriteHeader(http.StatusNoContent)
	}
}

func isOK(status int) bool     { return status >= 200 && status < 300 }
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
