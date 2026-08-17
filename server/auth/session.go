package auth

import (
	"net/http"
	"time"
)

// SessionCookieName holds the application's own session JWT. The token is kept
// in an HttpOnly cookie so frontend JavaScript cannot read it.
const SessionCookieName = "session"

// SetSession writes the session cookie. secure should be true whenever the app
// is served over HTTPS; it is disabled for plain-HTTP local development
// because browsers reject Secure cookies on http://localhost origins in some
// configurations.
func SetSession(w http.ResponseWriter, token string, ttl time.Duration, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   int(ttl.Seconds()),
		HttpOnly: true,
		Secure:   secure,
		// Lax still sends the cookie on the top-level GET redirect back from
		// the SSO provider, which Strict would block.
		SameSite: http.SameSiteLaxMode,
	})
}

// ClearSession expires the session cookie.
func ClearSession(w http.ResponseWriter, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

// SessionToken reads the raw session token from the request, preferring the
// cookie and falling back to a bearer header for non-browser clients.
func SessionToken(r *http.Request) string {
	if cookie, err := r.Cookie(SessionCookieName); err == nil && cookie.Value != "" {
		return cookie.Value
	}
	header := r.Header.Get("Authorization")
	if len(header) > 7 && (header[:7] == "Bearer " || header[:7] == "bearer ") {
		return header[7:]
	}
	return ""
}
