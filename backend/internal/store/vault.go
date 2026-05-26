package store

import (
	"crypto/rand"
	"sync"
)

const ghlBase = "https://services.leadconnectorhq.com"

// Vault holds GHL tokens server-side. Tokens never leave the server.
// Phase 2 will replace the in-memory map with an encrypted persistent store.
type Vault struct {
	mu          sync.RWMutex
	agencyToken string
	companyID   string
	locTokens   map[string]string // locationId → token
	locMeta     map[string]LocMeta
}

type LocMeta struct {
	Name        string
	Domain      string
	AcuityField string
	CalendarIDs string
	Active      bool
}

func NewVault() *Vault {
	b := make([]byte, 32)
	rand.Read(b) // future: use this as encryption key
	return &Vault{
		locTokens: make(map[string]string),
		locMeta:   make(map[string]LocMeta),
	}
}

func (v *Vault) SetAgency(token, companyID string) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.agencyToken = token
	v.companyID = companyID
}

func (v *Vault) AgencyToken() string {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.agencyToken
}

func (v *Vault) CompanyID() string {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.companyID
}

func (v *Vault) SetLocToken(locationID, token string) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.locTokens[locationID] = token
}

func (v *Vault) LocToken(locationID string) (string, bool) {
	v.mu.RLock()
	defer v.mu.RUnlock()
	tok, ok := v.locTokens[locationID]
	return tok, ok
}

func (v *Vault) AllLocTokens() map[string]string {
	v.mu.RLock()
	defer v.mu.RUnlock()
	out := make(map[string]string, len(v.locTokens))
	for k, tok := range v.locTokens {
		out[k] = tok
	}
	return out
}

func (v *Vault) GHLBase() string { return ghlBase }
