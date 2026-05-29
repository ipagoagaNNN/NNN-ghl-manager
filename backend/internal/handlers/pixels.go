package handlers

import "strings"

// expectedPixels maps a domain substring to its expected Meta Pixel ID.
// Sourced verbatim from the prototype's PIXEL_SCRIPTS map
// (ghl-manager-final.2026-04-24-123140.html line 6118). The substring match
// mirrors the prototype's getPixelForDomain (domain.includes(key)).
//
// This is config, not logic — when a new brand/domain is onboarded, add a row
// here (or, later, move to a persisted per-agency config alongside the vault).
var expectedPixels = []struct {
	DomainKey string
	PixelID   string
	Brand     string
}{
	{DomainKey: "firstouchbeauty.com", PixelID: "1180664739949233", Brand: "First Touch Beauty"},
	{DomainKey: "noneedleneeded.com", PixelID: "540031443835405", Brand: "No Needle Needed"},
	{DomainKey: "advancedbeautytreatments.com", PixelID: "819221296693583", Brand: "Adv Beauty Treatments"},
}

// expectedPixelForDomain returns the configured pixel ID + brand for a domain.
// known is false when no expected pixel is configured for the domain — callers
// then report pixel presence without a right/wrong verdict.
func expectedPixelForDomain(domain string) (pixelID, brand string, known bool) {
	d := strings.ToLower(strings.TrimSpace(domain))
	for _, p := range expectedPixels {
		if strings.Contains(d, p.DomainKey) {
			return p.PixelID, p.Brand, true
		}
	}
	return "", "", false
}
