package main

import (
	"log"
	"net/http"
	"time"

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
	mux.HandleFunc("POST /api/connect", handlers.Connect(vault))                  // agency-level PIT → list sub-accounts
	mux.HandleFunc("POST /api/connect/location", handlers.ConnectLocation(vault)) // location-scoped PIT → single sub-account
	mux.HandleFunc("POST /api/tokens/{locationId}", handlers.SaveToken(vault))

	// Module routes
	mux.HandleFunc("GET /api/accounts", handlers.ListAccounts(vault))
	mux.HandleFunc("GET /api/accounts/library", handlers.ListLibrary(vault))
	mux.HandleFunc("PUT /api/accounts/{locationId}/meta", handlers.UpdateLocMeta(vault))
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

	srv := &http.Server{
		Addr:              ":8091",
		Handler:           h,
		ReadTimeout:       15 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      35 * time.Second, // 30s upstream + 5s margin
		IdleTimeout:       60 * time.Second,
	}

	log.Println("NNN-GHL-Manager backend listening on :8091 (8080+8090+9090 reserved by Docker on Iker's machine)")
	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
