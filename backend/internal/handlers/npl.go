package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/ipagoagaNNN/nnn-ghl-manager/backend/internal/store"
)

// defaultNPLKey is the custom-value name the New Patient Link scanner looks for.
// Owner-confirmed default (M3 decision D-3.1, 2026-06-04); override with ?key=.
const defaultNPLKey = "New Patient Link"

type nplResult struct {
	LocationID string `json:"locationId"`
	Name       string `json:"name"`             // account display name (from vault meta)
	Domain     string `json:"domain"`           // account's registered domain (from vault meta)
	CVID       string `json:"cvId,omitempty"`   // matched custom value's id (empty if the CV doesn't exist)
	CVName     string `json:"cvName,omitempty"` // matched custom value's real name (for the PUT)
	Value      string `json:"value"`            // the NPL value found (empty if missing)
	Verdict    string `json:"verdict"`          // valid | valid_unverified | missing | malformed | domain_mismatch | error
	Detail     string `json:"detail,omitempty"` // human-readable explanation
	Error      string `json:"error,omitempty"`  // set when the account's CVs couldn't be fetched
}

type nplSummary struct {
	Total          int `json:"total"`
	Valid          int `json:"valid"` // valid + valid_unverified
	Missing        int `json:"missing"`
	Malformed      int `json:"malformed"`
	DomainMismatch int `json:"domainMismatch"`
	Errors         int `json:"errors"`
}

// NPLScan audits the "New Patient Link" custom value across the requested
// locations (?locationIds=a,b,c, optional ?key=). It is READ-ONLY: it reuses the
// CV fan-out and validates each account's booking link against its registered
// domain. The actual fix is applied through the existing POST /api/cv/bulk — this
// endpoint never writes to GHL. See M3 decision D-3.1 (module-decisions).
func NPLScan(vault *store.Vault) http.HandlerFunc {
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
		key := strings.TrimSpace(r.URL.Query().Get("key"))
		if key == "" {
			key = defaultNPLKey
		}

		out := make([]nplResult, len(ids))
		sem := make(chan struct{}, maxCVConcurrency)
		var wg sync.WaitGroup

		for i, id := range ids {
			wg.Add(1)
			go func(idx int, locationID string) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()

				res := nplResult{LocationID: locationID}
				if meta, ok := vault.LocMetaFor(locationID); ok {
					res.Name = meta.Name
					res.Domain = meta.Domain
				}

				token, ok := vault.LocToken(locationID)
				if !ok || token == "" {
					res.Verdict = "error"
					res.Error = "no token for this location"
					out[idx] = res
					return
				}

				cvs, err := fetchCustomValues(r, client, token, locationID)
				if err != nil {
					res.Verdict = "error"
					res.Error = err.Error()
					out[idx] = res
					return
				}

				cv, found := findCVByName(cvs, key)
				if found {
					res.CVID = cv.ID
					res.CVName = cv.Name
				}
				if !found || strings.TrimSpace(cv.Value) == "" {
					res.Verdict = "missing"
					res.Detail = "no '" + key + "' custom value set"
					out[idx] = res
					return
				}
				res.Value = cv.Value
				res.Verdict, res.Detail = classifyNPL(cv.Value, res.Domain)
				out[idx] = res
			}(i, id)
		}
		wg.Wait()

		sum := nplSummary{Total: len(out)}
		for _, res := range out {
			switch res.Verdict {
			case "valid", "valid_unverified":
				sum.Valid++
			case "missing":
				sum.Missing++
			case "malformed":
				sum.Malformed++
			case "domain_mismatch":
				sum.DomainMismatch++
			case "error":
				sum.Errors++
			}
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{
			"key":     key,
			"results": out,
			"summary": sum,
		}); err != nil {
			log.Printf("npl scan: encode response error: %v", err)
		}
	}
}

// findCVByName returns the first custom value whose name matches target
// (case-insensitive, trimmed).
func findCVByName(cvs []cvItem, target string) (cvItem, bool) {
	t := strings.ToLower(strings.TrimSpace(target))
	for _, cv := range cvs {
		if strings.ToLower(strings.TrimSpace(cv.Name)) == t {
			return cv, true
		}
	}
	return cvItem{}, false
}

// classifyNPL validates a New Patient Link value against the account's registered
// domain. Returns (verdict, detail). Verdicts: valid, valid_unverified, malformed,
// domain_mismatch. It parses the URL only — it never fetches it (no SSRF surface).
func classifyNPL(value, domain string) (string, string) {
	u, err := url.Parse(strings.TrimSpace(value))
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return "malformed", "not a valid absolute http(s) URL"
	}
	host := normalizeHost(u.Host)
	want := normalizeHost(domain)
	if want == "" {
		return "valid_unverified", "URL is well-formed; no registered domain to verify against"
	}
	if host == want || strings.HasSuffix(host, "."+want) {
		return "valid", "link host matches the registered domain"
	}
	return "domain_mismatch", "link host '" + host + "' does not match registered domain '" + want + "'"
}

// normalizeHost lowercases a host/domain and strips any scheme, leading "www.",
// and trailing port/path/query so a bare domain and a full URL compare equal.
func normalizeHost(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return ""
	}
	if i := strings.Index(s, "://"); i >= 0 {
		s = s[i+3:]
	}
	s = strings.TrimPrefix(s, "www.")
	if i := strings.IndexAny(s, "/:?"); i >= 0 {
		s = s[:i]
	}
	return s
}
