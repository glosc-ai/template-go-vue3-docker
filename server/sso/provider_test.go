package sso

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

// fakeSSO stands in for sso.gloscai.com so the flow can be exercised without
// network access or real credentials.
func fakeSSO(t *testing.T) (*httptest.Server, *int) {
	t.Helper()
	discoveryCalls := 0

	mux := http.NewServeMux()
	var base string

	mux.HandleFunc("/api/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		discoveryCalls++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"issuer": "` + base + `",
			"authorization_endpoint": "` + base + `/api/oauth/authorize",
			"token_endpoint": "` + base + `/api/oauth/token",
			"userinfo_endpoint": "` + base + `/api/oauth/userinfo",
			"end_session_endpoint": "` + base + `/api/oauth/logout"
		}`))
	})

	mux.HandleFunc("POST /api/oauth/token", func(w http.ResponseWriter, r *http.Request) {
		clientID, clientSecret, ok := r.BasicAuth()
		if !ok || clientID != "test-client" || clientSecret != "test-secret" {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"invalid_client"}`))
			return
		}
		if err := r.ParseForm(); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if r.Form.Get("grant_type") != "authorization_code" || r.Form.Get("code") != "good-code" {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"invalid_grant","error_description":"bad code"}`))
			return
		}
		if r.Form.Get("code_verifier") == "" {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"invalid_request","error_description":"missing code_verifier"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"access-123","token_type":"Bearer","expires_in":3600}`))
	})

	mux.HandleFunc("GET /api/oauth/userinfo", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer access-123" {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"invalid_token"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"sub":"user-9","name":"xiaom","nickname":"小莫","email":"a@b.com"}`))
	})

	server := httptest.NewServer(mux)
	base = server.URL
	t.Cleanup(server.Close)
	return server, &discoveryCalls
}

func newTestProvider(server *httptest.Server) *Provider {
	return NewProvider(
		server.URL+"/api/.well-known/openid-configuration",
		"test-client",
		"test-secret",
		"https://app.example.com/api/v1/auth/sso/callback",
		[]string{"user:read"},
	)
}

func TestAuthorizationURLCarriesPKCEAndState(t *testing.T) {
	t.Parallel()
	server, _ := fakeSSO(t)
	provider := newTestProvider(server)

	raw, err := provider.AuthorizationURL(context.Background(), "state-abc", codeChallenge("verifier-xyz"))
	if err != nil {
		t.Fatalf("AuthorizationURL returned an error: %v", err)
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("AuthorizationURL produced an unparsable URL: %v", err)
	}

	query := parsed.Query()
	want := map[string]string{
		"response_type":         "code",
		"client_id":             "test-client",
		"redirect_uri":          "https://app.example.com/api/v1/auth/sso/callback",
		"scope":                 "user:read",
		"state":                 "state-abc",
		"code_challenge":        codeChallenge("verifier-xyz"),
		"code_challenge_method": "S256",
	}
	for key, value := range want {
		if got := query.Get(key); got != value {
			t.Errorf("authorize query %q = %q, want %q", key, got, value)
		}
	}
}

func TestExchangeAndUserInfo(t *testing.T) {
	t.Parallel()
	server, _ := fakeSSO(t)
	provider := newTestProvider(server)
	ctx := context.Background()

	token, err := provider.Exchange(ctx, "good-code", "verifier-xyz")
	if err != nil {
		t.Fatalf("Exchange returned an error: %v", err)
	}
	if token.AccessToken != "access-123" {
		t.Fatalf("Exchange access token = %q, want access-123", token.AccessToken)
	}

	info, err := provider.UserInfo(ctx, token.AccessToken)
	if err != nil {
		t.Fatalf("UserInfo returned an error: %v", err)
	}
	if info.Subject != "user-9" || info.Nickname != "小莫" {
		t.Fatalf("UserInfo = %+v, want sub user-9 and nickname 小莫", info)
	}
}

func TestExchangeRejectsBadCode(t *testing.T) {
	t.Parallel()
	server, _ := fakeSSO(t)
	provider := newTestProvider(server)

	if _, err := provider.Exchange(context.Background(), "stolen-code", "verifier-xyz"); err == nil {
		t.Fatal("Exchange should fail for an unknown authorization code")
	}
}

func TestUserInfoRejectsBadToken(t *testing.T) {
	t.Parallel()
	server, _ := fakeSSO(t)
	provider := newTestProvider(server)

	if _, err := provider.UserInfo(context.Background(), "not-a-token"); err == nil {
		t.Fatal("UserInfo should fail for an invalid access token")
	}
}

func TestMetadataIsCached(t *testing.T) {
	t.Parallel()
	server, calls := fakeSSO(t)
	provider := newTestProvider(server)
	ctx := context.Background()

	for range 3 {
		if _, err := provider.Metadata(ctx); err != nil {
			t.Fatalf("Metadata returned an error: %v", err)
		}
	}
	if *calls != 1 {
		t.Fatalf("discovery was fetched %d times, want 1", *calls)
	}
}

func TestMetadataRejectsIncompleteDocument(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"issuer":"https://sso.example.com"}`))
	}))
	t.Cleanup(server.Close)

	provider := NewProvider(server.URL, "id", "secret", "https://app.example.com/cb", []string{"user:read"})
	if _, err := provider.Metadata(context.Background()); err == nil {
		t.Fatal("Metadata should reject a document without the required endpoints")
	}
}

func TestUserInfoAcceptsDataEnvelope(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	var base string
	mux.HandleFunc("/discovery", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{
			"issuer": "` + base + `",
			"authorization_endpoint": "` + base + `/authorize",
			"token_endpoint": "` + base + `/token",
			"userinfo_endpoint": "` + base + `/userinfo"
		}`))
	})
	mux.HandleFunc("/userinfo", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"data":{"sub":"wrapped-1","name":"enveloped"}}`))
	})
	server := httptest.NewServer(mux)
	base = server.URL
	t.Cleanup(server.Close)

	provider := NewProvider(server.URL+"/discovery", "id", "secret", "https://app.example.com/cb", []string{"user:read"})
	info, err := provider.UserInfo(context.Background(), "any")
	if err != nil {
		t.Fatalf("UserInfo returned an error: %v", err)
	}
	if info.Subject != "wrapped-1" {
		t.Fatalf("UserInfo sub = %q, want wrapped-1", info.Subject)
	}
}
