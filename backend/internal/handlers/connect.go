package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
		json.NewEncoder(w).Encode(connectResponse{
			LocationCount: len(locs),
			Locations:     locs,
		})
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
