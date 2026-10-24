package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"hls-hw-01/internal/db"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

func setupTestDB(t *testing.T) {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Fatal("TEST_DATABASE_URL is not set")
	}

	database, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}

	if err := database.Ping(); err != nil {
		database.Close()
		t.Fatal(err)
	}

	db.Database = database

	_, err = database.Exec(`
		CREATE TABLE IF NOT EXISTS items (
			id UUID PRIMARY KEY,
			name TEXT NOT NULL,
			count INTEGER NOT NULL,
			price INTEGER NOT NULL
		)
	`)
	if err != nil {
		database.Close()
		t.Fatal(err)
	}

	_, err = database.Exec("TRUNCATE TABLE items")
	if err != nil {
		database.Close()
		t.Fatal(err)
	}

	t.Cleanup(func() {
		db.Database = nil
		database.Close()
	})
}

func TestGetItems(t *testing.T) {
	setupTestDB(t)

	item := db.Item{
		ID:    uuid.New(),
		Name:  "Apple",
		Count: 10,
		Price: 100,
	}
	if err := db.AddItem(item); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/items", nil)
	rec := httptest.NewRecorder()

	GetItems(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("got status %d, want %d", rec.Code, http.StatusOK)
	}

	var resp GetItemsResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}

	if len(resp.Items) != 1 {
		t.Fatalf("got %d items, want 1", len(resp.Items))
	}

	got := resp.Items[0]
	if got.ID != item.ID || got.Name != item.Name ||
		got.Count != item.Count || got.Price != item.Price {
		t.Errorf("got %+v, want %+v", got, item)
	}
}

func TestGetItem(t *testing.T) {
	setupTestDB(t)

	item := db.Item{
		ID:    uuid.New(),
		Name:  "Book",
		Count: 3,
		Price: 500,
	}
	if err := db.AddItem(item); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(
		http.MethodGet,
		"/item?id="+item.ID.String(),
		nil,
	)
	rec := httptest.NewRecorder()

	GetItem(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("got status %d, want %d", rec.Code, http.StatusOK)
	}

	var resp GetItemResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}

	if resp.ID != item.ID || resp.Name != item.Name ||
		resp.Count != item.Count || resp.Price != item.Price {
		t.Errorf("got %+v, want %+v", resp, item)
	}
}

func TestGetItemInvalidID(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/item?id=invalid", nil)
	rec := httptest.NewRecorder()

	GetItem(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("got status %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestPutItem(t *testing.T) {
	setupTestDB(t)

	body := []byte(`{"name":"Apple","count":10,"price":100}`)
	req := httptest.NewRequest(
		http.MethodPost,
		"/item",
		bytes.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	PutItem(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("got status %d, want %d", rec.Code, http.StatusCreated)
	}

	var resp PostCreateItemResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}

	if resp.ID == uuid.Nil {
		t.Fatal("got empty UUID")
	}

	got, err := db.GetItem(resp.ID)
	if err != nil {
		t.Fatal(err)
	}

	if got.Name != "Apple" || got.Count != 10 || got.Price != 100 {
		t.Errorf("unexpected item in database: %+v", got)
	}
}

func TestPutItemInvalidJSON(t *testing.T) {
	req := httptest.NewRequest(
		http.MethodPost,
		"/item",
		bytes.NewBufferString(`{"name":`),
	)
	rec := httptest.NewRecorder()

	PutItem(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("got status %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestDeleteItem(t *testing.T) {
	setupTestDB(t)

	item := db.Item{
		ID:    uuid.New(),
		Name:  "Pen",
		Count: 5,
		Price: 20,
	}
	if err := db.AddItem(item); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(
		http.MethodDelete,
		"/item?id="+item.ID.String(),
		nil,
	)
	rec := httptest.NewRecorder()

	DeleteItem(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("got status %d, want %d", rec.Code, http.StatusNoContent)
	}

	_, err := db.GetItem(item.ID)
	if err != sql.ErrNoRows {
		t.Errorf("got error %v, want sql.ErrNoRows", err)
	}
}

func TestDeleteItemInvalidID(t *testing.T) {
	req := httptest.NewRequest(http.MethodDelete, "/item?id=invalid", nil)
	rec := httptest.NewRecorder()

	DeleteItem(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("got status %d, want %d", rec.Code, http.StatusBadRequest)
	}
}
