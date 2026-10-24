package db

import (
	"database/sql"
	"fmt"
	"os"
	"testing"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

func setupTestDB(t *testing.T) {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Fatal("TEST_DATABASE_URL must point to a separate test database")
	}

	database, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}

	if err := database.Ping(); err != nil {
		database.Close()
		t.Fatal(err)
	}

	Database = database

	t.Cleanup(func() {
		Database = nil
		database.Close()
	})

	_, err = Database.Exec(`
		CREATE TABLE IF NOT EXISTS items (
			id UUID PRIMARY KEY,
			name TEXT NOT NULL,
			count INTEGER NOT NULL,
			price INTEGER NOT NULL
		)
	`)
	if err != nil {
		t.Fatal(err)
	}

	_, err = Database.Exec("TRUNCATE TABLE items")
	if err != nil {
		t.Fatal(err)
	}
}

func TestAddItem(t *testing.T) {
	setupTestDB(t)
	clearItems(t)
	item := Item{
		ID:    uuid.New(),
		Name:  "Apple",
		Count: 10,
		Price: 100,
	}

	if err := AddItem(item); err != nil {
		t.Fatalf("AddItem() error: %v", err)
	}

	got, err := GetItem(item.ID)
	if err != nil {
		t.Fatalf("GetItem() error: %v", err)
	}

	if got != item {
		t.Errorf("got %+v, want %+v", got, item)
	}
}

func TestGetItem(t *testing.T) {

	setupTestDB(t)
	clearItems(t)
	item := Item{
		ID:    uuid.New(),
		Name:  "Book",
		Count: 3,
		Price: 500,
	}

	if err := AddItem(item); err != nil {
		t.Fatal(err)
	}

	got, err := GetItem(item.ID)
	if err != nil {
		t.Fatalf("GetItem() error: %v", err)
	}

	if got != item {
		t.Errorf("got %+v, want %+v", got, item)
	}
}

func TestGetItems(t *testing.T) {
	setupTestDB(t)
	clearItems(t)
	expected := []Item{
		{ID: uuid.New(), Name: "Apple", Count: 10, Price: 100},
		{ID: uuid.New(), Name: "Book", Count: 3, Price: 500},
	}

	for _, item := range expected {
		if err := AddItem(item); err != nil {
			t.Fatal(err)
		}
	}

	got, err := GetItems()
	if err != nil {
		t.Fatalf("GetItems() error: %v", err)
	}

	if len(got) != len(expected) {
		t.Fatalf("got %d items, want %d", len(got), len(expected))
	}

	for i, item := range expected {
		if got[i] != item {
			t.Errorf("item[%d] = %+v, want %+v", i, got[i], item)
		}
	}
}

func TestDeleteItem(t *testing.T) {
	setupTestDB(t)
	clearItems(t)
	item := Item{
		ID:    uuid.New(),
		Name:  "Pen",
		Count: 5,
		Price: 20,
	}

	if err := AddItem(item); err != nil {
		t.Fatal(err)
	}

	if err := DeleteItem(item.ID); err != nil {
		t.Fatalf("DeleteItem() error: %v", err)
	}

	_, err := GetItem(item.ID)
	if err != sql.ErrNoRows {
		t.Errorf("GetItem() error = %v, want sql.ErrNoRows", err)
	}
}

func TestGetItemNotFound(t *testing.T) {
	setupTestDB(t)
	clearItems(t)
	_, err := GetItem(uuid.New())
	if err != sql.ErrNoRows {
		t.Errorf("GetItem() error = %v, want sql.ErrNoRows", err)
	}
}

func clearItems(t *testing.T) {
	t.Helper()

	_, err := Database.Exec("TRUNCATE TABLE items")
	if err != nil {
		t.Fatal(err)
	}
}

var _ = fmt.Sprintf
