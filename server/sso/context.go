package sso

import "context"

// contextKey is unexported so no other package can collide with this key.
type contextKey struct{}

// WithUser attaches the authenticated user to the request context.
func WithUser(ctx context.Context, user User) context.Context {
	return context.WithValue(ctx, contextKey{}, user)
}

// UserFrom returns the authenticated user attached by RequireUser.
func UserFrom(ctx context.Context) (User, bool) {
	user, ok := ctx.Value(contextKey{}).(User)
	return user, ok
}
