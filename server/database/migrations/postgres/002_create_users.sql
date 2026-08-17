CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    sso_subject VARCHAR(255) NOT NULL UNIQUE,
    name VARCHAR(255),
    nickname VARCHAR(255),
    email VARCHAR(320),
    phone VARCHAR(32),
    avatar VARCHAR(1024),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_login_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- statement-breakpoint

CREATE INDEX idx_users_email ON users (email);
