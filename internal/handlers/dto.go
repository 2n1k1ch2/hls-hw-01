package handlers

import "github.com/google/uuid"

type ItemResponse struct {
	ID    uuid.UUID `json:"id"`
	Name  string    `json:"name"`
	Count uint32    `json:"count"`
	Price uint32    `json:"price"`
}
type GetItemsRequest struct {
}

type GetItemsResponse struct {
	Items []ItemResponse `json:"items"`
}
type GetItemRequest struct {
	ID uuid.UUID `json:"id"`
}

type GetItemResponse struct {
	ID    uuid.UUID `json:"uuid"`
	Name  string    `json:"name"`
	Count uint32    `json:"count"`
	Price uint32    `json:"price"`
}
type PostCreateItemRequest struct {
	Name  string `json:"name"`
	Count uint32 `json:"count"`
	Price uint32 `json:"price"`
}

type PostCreateItemResponse struct {
	ID uuid.UUID `json:"id"`
}

type DeleteItemRequest struct {
	ID uuid.UUID `json:"id"`
}

type DeleteItemResponse struct {
}
