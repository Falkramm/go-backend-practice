package catalog

import (
	"database/sql"
	"encoding/json"
	"net/http"
)

type Category struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type CategoryInput struct {
	Name string `json:"name"`
}

type Product struct {
	ID           int64   `json:"id"`
	SKU          string  `json:"sku"`
	CategoryID   int64   `json:"category_id"`
	CategoryName string  `json:"category_name"`
	Name         string  `json:"name"`
	PriceCents   int64   `json:"price_cents"`
	Stock        int64   `json:"stock"`
	Description  *string `json:"description"`
}

type ProductInput struct {
	SKU         string  `json:"sku"`
	CategoryID  int64   `json:"category_id"`
	Name        string  `json:"name"`
	PriceCents  int64   `json:"price_cents"`
	Stock       int64   `json:"stock"`
	Description *string `json:"description"`
}

type PatchProductInput struct {
	SKU         *string         `json:"sku"`
	CategoryID  *int64          `json:"category_id"`
	Name        *string         `json:"name"`
	PriceCents  *int64          `json:"price_cents"`
	Stock       *int64          `json:"stock"`
	Description json.RawMessage `json:"description"`
}

type Handler struct {
	db *sql.DB
}

func New(db *sql.DB) http.Handler {
	h := &Handler{db: db}
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	return mux
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
}
