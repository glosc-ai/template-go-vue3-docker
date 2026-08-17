package sso

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// ErrUnknownState means the state was never issued, already consumed, or has
// expired. Treated as a failed login rather than an internal error.
var ErrUnknownState = errors.New("unknown or expired login state")

// pendingLogin is what we remember between the authorize redirect and the
// callback. It lives in Redis so the code cannot be replayed and so state
// survives across API replicas.
type pendingLogin struct {
	CodeVerifier string `json:"code_verifier"`
	RedirectTo   string `json:"redirect_to"`
}

// StateStore persists in-flight logins for the duration of the authorization
// round trip.
type StateStore struct {
	client *redis.Client
	ttl    time.Duration
}

func NewStateStore(client *redis.Client) *StateStore {
	return &StateStore{client: client, ttl: 10 * time.Minute}
}

func (s *StateStore) key(state string) string {
	return "sso:login:" + state
}

// Save stores the pending login under the opaque state value.
func (s *StateStore) Save(ctx context.Context, state, codeVerifier, redirectTo string) error {
	payload, err := json.Marshal(pendingLogin{CodeVerifier: codeVerifier, RedirectTo: redirectTo})
	if err != nil {
		return fmt.Errorf("encoding pending login: %w", err)
	}
	if err := s.client.Set(ctx, s.key(state), payload, s.ttl).Err(); err != nil {
		return fmt.Errorf("storing pending login: %w", err)
	}
	return nil
}

// Consume atomically fetches and deletes the pending login, so a given state
// (and therefore a given authorization code) can only be used once.
func (s *StateStore) Consume(ctx context.Context, state string) (pendingLogin, error) {
	payload, err := s.client.GetDel(ctx, s.key(state)).Bytes()
	if errors.Is(err, redis.Nil) {
		return pendingLogin{}, ErrUnknownState
	}
	if err != nil {
		return pendingLogin{}, fmt.Errorf("reading pending login: %w", err)
	}

	var pending pendingLogin
	if err := json.Unmarshal(payload, &pending); err != nil {
		return pendingLogin{}, fmt.Errorf("decoding pending login: %w", err)
	}
	return pending, nil
}

// randomToken returns a URL-safe random string suitable for state values and
// PKCE verifiers.
func randomToken() (string, error) {
	value := make([]byte, 32)
	if _, err := rand.Read(value); err != nil {
		return "", fmt.Errorf("generating random token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}

// codeChallenge derives the S256 PKCE challenge for a verifier.
func codeChallenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}
