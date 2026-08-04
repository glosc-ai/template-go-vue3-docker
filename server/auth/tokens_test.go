package auth

import (
	"testing"
	"time"
)

func TestManagerRoundTrip(t *testing.T) {
	manager, err := NewManager("a-development-secret-with-32-characters", "test", time.Hour)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	token, err := manager.Issue("user-123")
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}
	subject, err := manager.Parse(token)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if subject != "user-123" {
		t.Fatalf("Parse() subject = %q, want user-123", subject)
	}
}

func TestNewManagerValidation(t *testing.T) {
	tests := []struct {
		name   string
		secret string
		issuer string
		ttl    time.Duration
	}{
		{name: "short secret", secret: "short", issuer: "test", ttl: time.Hour},
		{name: "missing issuer", secret: "a-development-secret-with-32-characters", ttl: time.Hour},
		{name: "invalid ttl", secret: "a-development-secret-with-32-characters", issuer: "test"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := NewManager(tt.secret, tt.issuer, tt.ttl); err == nil {
				t.Fatal("NewManager() error = nil, want validation error")
			}
		})
	}
}
