package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"hls-hw-01/internal/db"
	"log"
	"net/http"

	"github.com/google/uuid"
)

func GetItems(w http.ResponseWriter, r *http.Request) {
	items, err := db.GetItems()
	if err != nil {
		log.Println(err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	resp := GetItemsResponse{
		Items: make([]ItemResponse, 0, len(items)),
	}

	for _, item := range items {
		resp.Items = append(resp.Items, ItemResponse{
			ID:    item.ID,
			Name:  item.Name,
			Count: item.Count,
			Price: item.Price,
		})
	}

	writeJSON(w, http.StatusOK, resp)
}

func GetItem(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")

	uid, err := uuid.Parse(id)
	if err != nil {
		http.Error(w, "Invalid id", http.StatusBadRequest)
		return
	}

	item, err := db.GetItem(uid)
	if err != nil {
		log.Println(err)

		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "Item not found", http.StatusNotFound)
		} else {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	resp := GetItemResponse{
		ID:    item.ID,
		Name:  item.Name,
		Count: item.Count,
		Price: item.Price,
	}

	writeJSON(w, http.StatusOK, resp)
}

func PutItem(w http.ResponseWriter, r *http.Request) {
	var req PostCreateItemRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	item := db.Item{
		ID:    uuid.New(),
		Name:  req.Name,
		Price: req.Price,
		Count: req.Count,
	}
	err = db.AddItem(item)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, http.StatusCreated, PostCreateItemResponse{
		ID: item.ID,
	})

}

func DeleteItem(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	uid, err := uuid.Parse(id)
	if err != nil {
		http.Error(w, "Invalid id", http.StatusBadRequest)
		return
	}
	err = db.DeleteItem(uid)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Println(err)
	}
}
