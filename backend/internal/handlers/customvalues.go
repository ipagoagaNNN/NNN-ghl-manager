package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/ipagoagaNNN/nnn-ghl-manager/backend/internal/store"
)

// maxCVConcurrency bounds parallel GHL calls so we don't trip GHL's rate limits
// when fanning out across many locations.
const maxCVConcurrency = 6

type cvItem struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Value     string `json:"value"`
	FieldType string `json:"fieldType,omitempty"`
}

type locationCV struct {
	LocationID string   `json:"locationId"`
	Name       string   `json:"name"`
	CVs        []cvItem `json:"cvs"`
	Error      string   `json:"error,omitempty"`
}

// ListCV fans out across the requested locations (?locationIds=a,b,c) and returns
// each location's custom values. Each location is fetched in parallel with bounded
// concurrency. Tokens are read from the vault server-side and never returned.
func ListCV(vault *store.Vault) http.HandlerFunc {
	client := &http.Client{Timeout: 30 * time.Second}

	return func(w http.ResponseWriter, r *http.Request) {
		raw := r.URL.Query().Get("locationIds")
		if strings.TrimSpace(raw) == "" {
			http.Error(w, "locationIds query param required (comma-separated)", http.StatusBadRequest)
			return
		}
		ids := splitIDs(raw)
		if len(ids) == 0 {
			http.Error(w, "no valid locationIds", http.StatusBadRequest)
			return
		}

		out := make([]locationCV, len(ids))
		sem := make(chan struct{}, maxCVConcurrency)
		var wg sync.WaitGroup

		for i, id := range ids {
			wg.Add(1)
			go func(idx int, locationID string) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()

				res := locationCV{LocationID: locationID, CVs: []cvItem{}}
				if meta, ok := vault.LocMetaFor(locationID); ok {
					res.Name = meta.Name
				}

				token, ok := vault.LocToken(locationID)
				if !ok || token == "" {
					res.Error = "no token for this location"
					out[idx] = res
					return
				}

				cvs, err := fetchCustomValues(r, client, token, locationID)
				if err != nil {
					res.Error = err.Error()
					out[idx] = res
					return
				}
				res.CVs = cvs
				out[idx] = res
			}(i, id)
		}
		wg.Wait()

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{
			"locations": out,
			"count":     len(out),
		}); err != nil {
			log.Printf("cv list: encode response error: %v", err)
		}
	}
}

func fetchCustomValues(r *http.Request, client *http.Client, token, locationID string) ([]cvItem, error) {
	base := store.GHLBase()
	target := fmt.Sprintf("%s/locations/%s/customValues", base, url.PathEscape(locationID))
	if !strings.HasPrefix(target, base) {
		return nil, fmt.Errorf("invalid GHL target")
	}

	// #nosec G107 G704 -- base host hardcoded + validated; only escaped locationId varies
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, target, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Version", "2021-07-28")

	// #nosec G107 G704 -- see above
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			log.Printf("cv fetch: body close error: %v", cerr)
		}
	}()

	if !isOK(resp.StatusCode) {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GHL HTTP %d: %s", resp.StatusCode, truncate(string(b), 160))
	}

	var data struct {
		CustomValues []cvItem `json:"customValues"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	if data.CustomValues == nil {
		data.CustomValues = []cvItem{}
	}
	return data.CustomValues, nil
}

type cvUpdate struct {
	LocationID    string `json:"locationId"`
	CustomValueID string `json:"customValueId"`
	Value         string `json:"value"`
}

type cvUpdateResult struct {
	LocationID    string `json:"locationId"`
	CustomValueID string `json:"customValueId"`
	OK            bool   `json:"ok"`
	Error         string `json:"error,omitempty"`
}

// BulkUpdateCV applies a batch of custom-value updates in parallel (bounded).
// Each update PUTs to GHL /locations/{id}/customValues/{cvId}. Per-update results
// are returned so the UI can show partial success.
func BulkUpdateCV(vault *store.Vault) http.HandlerFunc {
	client := &http.Client{Timeout: 30 * time.Second}

	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Updates []cvUpdate `json:"updates"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "invalid json: "+err.Error(), http.StatusBadRequest)
			return
		}
		if len(body.Updates) == 0 {
			http.Error(w, "no updates provided", http.StatusBadRequest)
			return
		}

		results := make([]cvUpdateResult, len(body.Updates))
		sem := make(chan struct{}, maxCVConcurrency)
		var wg sync.WaitGroup

		for i, u := range body.Updates {
			wg.Add(1)
			go func(idx int, upd cvUpdate) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()

				res := cvUpdateResult{LocationID: upd.LocationID, CustomValueID: upd.CustomValueID}
				if upd.LocationID == "" || upd.CustomValueID == "" {
					res.Error = "locationId and customValueId required"
					results[idx] = res
					return
				}
				token, ok := vault.LocToken(upd.LocationID)
				if !ok || token == "" {
					res.Error = "no token for this location"
					results[idx] = res
					return
				}
				if err := putCustomValue(r, client, token, upd); err != nil {
					res.Error = err.Error()
					results[idx] = res
					return
				}
				res.OK = true
				results[idx] = res
			}(i, u)
		}
		wg.Wait()

		okCount := 0
		for _, res := range results {
			if res.OK {
				okCount++
			}
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{
			"results":    results,
			"okCount":    okCount,
			"errorCount": len(results) - okCount,
		}); err != nil {
			log.Printf("cv bulk: encode response error: %v", err)
		}
	}
}

func putCustomValue(r *http.Request, client *http.Client, token string, upd cvUpdate) error {
	base := store.GHLBase()
	target := fmt.Sprintf("%s/locations/%s/customValues/%s",
		base, url.PathEscape(upd.LocationID), url.PathEscape(upd.CustomValueID))
	if !strings.HasPrefix(target, base) {
		return fmt.Errorf("invalid GHL target")
	}

	payload, err := json.Marshal(map[string]string{"value": upd.Value})
	if err != nil {
		return err
	}

	// #nosec G107 G704 -- base host hardcoded + validated; only escaped IDs vary
	req, err := http.NewRequestWithContext(r.Context(), http.MethodPut, target, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Version", "2021-07-28")
	req.Header.Set("Content-Type", "application/json")

	// #nosec G107 G704 -- see above
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			log.Printf("cv put: body close error: %v", cerr)
		}
	}()

	if !isOK(resp.StatusCode) {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("GHL HTTP %d: %s", resp.StatusCode, truncate(string(b), 160))
	}
	return nil
}

// splitIDs parses a comma-separated id list, trimming blanks.
func splitIDs(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}
