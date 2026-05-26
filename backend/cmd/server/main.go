package main

import (
	"log"
	"net/http"

	"github.com/ipagoagaNNN/nnn-ghl-manager/backend/internal/handlers"
	"github.com/ipagoagaNNN/nnn-ghl-manager/backend/internal/middleware"
	"github.com/ipagoagaNNN/nnn-ghl-manager/backend/internal/proxy"
	"github.com/ipagoagaNNN/nnn-ghl-manager/backend/internal/store"
)

func main() {
	vault := store.NewVault()
	ghlProxy := proxy.New(vault)

	mux := http.NewServeMux()

	// Auth routes (no-op middleware until Phase 2)
	mux.HandleFunc("POST /api/connect", handlers.Connect(vault))
	mux.HandleFunc("POST /api/tokens/{locationId}", handlers.SaveToken(vault))

	// Module routes
	mux.HandleFunc("GET /api/accounts", handlers.ListAccounts(vault))
	mux.HandleFunc("GET /api/cv", handlers.ListCV(vault))
	mux.HandleFunc("POST /api/cv/bulk", handlers.BulkUpdateCV(vault))
	mux.HandleFunc("GET /api/workflows/{locationId}", handlers.ListWorkflows(vault))
	mux.HandleFunc("GET /api/funnels/{locationId}", handlers.ListFunnels(vault))
	mux.HandleFunc("PUT /api/funnels/page/{pageId}", handlers.UpdateFunnelPage(vault))
	mux.HandleFunc("GET /api/dashboard/{locationId}/contacts", handlers.DashboardContacts(vault))
	mux.HandleFunc("POST /api/numbers/library", handlers.UpdateNumbersLibrary(vault))

	// GHL proxy — all /api/ghl/* requests forwarded with server-side token injection
	mux.Handle("/api/ghl/", ghlProxy)

	h := middleware.Chain(mux,
		middleware.CORS,
		middleware.RateLimit,
		middleware.Auth, // no-op until Phase 2
	)

	log.Println("NNN-GHL-Manager backend listening on :8080")
	if err := http.ListenAndServe(":8080", h); err != nil {
		log.Fatal(err)
	}
}
