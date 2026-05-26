package store

// GHLBase returns the GoHighLevel API base URL.
// Used by handlers that need to make direct GHL calls (not through the proxy).
func GHLBase() string { return ghlBase }
