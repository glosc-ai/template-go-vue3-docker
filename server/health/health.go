package health

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
)

const pingTimeout = 3 * time.Second

type Handler struct {
	db    *sql.DB
	cache *redis.Client
}

func New(db *sql.DB, cache *redis.Client) *Handler {
	return &Handler{db: db, cache: cache}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /health/live", h.live)
	mux.HandleFunc("GET /health/ready", h.ready)
}

func (h *Handler) live(w http.ResponseWriter, _ *http.Request) {
	write(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (h *Handler) ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), pingTimeout)
	defer cancel()

	checks := map[string]string{"database": "ok", "redis": "ok"}
	status := http.StatusOK
	if err := h.db.PingContext(ctx); err != nil {
		checks["database"] = "unavailable"
		status = http.StatusServiceUnavailable
	}
	if err := h.cache.Ping(ctx).Err(); err != nil {
		checks["redis"] = "unavailable"
		status = http.StatusServiceUnavailable
	}
	state := "ok"
	if status != http.StatusOK {
		state = "degraded"
	}
	write(w, status, map[string]any{"status": state, "checks": checks})
}

func write(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
