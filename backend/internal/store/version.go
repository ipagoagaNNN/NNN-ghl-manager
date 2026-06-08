package store

import "strings"

// GHL's API v2 requires a dated Version header on every request, and the correct
// value varies by endpoint family. Sending the wrong one can change the response
// envelope or be rejected outright — the custom-objects API (/objects) explicitly
// rejects the legacy 2021-07-28. Centralise the mapping here so the generic proxy
// and every typed handler send the same, correct value.
const (
	// VersionDefault — locations/search, workflows, contacts, funnels, forms.
	VersionDefault = "2021-07-28"
	// VersionCustomValues — /customValues and /customFields (per the GHL reference
	// and the project's verified request set in docs/"api requests.md").
	VersionCustomValues = "2021-04-15"
	// VersionObjects — REQUIRED for the custom-objects API (/objects/...); it
	// rejects/ignores 2021-07-28. See docs/api-requests/get-objects-by-location.md.
	VersionObjects = "2023-02-21"
)

// GHLVersionFor returns the correct GHL Version header for an API path — the part
// after the host, e.g. "/locations/X/customValues/Y" or "/objects/?locationId=X".
func GHLVersionFor(path string) string {
	p := strings.ToLower(path)
	switch {
	case strings.HasPrefix(p, "/objects"):
		return VersionObjects
	case strings.Contains(p, "/customvalues") || strings.Contains(p, "/customfields"):
		return VersionCustomValues
	default:
		return VersionDefault
	}
}
