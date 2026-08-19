package health

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

// fakeDriver lets tests control whether DB.PingContext succeeds or fails
// without needing a real database connection.
type fakeDriver struct {
	pingErr error
}

func (d *fakeDriver) Open(string) (driver.Conn, error) {
	return &fakeConn{pingErr: d.pingErr}, nil
}

type fakeConn struct {
	pingErr error
}

func (c *fakeConn) Prepare(string) (driver.Stmt, error) { return nil, errors.New("not implemented") }
func (c *fakeConn) Close() error                        { return nil }
func (c *fakeConn) Begin() (driver.Tx, error)           { return nil, errors.New("not implemented") }
func (c *fakeConn) Ping(context.Context) error          { return c.pingErr }

func openFakeDB(t *testing.T, name string, pingErr error) *sql.DB {
	t.Helper()
	sql.Register(name, &fakeDriver{pingErr: pingErr})
	db, err := sql.Open(name, "")
	if err != nil {
		t.Fatalf("open fake db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func newTestRedis(t *testing.T) *redis.Client {
	t.Helper()
	server := miniredis.RunT(t)
	return redis.NewClient(&redis.Options{Addr: server.Addr()})
}

func TestLiveReturnsOK(t *testing.T) {
	h := New(openFakeDB(t, "fakehealth-live", nil), newTestRedis(t))
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/health/live", nil)

	h.live(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
}

func TestReadyAllHealthy(t *testing.T) {
	h := New(openFakeDB(t, "fakehealth-ready-ok", nil), newTestRedis(t))
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/health/ready", nil)

	h.ready(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	var body map[string]any
	if err := json.NewDecoder(recorder.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["status"] != "ok" {
		t.Fatalf("status field = %v, want ok", body["status"])
	}
}

func TestReadyDatabaseDown(t *testing.T) {
	h := New(openFakeDB(t, "fakehealth-ready-dbdown", errors.New("db unreachable")), newTestRedis(t))
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/health/ready", nil)

	h.ready(recorder, request)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusServiceUnavailable)
	}
	var body map[string]any
	if err := json.NewDecoder(recorder.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["status"] != "degraded" {
		t.Fatalf("status field = %v, want degraded", body["status"])
	}
	checks := body["checks"].(map[string]any)
	if checks["database"] != "unavailable" {
		t.Fatalf("database check = %v, want unavailable", checks["database"])
	}
	if checks["redis"] != "ok" {
		t.Fatalf("redis check = %v, want ok", checks["redis"])
	}
}

func TestReadyRedisDown(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	server.Close()

	h := New(openFakeDB(t, "fakehealth-ready-redisdown", nil), client)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/health/ready", nil)

	h.ready(recorder, request)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusServiceUnavailable)
	}
}

func TestReadyRespectsTimeout(t *testing.T) {
	h := New(openFakeDB(t, "fakehealth-ready-timeout", nil), newTestRedis(t))
	recorder := httptest.NewRecorder()
	ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond)
	defer cancel()
	request := httptest.NewRequest(http.MethodGet, "/health/ready", nil).WithContext(ctx)

	h.ready(recorder, request)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusServiceUnavailable)
	}
}
