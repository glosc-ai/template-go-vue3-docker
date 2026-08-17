package sso

import "testing"

func TestSafeRedirectAllowsSameSitePaths(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"/profile":           "/profile",
		"/tasks?filter=open": "/tasks?filter=open",
		"/a/b/c":             "/a/b/c",
		"/profile#section":   "/profile",
		"/search?q=a%20b":    "/search?q=a%20b",
	}
	for candidate, want := range cases {
		if got := safeRedirect(candidate, "/fallback"); got != want {
			t.Errorf("safeRedirect(%q) = %q, want %q", candidate, got, want)
		}
	}
}

func TestSafeRedirectRejectsOffSiteTargets(t *testing.T) {
	t.Parallel()

	// Every one of these must fall back: an attacker who controls redirect_to
	// must not be able to bounce a freshly authenticated user off-site.
	candidates := []string{
		"",
		"https://evil.example.com/steal",
		"//evil.example.com",
		"/\\evil.example.com",
		"\\/evil.example.com",
		"http://evil.example.com",
		"javascript:alert(1)",
		"profile",
		"../etc/passwd",
	}
	for _, candidate := range candidates {
		if got := safeRedirect(candidate, "/fallback"); got != "/fallback" {
			t.Errorf("safeRedirect(%q) = %q, want the fallback", candidate, got)
		}
	}
}

func TestCodeChallengeIsDeterministicS256(t *testing.T) {
	t.Parallel()

	// Vector from RFC 7636 appendix B.
	const verifier = "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	const want = "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"

	if got := codeChallenge(verifier); got != want {
		t.Errorf("codeChallenge(%q) = %q, want %q", verifier, got, want)
	}
}

func TestRandomTokenIsUniqueAndURLSafe(t *testing.T) {
	t.Parallel()

	seen := make(map[string]struct{}, 128)
	for range 128 {
		token, err := randomToken()
		if err != nil {
			t.Fatalf("randomToken() returned an error: %v", err)
		}
		if len(token) < 32 {
			t.Fatalf("randomToken() = %q, want at least 32 characters", token)
		}
		for _, r := range token {
			isSafe := r == '-' || r == '_' ||
				(r >= '0' && r <= '9') ||
				(r >= 'a' && r <= 'z') ||
				(r >= 'A' && r <= 'Z')
			if !isSafe {
				t.Fatalf("randomToken() = %q contains non URL-safe %q", token, r)
			}
		}
		if _, duplicate := seen[token]; duplicate {
			t.Fatalf("randomToken() produced a duplicate value %q", token)
		}
		seen[token] = struct{}{}
	}
}

func TestUserIDFromSubject(t *testing.T) {
	t.Parallel()

	if id, err := userIDFromSubject("42"); err != nil || id != 42 {
		t.Errorf("userIDFromSubject(\"42\") = (%d, %v), want (42, nil)", id, err)
	}
	for _, subject := range []string{"", "0", "-1", "abc", "1.5", " 7"} {
		if _, err := userIDFromSubject(subject); err == nil {
			t.Errorf("userIDFromSubject(%q) should have failed", subject)
		}
	}
}
