package db

import "github.com/google/uuid"

type Item struct {
	ID    uuid.UUID
	Name  string
	Price uint32
	Count uint32
}
