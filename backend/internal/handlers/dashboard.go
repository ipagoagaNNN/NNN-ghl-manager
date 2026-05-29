package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ipagoagaNNN/nnn-ghl-manager/backend/internal/store"
)

const (
	dashCacheTTL  = 60 * time.Second
	dashPageSize  = 100
	dashMaxPages  = 50 // bound work: up to 5000 contacts per location
	dashPageSleep = 60 * time.Millisecond
)

// rawContact captures the fields we aggregate. GHL contacts inconsistently
// populate dateAdded vs createdAt, so we read both (prototype does the same).
type rawContact struct {
	DateAdded string `json:"dateAdded"`
	CreatedAt string `json:"createdAt"`
	Source    string `json:"source"`
}

type dayCount struct {
	Date  string `json:"date"`  // YYYY-MM-DD
	Count int    `json:"count"`
}

type sourceCount struct {
	Source string `json:"source"`
	Count  int    `json:"count"`
}

type dashboardData struct {
	LocationID string        `json:"locationId"`
	Total      int           `json:"total"`
	LeadsByDay []dayCount    `json:"leadsByDay"`
	TopSources []sourceCount `json:"topSources"`
	FetchedAt  string        `json:"fetchedAt"`
	Truncated  bool          `json:"truncated"` // true if hit dashMaxPages cap
}

type dashCacheEntry struct {
	data      dashboardData
	expiresAt time.Time
}

// DashboardContacts paginates a location's contacts, aggregates leads-by-day and
// top sources, and returns the summary. Results are cached in-memory for 60s per
// (locationId + date range) to avoid hammering GHL on repeated panel opens.
func DashboardContacts(vault *store.Vault) http.HandlerFunc {
	client := &http.Client{Timeout: 30 * time.Second}
	var mu sync.RWMutex
	cache := make(map[string]dashCacheEntry)

	return func(w http.ResponseWriter, r *http.Request) {
		locationID := r.PathValue("locationId")
		if locationID == "" {
			http.Error(w, "locationId required", http.StatusBadRequest)
			return
		}
		startDate := r.URL.Query().Get("startDate")
		endDate := r.URL.Query().Get("endDate")
		cacheKey := locationID + "|" + startDate + "|" + endDate

		// Serve from cache if fresh. (time.Now is allowed in handlers; only workflow
		// scripts forbid it.)
		mu.RLock()
		if ent, ok := cache[cacheKey]; ok && time.Now().Before(ent.expiresAt) {
			mu.RUnlock()
			writeJSON(w, ent.data)
			return
		}
		mu.RUnlock()

		token, ok := vault.LocToken(locationID)
		if !ok || token == "" {
			http.Error(w, "no token for this location — save one via POST /api/tokens/{locationId}", http.StatusUnauthorized)
			return
		}

		data, err := aggregateContacts(r, client, token, locationID, startDate, endDate)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}

		mu.Lock()
		cache[cacheKey] = dashCacheEntry{data: data, expiresAt: time.Now().Add(dashCacheTTL)}
		mu.Unlock()

		writeJSON(w, data)
	}
}

func aggregateContacts(r *http.Request, client *http.Client, token, locationID, startDate, endDate string) (dashboardData, error) {
	base := store.GHLBase()
	byDay := map[string]int{}
	bySource := map[string]int{}
	total := 0
	truncated := false

	// Date filtering is done server-side here, NOT via GHL query params. The
	// production prototype deliberately fetches all contacts and filters by
	// dateAdded||createdAt (lines 3310/3461) rather than trusting GHL's
	// startDate/endDate, which it conspicuously never sends.
	startBound := normalizeDay(startDate)
	endBound := normalizeDay(endDate)

	for page := 1; page <= dashMaxPages; page++ {
		q := url.Values{}
		q.Set("locationId", locationID)
		q.Set("limit", fmt.Sprintf("%d", dashPageSize))
		q.Set("sortBy", "date_added")
		q.Set("page", fmt.Sprintf("%d", page))
		// NOTE: startDate/endDate intentionally NOT sent to GHL — filtered server-side below.

		target := base + "/contacts/?" + q.Encode()
		if !strings.HasPrefix(target, base) {
			return dashboardData{}, fmt.Errorf("invalid GHL target")
		}

		// #nosec G107 G704 -- base host hardcoded + validated; only query params vary
		req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, target, nil)
		if err != nil {
			return dashboardData{}, err
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Version", "2021-07-28")

		// #nosec G107 G704 -- see above
		resp, err := client.Do(req)
		if err != nil {
			return dashboardData{}, err
		}

		if !isOK(resp.StatusCode) {
			b, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			return dashboardData{}, fmt.Errorf("GHL HTTP %d: %s", resp.StatusCode, truncate(string(b), 160))
		}

		var data struct {
			Contacts []rawContact `json:"contacts"`
			Data     []rawContact `json:"data"`
		}
		decErr := json.NewDecoder(resp.Body).Decode(&data)
		if cerr := resp.Body.Close(); cerr != nil {
			log.Printf("dashboard: body close error: %v", cerr)
		}
		if decErr != nil {
			return dashboardData{}, decErr
		}

		// Mirror prototype: contacts may arrive under "contacts" or "data".
		// Use the resolved batch length for pagination so an alt envelope
		// doesn't make us stop after page 1.
		batch := data.Contacts
		if len(batch) == 0 {
			batch = data.Data
		}

		for _, c := range batch {
			// Prototype uses dateAdded || createdAt — some contacts only have one.
			day := c.DateAdded
			if day == "" {
				day = c.CreatedAt
			}
			if len(day) >= 10 {
				day = day[:10] // YYYY-MM-DD
			}
			// Server-side date filter (mirrors prototype's client-side filter).
			// A contact with no date is excluded when any bound is set.
			if startBound != "" && (day == "" || day < startBound) {
				continue
			}
			if endBound != "" && (day == "" || day > endBound) {
				continue
			}
			total++
			if day != "" {
				byDay[day]++
			}
			src := strings.TrimSpace(c.Source)
			if src == "" {
				src = "(unknown)"
			}
			bySource[src]++
		}

		if len(batch) < dashPageSize {
			break // last page
		}
		if page == dashMaxPages {
			truncated = true
			break
		}
		time.Sleep(dashPageSleep)
	}

	return dashboardData{
		LocationID: locationID,
		Total:      total,
		LeadsByDay: sortDayCounts(byDay),
		TopSources: topSourceCounts(bySource, 10),
		FetchedAt:  time.Now().UTC().Format(time.RFC3339),
		Truncated:  truncated,
	}, nil
}

// normalizeDay trims a date/datetime string to its YYYY-MM-DD prefix for
// string-comparison filtering (matches prototype's date handling).
func normalizeDay(s string) string {
	if len(s) >= 10 {
		return s[:10]
	}
	return s
}

func sortDayCounts(m map[string]int) []dayCount {
	out := make([]dayCount, 0, len(m))
	for d, c := range m {
		out = append(out, dayCount{Date: d, Count: c})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Date < out[j].Date })
	return out
}

func topSourceCounts(m map[string]int, n int) []sourceCount {
	out := make([]sourceCount, 0, len(m))
	for s, c := range m {
		out = append(out, sourceCount{Source: s, Count: c})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].Source < out[j].Source
	})
	if len(out) > n {
		out = out[:n]
	}
	return out
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("dashboard: encode response error: %v", err)
	}
}
