// Package sso implements an OAuth 2.0 / OpenID Connect relying party for
// https://sso.gloscai.com. The browser only ever talks to this API: it is
// redirected to the authorization endpoint, comes back to the callback with a
// one-time code, and the code-for-token exchange plus the UserInfo call happen
// here so client_secret and SSO tokens never reach the frontend.
//
// After a successful callback the SSO identity is upserted into the local
// users table and the application issues its own session JWT — the SSO access
// token is deliberately not reused as a browser session.
package sso

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Metadata is the subset of the OpenID Connect discovery document this
// package uses.
type Metadata struct {
	Issuer                string `json:"issuer"`
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	UserInfoEndpoint      string `json:"userinfo_endpoint"`
	EndSessionEndpoint    string `json:"end_session_endpoint"`
	RevocationEndpoint    string `json:"revocation_endpoint"`
}

func (m Metadata) validate() error {
	for name, value := range map[string]string{
		"issuer":                 m.Issuer,
		"authorization_endpoint": m.AuthorizationEndpoint,
		"token_endpoint":         m.TokenEndpoint,
		"userinfo_endpoint":      m.UserInfoEndpoint,
	} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("discovery document is missing %s", name)
		}
	}
	return nil
}

// UserInfo mirrors the claims documented by the provider. Only sub is
// guaranteed to be present, so everything else is treated as optional.
type UserInfo struct {
	Subject  string `json:"sub"`
	Name     string `json:"name"`
	Nickname string `json:"nickname"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Avatar   string `json:"avatar"`
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
}

// Provider performs the server-side half of the authorization code flow and
// caches the discovery document.
type Provider struct {
	discoveryURL string
	clientID     string
	clientSecret string
	redirectURL  string
	scopes       []string
	client       *http.Client

	mu        sync.Mutex
	metadata  *Metadata
	fetchedAt time.Time
	cacheTTL  time.Duration
}

// NewProvider builds a Provider. Discovery is lazy: a slow or unavailable SSO
// service must not keep the API from starting.
func NewProvider(discoveryURL, clientID, clientSecret, redirectURL string, scopes []string) *Provider {
	return &Provider{
		discoveryURL: discoveryURL,
		clientID:     clientID,
		clientSecret: clientSecret,
		redirectURL:  redirectURL,
		scopes:       scopes,
		client:       &http.Client{Timeout: 10 * time.Second},
		cacheTTL:     15 * time.Minute,
	}
}

// Metadata returns the cached discovery document, fetching it when the cache
// is empty or stale.
func (p *Provider) Metadata(ctx context.Context) (Metadata, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.metadata != nil && time.Since(p.fetchedAt) < p.cacheTTL {
		return *p.metadata, nil
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, p.discoveryURL, nil)
	if err != nil {
		return Metadata{}, fmt.Errorf("building discovery request: %w", err)
	}
	request.Header.Set("Accept", "application/json")

	response, err := p.client.Do(request)
	if err != nil {
		return Metadata{}, fmt.Errorf("fetching discovery document: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return Metadata{}, fmt.Errorf("fetching discovery document: unexpected status %d", response.StatusCode)
	}

	var metadata Metadata
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&metadata); err != nil {
		return Metadata{}, fmt.Errorf("decoding discovery document: %w", err)
	}
	if err := metadata.validate(); err != nil {
		return Metadata{}, err
	}

	p.metadata = &metadata
	p.fetchedAt = time.Now()
	return metadata, nil
}

// AuthorizationURL builds the URL the browser is redirected to. The caller
// owns state and the PKCE verifier and must persist them until the callback.
func (p *Provider) AuthorizationURL(ctx context.Context, state, codeChallenge string) (string, error) {
	metadata, err := p.Metadata(ctx)
	if err != nil {
		return "", err
	}

	target, err := url.Parse(metadata.AuthorizationEndpoint)
	if err != nil {
		return "", fmt.Errorf("parsing authorization endpoint: %w", err)
	}

	query := target.Query()
	query.Set("response_type", "code")
	query.Set("client_id", p.clientID)
	query.Set("redirect_uri", p.redirectURL)
	query.Set("scope", strings.Join(p.scopes, " "))
	query.Set("state", state)
	query.Set("code_challenge", codeChallenge)
	query.Set("code_challenge_method", "S256")
	target.RawQuery = query.Encode()

	return target.String(), nil
}

// Exchange trades the one-time authorization code for an access token using
// client_secret_basic, the provider's recommended authentication method.
func (p *Provider) Exchange(ctx context.Context, code, codeVerifier string) (tokenResponse, error) {
	metadata, err := p.Metadata(ctx)
	if err != nil {
		return tokenResponse{}, err
	}

	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", p.redirectURL)
	form.Set("code_verifier", codeVerifier)

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, metadata.TokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return tokenResponse{}, fmt.Errorf("building token request: %w", err)
	}
	request.SetBasicAuth(url.QueryEscape(p.clientID), url.QueryEscape(p.clientSecret))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Accept", "application/json")

	response, err := p.client.Do(request)
	if err != nil {
		return tokenResponse{}, fmt.Errorf("calling token endpoint: %w", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return tokenResponse{}, fmt.Errorf("reading token response: %w", err)
	}
	if response.StatusCode != http.StatusOK {
		return tokenResponse{}, fmt.Errorf("token endpoint returned status %d: %s", response.StatusCode, oauthErrorText(body))
	}

	var token tokenResponse
	if err := json.Unmarshal(body, &token); err != nil {
		return tokenResponse{}, fmt.Errorf("decoding token response: %w", err)
	}
	if token.AccessToken == "" {
		return tokenResponse{}, fmt.Errorf("token response did not contain an access token")
	}
	return token, nil
}

// UserInfo reads the standard claims for the authenticated user.
func (p *Provider) UserInfo(ctx context.Context, accessToken string) (UserInfo, error) {
	metadata, err := p.Metadata(ctx)
	if err != nil {
		return UserInfo{}, err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, metadata.UserInfoEndpoint, nil)
	if err != nil {
		return UserInfo{}, fmt.Errorf("building userinfo request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+accessToken)
	request.Header.Set("Accept", "application/json")

	response, err := p.client.Do(request)
	if err != nil {
		return UserInfo{}, fmt.Errorf("calling userinfo endpoint: %w", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return UserInfo{}, fmt.Errorf("reading userinfo response: %w", err)
	}
	if response.StatusCode != http.StatusOK {
		return UserInfo{}, fmt.Errorf("userinfo endpoint returned status %d: %s", response.StatusCode, oauthErrorText(body))
	}

	// The provider wraps successful payloads in {"data": ...} on some
	// deployments and returns bare claims on others; accept both.
	var envelope struct {
		Data *UserInfo `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err == nil && envelope.Data != nil && envelope.Data.Subject != "" {
		return *envelope.Data, nil
	}

	var info UserInfo
	if err := json.Unmarshal(body, &info); err != nil {
		return UserInfo{}, fmt.Errorf("decoding userinfo response: %w", err)
	}
	if info.Subject == "" {
		return UserInfo{}, fmt.Errorf("userinfo response did not contain sub")
	}
	return info, nil
}

// LogoutURL returns the provider's end-session URL, or an empty string when
// the deployment does not advertise one.
func (p *Provider) LogoutURL(ctx context.Context) string {
	metadata, err := p.Metadata(ctx)
	if err != nil {
		return ""
	}
	return metadata.EndSessionEndpoint
}

// oauthErrorText extracts an OAuth error payload for logging without leaking
// an entire HTML error page into the logs.
func oauthErrorText(body []byte) string {
	var payload struct {
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
		Message          string `json:"message"`
	}
	if err := json.Unmarshal(body, &payload); err == nil {
		switch {
		case payload.Error != "" && payload.ErrorDescription != "":
			return payload.Error + ": " + payload.ErrorDescription
		case payload.Error != "":
			return payload.Error
		case payload.Message != "":
			return payload.Message
		}
	}
	if len(body) > 200 {
		body = body[:200]
	}
	return string(body)
}
