package sso

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gloscai/template-go-vue3-docker/server/auth"
)

// Store is the persistence this package needs; main injects the SQL
// implementation.
type Store interface {
	Upsert(context.Context, UserInfo) (User, error)
	Lookup(context.Context, int64) (User, error)
}

// Sessions issues and validates the application's own session tokens.
type Sessions interface {
	Issue(subject string) (string, error)
	Parse(token string) (string, error)
}

type Handler struct {
	provider *Provider
	state    *StateStore
	store    Store
	sessions Sessions
	logger   *slog.Logger

	sessionTTL    time.Duration
	secureCookies bool
	// defaultRedirect is where the callback sends the browser when the login
	// did not request a specific destination.
	defaultRedirect string
}

type Options struct {
	Provider        *Provider
	State           *StateStore
	Store           Store
	Sessions        Sessions
	Logger          *slog.Logger
	SessionTTL      time.Duration
	SecureCookies   bool
	DefaultRedirect string
}

func NewHandler(opts Options) *Handler {
	return &Handler{
		provider:        opts.Provider,
		state:           opts.State,
		store:           opts.Store,
		sessions:        opts.Sessions,
		logger:          opts.Logger,
		sessionTTL:      opts.SessionTTL,
		secureCookies:   opts.SecureCookies,
		defaultRedirect: cmpOr(opts.DefaultRedirect, "/"),
	}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/auth/sso/login", h.login)
	mux.HandleFunc("GET /api/v1/auth/sso/callback", h.callback)
	mux.HandleFunc("POST /api/v1/auth/logout", h.logout)
	mux.HandleFunc("GET /api/v1/auth/session", h.session)
}

// login starts the authorization code flow: mint state + PKCE verifier, park
// them in Redis, then redirect the browser to the SSO provider.
func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	state, err := randomToken()
	if err != nil {
		h.logger.Error("generating login state", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "could not start login")
		return
	}
	verifier, err := randomToken()
	if err != nil {
		h.logger.Error("generating PKCE verifier", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "could not start login")
		return
	}

	redirectTo := safeRedirect(r.URL.Query().Get("redirect_to"), h.defaultRedirect)
	if err := h.state.Save(r.Context(), state, verifier, redirectTo); err != nil {
		h.logger.Error("saving login state", "error", err)
		writeError(w, http.StatusServiceUnavailable, "sso_unavailable", "could not start login")
		return
	}

	target, err := h.provider.AuthorizationURL(r.Context(), state, codeChallenge(verifier))
	if err != nil {
		h.logger.Error("building authorization URL", "error", err)
		writeError(w, http.StatusServiceUnavailable, "sso_unavailable", "identity provider is unavailable")
		return
	}

	http.Redirect(w, r, target, http.StatusFound)
}

// callback validates state, exchanges the code, links the identity locally and
// issues the application session. It always redirects — this endpoint is
// reached by a browser navigation, not by fetch().
func (h *Handler) callback(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	if providerError := query.Get("error"); providerError != "" {
		h.logger.Warn("SSO returned an error",
			"error", providerError,
			"description", query.Get("error_description"))
		h.redirectFailure(w, r, providerError)
		return
	}

	state := query.Get("state")
	code := query.Get("code")
	if state == "" || code == "" {
		h.redirectFailure(w, r, "invalid_response")
		return
	}

	pending, err := h.state.Consume(r.Context(), state)
	if errors.Is(err, ErrUnknownState) {
		// Expired, replayed, or forged — never proceed to the exchange.
		h.logger.Warn("rejected SSO callback with unknown state")
		h.redirectFailure(w, r, "expired_state")
		return
	}
	if err != nil {
		h.logger.Error("reading login state", "error", err)
		h.redirectFailure(w, r, "internal_error")
		return
	}

	token, err := h.provider.Exchange(r.Context(), code, pending.CodeVerifier)
	if err != nil {
		h.logger.Error("exchanging authorization code", "error", err)
		h.redirectFailure(w, r, "exchange_failed")
		return
	}

	info, err := h.provider.UserInfo(r.Context(), token.AccessToken)
	if err != nil {
		h.logger.Error("reading SSO user info", "error", err)
		h.redirectFailure(w, r, "userinfo_failed")
		return
	}

	user, err := h.store.Upsert(r.Context(), info)
	if err != nil {
		h.logger.Error("linking SSO user", "error", err)
		h.redirectFailure(w, r, "internal_error")
		return
	}

	sessionToken, err := h.sessions.Issue(strconv.FormatInt(user.ID, 10))
	if err != nil {
		h.logger.Error("issuing session", "error", err)
		h.redirectFailure(w, r, "internal_error")
		return
	}

	auth.SetSession(w, sessionToken, h.sessionTTL, h.secureCookies)
	h.logger.Info("SSO login succeeded", "user_id", user.ID)
	http.Redirect(w, r, safeRedirect(pending.RedirectTo, h.defaultRedirect), http.StatusFound)
}

// session returns the signed-in user, or 401 when there is no valid session.
func (h *Handler) session(w http.ResponseWriter, r *http.Request) {
	user, ok := h.currentUser(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthenticated", "sign in required")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": user})
}

// logout clears the local session. The SSO session itself is left alone unless
// the frontend follows up with sso_logout_url.
func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	auth.ClearSession(w, h.secureCookies)
	writeJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{"sso_logout_url": h.provider.LogoutURL(r.Context())},
	})
}

// currentUser resolves the session cookie to a stored user.
func (h *Handler) currentUser(r *http.Request) (User, bool) {
	token := auth.SessionToken(r)
	if token == "" {
		return User{}, false
	}
	subject, err := h.sessions.Parse(token)
	if err != nil {
		return User{}, false
	}
	id, err := userIDFromSubject(subject)
	if err != nil {
		return User{}, false
	}
	user, err := h.store.Lookup(r.Context(), id)
	if err != nil {
		if !errors.Is(err, ErrUserNotFound) {
			h.logger.Error("loading session user", "error", err)
		}
		return User{}, false
	}
	return user, true
}

// RequireUser wraps handlers that must not run without a session.
func (h *Handler) RequireUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := h.currentUser(r)
		if !ok {
			writeError(w, http.StatusUnauthorized, "unauthenticated", "sign in required")
			return
		}
		next.ServeHTTP(w, r.WithContext(WithUser(r.Context(), user)))
	})
}

// redirectFailure sends the browser back to the SPA with a machine-readable
// reason so the frontend can show a message instead of a blank page.
func (h *Handler) redirectFailure(w http.ResponseWriter, r *http.Request, reason string) {
	target := url.URL{Path: "/login", RawQuery: url.Values{"error": {reason}}.Encode()}
	http.Redirect(w, r, target.String(), http.StatusFound)
}

// safeRedirect only allows same-site absolute paths, so `redirect_to` cannot be
// abused as an open redirect to an attacker's host.
func safeRedirect(candidate, fallback string) string {
	if candidate == "" {
		return fallback
	}
	// Reject anything that is not a plain path: absolute URLs, scheme-relative
	// //evil.com, and backslash variants browsers may normalise.
	if !strings.HasPrefix(candidate, "/") || strings.HasPrefix(candidate, "//") || strings.Contains(candidate, "\\") {
		return fallback
	}
	parsed, err := url.Parse(candidate)
	if err != nil || parsed.Host != "" || parsed.Scheme != "" {
		return fallback
	}
	return parsed.RequestURI()
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{
		"error": map[string]string{"code": code, "message": message},
	})
}

func cmpOr(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
