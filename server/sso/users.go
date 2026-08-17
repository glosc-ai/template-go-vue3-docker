package sso

import "time"

// User is a local account linked to an SSO identity. Subject is the provider's
// `sub` claim and is the stable join key — name, email and phone may change.
type User struct {
	ID        int64     `json:"id"`
	Subject   string    `json:"subject"`
	Name      string    `json:"name"`
	Nickname  string    `json:"nickname"`
	Email     string    `json:"email"`
	Phone     string    `json:"phone"`
	Avatar    string    `json:"avatar"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	LastLogin time.Time `json:"last_login_at"`
}
