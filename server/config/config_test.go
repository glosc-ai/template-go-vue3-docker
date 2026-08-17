package config

import "testing"

func TestLoadLeavesSSODisabledWithoutClientID(t *testing.T) {
	settings, err := loadSSO("development")
	if err != nil {
		t.Fatalf("loadSSO returned an error: %v", err)
	}
	if settings.Enabled {
		t.Error("SSO should stay disabled when SSO_CLIENT_ID is unset")
	}
	if settings.DiscoveryURL != "https://sso.gloscai.com/api/.well-known/openid-configuration" {
		t.Errorf("default discovery URL = %q", settings.DiscoveryURL)
	}
}

func TestLoadSSORequiresSecretAndRedirect(t *testing.T) {
	t.Setenv("SSO_CLIENT_ID", "client-1")

	if _, err := loadSSO("development"); err == nil {
		t.Fatal("loadSSO should fail without SSO_CLIENT_SECRET")
	}

	t.Setenv("SSO_CLIENT_SECRET", "secret-1")
	if _, err := loadSSO("development"); err == nil {
		t.Fatal("loadSSO should fail without SSO_REDIRECT_URL")
	}

	t.Setenv("SSO_REDIRECT_URL", "/api/v1/auth/sso/callback")
	if _, err := loadSSO("development"); err == nil {
		t.Fatal("loadSSO should reject a relative SSO_REDIRECT_URL")
	}

	t.Setenv("SSO_REDIRECT_URL", "http://localhost:5173/api/v1/auth/sso/callback")
	settings, err := loadSSO("development")
	if err != nil {
		t.Fatalf("loadSSO returned an error: %v", err)
	}
	if !settings.Enabled {
		t.Error("SSO should be enabled once client ID, secret and redirect URL are set")
	}
	if settings.SecureCookies {
		t.Error("development should default to non-Secure cookies for plain HTTP")
	}
	if settings.PostLoginPath != "/profile" {
		t.Errorf("PostLoginPath = %q, want /profile", settings.PostLoginPath)
	}
}

func TestLoadSSOSecuresCookiesInProduction(t *testing.T) {
	settings, err := loadSSO("production")
	if err != nil {
		t.Fatalf("loadSSO returned an error: %v", err)
	}
	if !settings.SecureCookies {
		t.Error("production should default to Secure session cookies")
	}
}

func TestLoadSSORejectsRelativePostLoginPath(t *testing.T) {
	t.Setenv("SSO_CLIENT_ID", "client-1")
	t.Setenv("SSO_CLIENT_SECRET", "secret-1")
	t.Setenv("SSO_REDIRECT_URL", "https://app.example.com/api/v1/auth/sso/callback")
	t.Setenv("SSO_POST_LOGIN_PATH", "profile")

	if _, err := loadSSO("production"); err == nil {
		t.Fatal("loadSSO should reject a post-login path that is not root-relative")
	}
}
