package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/ipagoagaNNN/nnn-ghl-manager/backend/internal/store"
)

const (
	// maxFunnelConcurrency bounds parallel live-page fetches during an audit.
	maxFunnelConcurrency = 6
	// maxPageBytes caps how much of each fetched page we read (defends against huge bodies).
	maxPageBytes = 2 << 20 // 2 MiB
	// pageFetchTimeout bounds a single live-page fetch.
	pageFetchTimeout = 12 * time.Second
	// funnelAuditCacheTTL is how long an audit result is reused.
	funnelAuditCacheTTL = 60 * time.Second
)

// --- GHL decode types (envelope + field-name fallbacks, per prototype) ---

type rawFunnel struct {
	ID         string    `json:"id"`
	IDAlt      string    `json:"_id"`
	Name       string    `json:"name"`
	Title      string    `json:"title"`
	Domain     string    `json:"domain"`
	DomainName string    `json:"domainName"`
	URL        string    `json:"url"`
	PreviewURL string    `json:"previewUrl"`
	DomainID   string    `json:"domainId"`
	Steps      []rawStep `json:"steps"`
	// Funnel-level tracking code. Live API returns trackingCodeHead/Body; the
	// prototype also looked for head/body/header/footer variants. Concatenated
	// for a domain-independent pixel signal (see funnelTrackingCode).
	TrackingCodeHead string `json:"trackingCodeHead"`
	TrackingCodeBody string `json:"trackingCodeBody"`
	HeadCode         string `json:"headCode"`
	HeaderCode       string `json:"headerCode"`
	BodyCode         string `json:"bodyCode"`
	FooterCode       string `json:"footerCode"`
}

// funnelTrackingCode joins every funnel-level tracking-code field so auditHTML
// can detect a configured pixel without fetching a live page.
func (f rawFunnel) funnelTrackingCode() string {
	return strings.Join([]string{
		f.TrackingCodeHead, f.TrackingCodeBody,
		f.HeadCode, f.HeaderCode, f.BodyCode, f.FooterCode,
	}, "\n")
}

func (f rawFunnel) id() string {
	if f.ID != "" {
		return f.ID
	}
	return f.IDAlt
}

func (f rawFunnel) name() string {
	if f.Name != "" {
		return f.Name
	}
	return f.Title
}

func (f rawFunnel) directDomain() string {
	for _, d := range []string{f.Domain, f.DomainName, f.URL, f.PreviewURL} {
		if c := cleanDomain(d); c != "" {
			return c
		}
	}
	return ""
}

type rawStep struct {
	Name  string `json:"name"`
	Title string `json:"title"`
	Slug  string `json:"slug"`
	Path  string `json:"path"`
	URL   string `json:"url"`
}

func (s rawStep) name() string {
	if s.Name != "" {
		return s.Name
	}
	if s.Title != "" {
		return s.Title
	}
	return "Unnamed step"
}

func (s rawStep) slug() string {
	for _, v := range []string{s.Slug, s.Path, s.URL} {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

type funnelListEnvelope struct {
	Funnels []rawFunnel `json:"funnels"`
	Data    []rawFunnel `json:"data"`
	List    []rawFunnel `json:"list"`
	Items   []rawFunnel `json:"items"`
}

func (e funnelListEnvelope) pick() []rawFunnel {
	switch {
	case len(e.Funnels) > 0:
		return e.Funnels
	case len(e.Data) > 0:
		return e.Data
	case len(e.List) > 0:
		return e.List
	case len(e.Items) > 0:
		return e.Items
	default:
		return nil
	}
}

// --- API response types ---

type funnelStepOut struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
	URL  string `json:"url"`
}

type funnelOut struct {
	ID     string          `json:"id"`
	Name   string          `json:"name"`
	Domain string          `json:"domain"`
	Steps  []funnelStepOut `json:"steps"`
	// ConfiguredPixel is the Meta pixel ID found in the funnel-level tracking
	// code (read from the list API, no live fetch). HasTrackingPixel is true when
	// any pixel snippet is present. These let the UI flag pixels even when no
	// domain is configured (so the live-page audit can't build a URL).
	ConfiguredPixel  string `json:"configuredPixel,omitempty"`
	HasTrackingPixel bool   `json:"hasTrackingPixel"`
}

// ListFunnels returns the funnels for a location with each funnel's steps and
// resolved domain. Read-only; works with a location PIT and Version 2021-07-28.
func ListFunnels(vault *store.Vault) http.HandlerFunc {
	client := &http.Client{Timeout: 30 * time.Second}

	return func(w http.ResponseWriter, r *http.Request) {
		locationID := r.PathValue("locationId")
		if locationID == "" {
			http.Error(w, "locationId required", http.StatusBadRequest)
			return
		}
		token, ok := vault.LocToken(locationID)
		if !ok || token == "" {
			http.Error(w, "no token for this location", http.StatusUnauthorized)
			return
		}

		funnels, err := fetchFunnels(r.Context(), client, token, locationID)
		if err != nil {
			http.Error(w, fmt.Sprintf("GHL error: %v", err), http.StatusBadGateway)
			return
		}

		metaDomain := ""
		if meta, ok := vault.LocMetaFor(locationID); ok {
			metaDomain = cleanDomain(meta.Domain)
		}

		out := make([]funnelOut, 0, len(funnels))
		domainCache := map[string]string{}
		for _, f := range funnels {
			domain := resolveFunnelDomain(r.Context(), client, token, locationID, f, metaDomain, domainCache)
			steps := make([]funnelStepOut, 0, len(f.Steps))
			for _, s := range f.Steps {
				steps = append(steps, funnelStepOut{
					Name: s.name(),
					Slug: s.slug(),
					URL:  buildStepURL(s.slug(), domain),
				})
			}
			hasPixel, pixelID, _, _ := auditHTML(f.funnelTrackingCode())
			out = append(out, funnelOut{
				ID:               f.id(),
				Name:             f.name(),
				Domain:           domain,
				Steps:            steps,
				ConfiguredPixel:  pixelID,
				HasTrackingPixel: hasPixel,
			})
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{
			"locationId": locationID,
			"funnels":    out,
			"count":      len(out),
		}); err != nil {
			log.Printf("funnels list: encode error: %v", err)
		}
	}
}

// --- Audit ---

type pageAudit struct {
	Funnel        string `json:"funnel"`
	Step          string `json:"step"`
	URL           string `json:"url"`
	FetchOK       bool   `json:"fetchOk"`
	HasPixel      bool   `json:"hasPixel"`
	PixelID       string `json:"pixelId"`
	ExpectedPixel string `json:"expectedPixel,omitempty"`
	PixelStatus   string `json:"pixelStatus"` // ok | missing | wrong-pixel | unknown-domain | no-url
	HasUTM        bool   `json:"hasUtm"`
	HasAcuity     bool   `json:"hasAcuity"`
	Error         string `json:"error,omitempty"`
}

type auditSummary struct {
	Funnels int `json:"funnels"`
	Pages   int `json:"pages"`
	OK      int `json:"ok"`
	Missing int `json:"missing"`
	Wrong   int `json:"wrong"`
	Errors  int `json:"errors"`
}

type auditResponse struct {
	LocationID string       `json:"locationId"`
	Pages      []pageAudit  `json:"pages"`
	Summary    auditSummary `json:"summary"`
}

type auditCacheEntry struct {
	at   time.Time
	resp auditResponse
}

// AuditFunnels scans the live published pages of a location's funnels for the
// Meta pixel, UTM tracking script, and Acuity footer — fetching each page
// SERVER-SIDE (no browser, no third-party CORS proxy). Optional ?funnelId=
// limits the audit to one funnel. Results are cached for 60s per (location,funnel).
func AuditFunnels(vault *store.Vault) http.HandlerFunc {
	listClient := &http.Client{Timeout: 30 * time.Second}
	// Dedicated page client: enforces host-locked, bounded redirects.
	pageClient := &http.Client{
		Timeout: pageFetchTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return fmt.Errorf("stopped after 5 redirects")
			}
			origin := via[0].URL.Hostname()
			if !strings.EqualFold(req.URL.Hostname(), origin) {
				return fmt.Errorf("cross-host redirect blocked: %s -> %s", origin, req.URL.Hostname())
			}
			return guardPublicHost(req.URL.Hostname())
		},
	}

	var cacheMu sync.Mutex
	cache := map[string]auditCacheEntry{}

	return func(w http.ResponseWriter, r *http.Request) {
		locationID := r.PathValue("locationId")
		if locationID == "" {
			http.Error(w, "locationId required", http.StatusBadRequest)
			return
		}
		funnelID := strings.TrimSpace(r.URL.Query().Get("funnelId"))
		cacheKey := locationID + "|" + funnelID

		cacheMu.Lock()
		if ent, ok := cache[cacheKey]; ok && time.Since(ent.at) < funnelAuditCacheTTL {
			resp := ent.resp
			cacheMu.Unlock()
			writeJSON(w, resp)
			return
		}
		cacheMu.Unlock()

		token, ok := vault.LocToken(locationID)
		if !ok || token == "" {
			http.Error(w, "no token for this location", http.StatusUnauthorized)
			return
		}

		funnels, err := fetchFunnels(r.Context(), listClient, token, locationID)
		if err != nil {
			http.Error(w, fmt.Sprintf("GHL error: %v", err), http.StatusBadGateway)
			return
		}
		if funnelID != "" {
			filtered := funnels[:0:0]
			for _, f := range funnels {
				if f.id() == funnelID {
					filtered = append(filtered, f)
				}
			}
			funnels = filtered
		}

		metaDomain := ""
		if meta, ok := vault.LocMetaFor(locationID); ok {
			metaDomain = cleanDomain(meta.Domain)
		}

		// Build the work list: one entry per (funnel, step) with a usable URL.
		type job struct {
			funnel string
			step   string
			url    string
			domain string
		}
		var jobs []job
		domainCache := map[string]string{}
		for _, f := range funnels {
			domain := resolveFunnelDomain(r.Context(), listClient, token, locationID, f, metaDomain, domainCache)
			for _, s := range f.Steps {
				jobs = append(jobs, job{
					funnel: f.name(),
					step:   s.name(),
					url:    buildStepURL(s.slug(), domain),
					domain: domain,
				})
			}
		}

		results := make([]pageAudit, len(jobs))
		sem := make(chan struct{}, maxFunnelConcurrency)
		var wg sync.WaitGroup
		for i, j := range jobs {
			wg.Add(1)
			go func(idx int, jb job) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()
				results[idx] = auditOnePage(r.Context(), pageClient, jb.funnel, jb.step, jb.url, jb.domain)
			}(i, j)
		}
		wg.Wait()

		resp := auditResponse{LocationID: locationID, Pages: results}
		resp.Summary = summarize(len(funnels), results)

		cacheMu.Lock()
		cache[cacheKey] = auditCacheEntry{at: time.Now(), resp: resp}
		cacheMu.Unlock()

		writeJSON(w, resp)
	}
}

func auditOnePage(ctx context.Context, client *http.Client, funnel, step, pageURL, domain string) pageAudit {
	pa := pageAudit{Funnel: funnel, Step: step, URL: pageURL}
	expID, _, known := expectedPixelForDomain(domain)
	if known {
		pa.ExpectedPixel = expID
	}
	if pageURL == "" {
		pa.PixelStatus = "no-url"
		return pa
	}

	html, err := fetchPageHTML(ctx, client, pageURL)
	if err != nil {
		pa.Error = truncate(err.Error(), 160)
		pa.PixelStatus = "missing"
		return pa
	}
	pa.FetchOK = true
	pa.HasPixel, pa.PixelID, pa.HasUTM, pa.HasAcuity = auditHTML(html)

	switch {
	case !known:
		pa.PixelStatus = "unknown-domain"
	case !pa.HasPixel:
		pa.PixelStatus = "missing"
	case pa.PixelID != "" && pa.PixelID != expID:
		pa.PixelStatus = "wrong-pixel"
	default:
		pa.PixelStatus = "ok"
	}
	return pa
}

func summarize(funnelCount int, results []pageAudit) auditSummary {
	s := auditSummary{Funnels: funnelCount, Pages: len(results)}
	for _, r := range results {
		switch r.PixelStatus {
		case "ok":
			s.OK++
		case "wrong-pixel":
			s.Wrong++
		case "missing":
			if r.Error != "" {
				s.Errors++
			} else {
				s.Missing++
			}
		}
	}
	return s
}

// --- GHL + helpers ---

func fetchFunnels(ctx context.Context, client *http.Client, token, locationID string) ([]rawFunnel, error) {
	base := store.GHLBase()
	target := fmt.Sprintf("%s/funnels/funnel/list?locationId=%s&limit=100", base, url.QueryEscape(locationID))
	if !strings.HasPrefix(target, base) {
		return nil, fmt.Errorf("invalid GHL target")
	}

	// #nosec G107 G704 -- base host hardcoded + validated; only escaped locationId varies
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
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
			log.Printf("funnels fetch: body close error: %v", cerr)
		}
	}()
	if !isOK(resp.StatusCode) {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GHL HTTP %d: %s", resp.StatusCode, truncate(string(b), 160))
	}

	var env funnelListEnvelope
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		return nil, err
	}
	return env.pick(), nil
}

// resolveFunnelDomain mirrors the prototype: direct fields → domainId lookup →
// vault meta fallback. domainCache dedupes domainId lookups within one request.
func resolveFunnelDomain(ctx context.Context, client *http.Client, token, locationID string, f rawFunnel, metaDomain string, domainCache map[string]string) string {
	if d := f.directDomain(); d != "" {
		return d
	}
	if f.DomainID != "" {
		if cached, ok := domainCache[f.DomainID]; ok {
			return cached
		}
		d := lookupDomain(ctx, client, token, locationID, f.DomainID)
		if d != "" {
			domainCache[f.DomainID] = d
			return d
		}
	}
	return metaDomain
}

func lookupDomain(ctx context.Context, client *http.Client, token, locationID, domainID string) string {
	base := store.GHLBase()
	for _, path := range []string{
		fmt.Sprintf("/locations/%s/domains/%s", url.PathEscape(locationID), url.PathEscape(domainID)),
		fmt.Sprintf("/domains/%s", url.PathEscape(domainID)),
	} {
		target := base + path
		if !strings.HasPrefix(target, base) {
			continue
		}
		// #nosec G107 G704 -- base host hardcoded + validated; only escaped IDs vary
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
		if err != nil {
			continue
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Version", "2021-07-28")

		// #nosec G107 G704 -- see above
		resp, err := client.Do(req)
		if err != nil {
			continue
		}
		var data struct {
			Domain string `json:"domain"`
			URL    string `json:"url"`
			Name   string `json:"name"`
		}
		derr := json.NewDecoder(resp.Body).Decode(&data)
		if cerr := resp.Body.Close(); cerr != nil {
			log.Printf("domain lookup: body close error: %v", cerr)
		}
		if !isOK(resp.StatusCode) || derr != nil {
			continue
		}
		for _, d := range []string{data.Domain, data.URL, data.Name} {
			if c := cleanDomain(d); c != "" {
				return c
			}
		}
	}
	return ""
}

// fetchPageHTML fetches a live funnel page server-side with SSRF guards.
func fetchPageHTML(ctx context.Context, client *http.Client, pageURL string) (string, error) {
	u, err := url.Parse(pageURL)
	if err != nil {
		return "", err
	}
	if !strings.EqualFold(u.Scheme, "https") {
		return "", fmt.Errorf("non-https url")
	}
	if err := guardPublicHost(u.Hostname()); err != nil {
		return "", err
	}

	// #nosec G107 -- host validated public+non-loopback via guardPublicHost; https enforced; cross-host redirects blocked by client.CheckRedirect
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pageURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "NNN-GHL-Manager/funnel-audit")

	// #nosec G107 -- see above
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			log.Printf("page fetch: body close error: %v", cerr)
		}
	}()
	if !isOK(resp.StatusCode) {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, maxPageBytes))
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// guardPublicHost rejects loopback/private/link-local/internal hosts so the
// server-side page fetch cannot be steered at internal infrastructure (SSRF).
func guardPublicHost(host string) error {
	host = strings.TrimSpace(strings.ToLower(host))
	if host == "" {
		return fmt.Errorf("empty host")
	}
	if host == "localhost" || strings.HasSuffix(host, ".local") || strings.HasSuffix(host, ".internal") {
		return fmt.Errorf("blocked host %q", host)
	}
	if ip := net.ParseIP(host); ip != nil {
		if !isPublicIP(ip) {
			return fmt.Errorf("non-public ip %s", ip)
		}
		return nil
	}
	ips, err := net.LookupIP(host)
	if err != nil {
		return fmt.Errorf("dns lookup failed for %s", host)
	}
	for _, ip := range ips {
		if !isPublicIP(ip) {
			return fmt.Errorf("host %s resolves to non-public ip %s", host, ip)
		}
	}
	return nil
}

func isPublicIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsUnspecified() || ip.IsMulticast() {
		return false
	}
	return true
}

var reFbqInit = regexp.MustCompile(`fbq\(\s*['"]init['"]\s*,\s*['"]?(\d+)['"]?`)

// auditHTML detects the Meta pixel, UTM tracking script, and Acuity footer in a
// page's HTML, mirroring the prototype's extractTrackingScanDetails (HTML L2863).
func auditHTML(html string) (hasPixel bool, pixelID string, hasUTM, hasAcuity bool) {
	hasPixel = strings.Contains(html, "fbq(") ||
		strings.Contains(html, "fbevents.js") ||
		strings.Contains(html, "Meta Pixel")
	if m := reFbqInit.FindStringSubmatch(html); m != nil {
		pixelID = m[1]
	}
	hasUTM = strings.Contains(html, "user_ip_address") ||
		strings.Contains(html, "full_url_utm") ||
		strings.Contains(html, "buildfbc") ||
		strings.Contains(html, "api.ipify.org") ||
		strings.Contains(html, "fbclid")
	hasAcuity = strings.Contains(html, "ACUITY_FIELD_ID") ||
		strings.Contains(html, "acuityscheduling.com") ||
		strings.Contains(html, "MARKETING DATA TRACKER")
	return hasPixel, pixelID, hasUTM, hasAcuity
}

// buildStepURL mirrors the prototype's buildStepUrl (HTML L2816).
func buildStepURL(slug, domain string) string {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return ""
	}
	low := strings.ToLower(slug)
	if strings.HasPrefix(low, "http://") || strings.HasPrefix(low, "https://") {
		return slug
	}
	d := cleanDomain(domain)
	if d == "" {
		return ""
	}
	slug = strings.TrimPrefix(slug, "/")
	if slug == "" {
		return "https://" + d
	}
	return "https://" + d + "/" + slug
}

// cleanDomain strips scheme, path and trailing slash, leaving a bare host.
func cleanDomain(domain string) string {
	d := strings.TrimSpace(domain)
	d = strings.TrimPrefix(d, "https://")
	d = strings.TrimPrefix(d, "http://")
	d = strings.TrimSuffix(d, "/")
	if i := strings.IndexByte(d, '/'); i >= 0 {
		d = d[:i]
	}
	return d
}
