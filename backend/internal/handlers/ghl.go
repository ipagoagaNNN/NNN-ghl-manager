package handlers

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/ipagoagaNNN/nnn-ghl-manager/backend/internal/store"
)

// ghlRequest issues an authenticated GHL API request with the correct, per-endpoint
// Version header (resolved from the path via store.GHLVersionFor) and enforces the
// GHL host lock. It returns the response body and status code. The typed handlers
// use this so the Version header is never hardcoded per call; the generic proxy has
// its own equivalent path.
func ghlRequest(ctx context.Context, client *http.Client, method, token, target string, body io.Reader) ([]byte, int, error) {
	base := store.GHLBase()
	if !strings.HasPrefix(target, base) {
		return nil, 0, fmt.Errorf("invalid GHL target")
	}
	// Path after the host (drop any query) drives Version selection.
	path := strings.TrimPrefix(target, base)
	if i := strings.IndexByte(path, '?'); i >= 0 {
		path = path[:i]
	}

	// #nosec G107 G704 -- target host is store.GHLBase() (hardcoded) and validated above; only escaped path/query vary
	req, err := http.NewRequestWithContext(ctx, method, target, body)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Version", store.GHLVersionFor(path))
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// #nosec G107 G704 -- see above; host is locked to the GHL base
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			log.Printf("ghl request: body close error: %v", cerr)
		}
	}()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return b, resp.StatusCode, nil
}
