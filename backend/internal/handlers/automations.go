package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ipagoagaNNN/nnn-ghl-manager/backend/internal/store"
)

type workflow struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Status    string `json:"status"`
	Version   int    `json:"version"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

// ListWorkflows returns the GHL workflows (automations) for a single location.
// Module 5 (Automations) is read-only — no mutation endpoints.
// The sub-account token is read from the vault server-side; it never reaches the browser.
func ListWorkflows(vault *store.Vault) http.HandlerFunc {
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

		base := store.GHLBase()
		target := fmt.Sprintf("%s/workflows/?locationId=%s", base, url.QueryEscape(locationID))
		// Defensive: target host is locked to the hardcoded GHL base.
		if !strings.HasPrefix(target, base) {
			http.Error(w, "invalid GHL target", http.StatusInternalServerError)
			return
		}

		// #nosec G107 G704 -- base host hardcoded + validated above; only the escaped locationId varies
		req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, target, nil)
		if err != nil {
			http.Error(w, fmt.Sprintf("request build error: %v", err), http.StatusInternalServerError)
			return
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Version", "2021-07-28")

		// #nosec G107 G704 -- target host validated above; only escaped locationId varies
		resp, err := client.Do(req)
		if err != nil {
			http.Error(w, fmt.Sprintf("GHL error: %v", err), http.StatusBadGateway)
			return
		}
		defer func() {
			if cerr := resp.Body.Close(); cerr != nil {
				log.Printf("workflows: body close error: %v", cerr)
			}
		}()

		if !isOK(resp.StatusCode) {
			b, _ := io.ReadAll(resp.Body)
			http.Error(w, fmt.Sprintf("GHL HTTP %d: %s", resp.StatusCode, truncate(string(b), 200)), resp.StatusCode)
			return
		}

		var data struct {
			Workflows []workflow `json:"workflows"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
			http.Error(w, fmt.Sprintf("decode error: %v", err), http.StatusBadGateway)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{
			"locationId": locationID,
			"workflows":  data.Workflows,
			"count":      len(data.Workflows),
		}); err != nil {
			log.Printf("workflows: encode response error: %v", err)
		}
	}
}
