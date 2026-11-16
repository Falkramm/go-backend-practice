package handler

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
)

// TODO: Реализовать HTTP CRUD категорий и товаров, фильтры, пагинацию, PATCH и upsert по SKU.
// TODO: Передавать request.Context и соблюдать HTTP/JSON-контракт из README.md.

type Handler struct {
	db *sql.DB
}

func NewHandler(db *sql.DB) http.Handler {
	h := &Handler{
		db: db,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /notes", h.CreateCategory)
	return mux
}

type Category struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func (h *Handler) CreateCategory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(
			w,
			"invalid method",
			http.StatusMethodNotAllowed,
		)
		log.Printf("method got: %v, want: %v", r.Method, http.MethodPost)
		return
	}

	var input struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}

	dec := json.NewDecoder(r.Body)

	if err := dec.Decode(&input); err != nil {
		http.Error(
			w,
			"invalid request",
			http.StatusBadRequest,
		)
		log.Printf("create category: %v", err)
		return
	}

	const query = `
		insert into categories(id, name)
		values ($1, $2)
		on conflict (id) do update
			set name = EXCLUDE.name
		returning id, name;
	`

	var category Category

	err := h.db.QueryRowContext(
		r.Context(),
		query,
		input.ID,
		input.Name,
	).Scan(
		&category.ID,
		&category.Name,
	)

	if err != nil {
		http.Error(
			w,
			"internal server error",
			http.StatusInternalServerError,
		)
		log.Printf("create category: %v", err)
		return
	}

	data, err := json.Marshal(category)

	if err != nil {
		http.Error(
			w,
			"internal server error",
			http.StatusInternalServerError,
		)
		log.Printf("create category: %v", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
	w.WriteHeader(http.StatusCreated)
}
