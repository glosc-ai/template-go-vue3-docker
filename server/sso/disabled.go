package sso

import "net/http"

// RegisterDisabled keeps the auth routes answering in a predictable, typed way
// when no SSO credentials are configured. Without it these paths would fall
// through to the SPA handler's generic 404 and the frontend could not tell
// "not signed in" apart from "login is not available in this deployment".
func RegisterDisabled(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/auth/session", func(w http.ResponseWriter, _ *http.Request) {
		writeError(w, http.StatusUnauthorized, "unauthenticated", "sign in required")
	})
	mux.HandleFunc("POST /api/v1/auth/logout", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"data": map[string]any{"sso_logout_url": ""}})
	})

	unavailable := func(w http.ResponseWriter, _ *http.Request) {
		writeError(w, http.StatusServiceUnavailable, "sso_disabled",
			"single sign-on is not configured on this server")
	}
	mux.HandleFunc("GET /api/v1/auth/sso/login", unavailable)
	mux.HandleFunc("GET /api/v1/auth/sso/callback", unavailable)
}
