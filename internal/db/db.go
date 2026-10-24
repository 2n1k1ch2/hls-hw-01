package db

import (
	"database/sql"
	"log"

	"github.com/google/uuid"
)

var Database *sql.DB

func GetItems() ([]Item, error) {
	var items []Item
	rows, err := Database.Query("SELECT id, name, count, price FROM items")
	if err != nil {
		log.Print(err)
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var item Item
		err = rows.Scan(&item.ID, &item.Name, &item.Count, &item.Price)
		if err != nil {
			log.Print(err)
			return nil, err
		}
		items = append(items, item)
	}
	if rows.Err() != nil {
		log.Print(err)
		return nil, err
	}
	return items, nil
}

func GetItem(id uuid.UUID) (Item, error) {
	var item Item
	row := Database.QueryRow("SELECT id, name, count, price FROM items WHERE id = $1", id)
	err := row.Scan(&item.ID, &item.Name, &item.Count, &item.Price)
	if err != nil {
		log.Print(err)
		return Item{}, err
	}
	return item, nil
}

func AddItem(item Item) error {
	_, err := Database.Exec("INSERT INTO items (id, name, count, price) VALUES ($1, $2, $3, $4) ",
		item.ID,
		item.Name,
		item.Count,
		item.Price)
	if err != nil {
		log.Print(err)
		return err
	}
	return nil
}

func DeleteItem(id uuid.UUID) error {
	_, err := Database.Exec("DELETE FROM items WHERE id = $1", id)
	if err != nil {
		log.Print(err)
		return err
	}
	return nil
}
