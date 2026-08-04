package tasks

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type fakeStore struct {
	items []Task
}

func (f *fakeStore) List(context.Context) ([]Task, error) {
	return f.items, nil
}

func (f *fakeStore) Create(_ context.Context, title string) (Task, error) {
	task := Task{ID: 1, Title: title, CreatedAt: time.Unix(0, 0).UTC()}
	f.items = append(f.items, task)
	return task, nil
}

func (f *fakeStore) SetCompleted(_ context.Context, id int64, completed bool) (Task, error) {
	for index := range f.items {
		if f.items[index].ID == id {
			f.items[index].Completed = completed
			return f.items[index], nil
		}
	}
	return Task{}, ErrNotFound
}

func (f *fakeStore) Delete(_ context.Context, id int64) error {
	for index := range f.items {
		if f.items[index].ID == id {
			f.items = append(f.items[:index], f.items[index+1:]...)
			return nil
		}
	}
	return ErrNotFound
}

func TestHandlerCreateTask(t *testing.T) {
	store := new(fakeStore)
	mux := http.NewServeMux()
	NewHandler(store).Register(mux)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", strings.NewReader(`{"title":"Ship starter"}`))

	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body = %s", recorder.Code, http.StatusCreated, recorder.Body.String())
	}
	if len(store.items) != 1 || store.items[0].Title != "Ship starter" {
		t.Fatalf("created items = %#v", store.items)
	}
}

func TestHandlerRejectsInvalidTask(t *testing.T) {
	mux := http.NewServeMux()
	NewHandler(new(fakeStore)).Register(mux)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", strings.NewReader(`{"title":" "}`))

	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnprocessableEntity)
	}
	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&body); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if body.Error.Code != "invalid_title" {
		t.Fatalf("error code = %q, want invalid_title", body.Error.Code)
	}
}

func TestHandlerUpdatesTask(t *testing.T) {
	store := &fakeStore{items: []Task{{ID: 7, Title: "Test API"}}}
	mux := http.NewServeMux()
	NewHandler(store).Register(mux)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPatch, "/api/v1/tasks/7", strings.NewReader(`{"completed":true}`))

	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if !store.items[0].Completed {
		t.Fatal("task was not marked complete")
	}
}
