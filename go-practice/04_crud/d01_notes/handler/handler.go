package handler

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Note struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	Body      *string   `json:"body"`
	Pinned    bool      `json:"pinned"`
	CreatedAt time.Time `json:"created_at"`
}

type Handler struct {
	db *sql.DB
}

func New(db *sql.DB) http.Handler {
	h := &Handler{db: db}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /notes", h.CreateNote)
	mux.HandleFunc("GET /notes", h.GetNotes)
	mux.HandleFunc("GET /notes/{id}", h.GetNote)
	mux.HandleFunc("PUT /notes/{id}", h.ReplaceNote)
	mux.HandleFunc("PATCH /notes/{id}", h.PatchNote)
	mux.HandleFunc("DELETE /notes/{id}", h.DeleteNote)
	return mux
}

func (h *Handler) CreateNote(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		http.Error(
			w,
			"invalid method",
			http.StatusMethodNotAllowed,
		)
		return
	}

	if r.Header.Get("Content-Type") != "application/json" {
		http.Error(
			w,
			"invalid request",
			http.StatusBadRequest,
		)
		return
	}

	var input struct {
		Title  string  `json:"title"`
		Body   *string `json:"body"`
		Pinned bool    `json:"pinned"`
	}

	dec := json.NewDecoder(r.Body)

	if err := dec.Decode(&input); err != nil {
		http.Error(
			w,
			"invalid json",
			http.StatusBadRequest,
		)
	}

	if strings.TrimSpace(input.Title) == "" {
		http.Error(
			w,
			"invalid request",
			http.StatusBadRequest,
		)
		return
	}

	var note Note

	const query = `
		insert into notes (title, body, pinned) values ($1, $2, $3)
		returning id, title, body, pinned, created_at;
	`
	err := h.db.QueryRowContext(
		r.Context(),
		query,
		input.Title,
		input.Body,
		input.Pinned,
	).Scan(
		&note.ID,
		&note.Title,
		&note.Body,
		&note.Pinned,
		&note.CreatedAt,
	)

	if err != nil {
		http.Error(
			w,
			"internal server error",
			http.StatusInternalServerError,
		)
		return
	}

	data, err := json.Marshal(note)
	if err != nil {
		http.Error(
			w,
			"internal server error",
			http.StatusInternalServerError,
		)
		return
	}

	w.Write(data)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
}

func (h *Handler) GetNote(w http.ResponseWriter, r *http.Request) {
	// TODO: Parse r.PathValue("id"), SELECT with r.Context(), then write 200 or 404.
	if r.Method != http.MethodGet {
		http.Error(
			w,
			"invalid method",
			http.StatusMethodNotAllowed,
		)
		return
	}
	id, err := strconv.Atoi(r.PathValue("id"))

	if err != nil {
		http.Error(
			w,
			"invalid request",
			http.StatusBadRequest,
		)
		return
	}

	const query = `
		select
			id,
			title,
			body,
			pinned,
			created_at
		from notes
		where id = $1;
	`

	var note Note

	err = h.db.QueryRowContext(
		r.Context(),
		query,
		id,
	).Scan(
		&note.ID,
		&note.Title,
		&note.Body,
		&note.Pinned,
		&note.CreatedAt,
	)

	if err != nil {
		http.Error(
			w,
			"internal server error",
			http.StatusInternalServerError,
		)
		return
	}

	data, err := json.Marshal(note)

	if err != nil {
		http.Error(
			w,
			"internal server error",
			http.StatusInternalServerError,
		)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func (h *Handler) GetNotes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(
			w,
			"invalid method",
			http.StatusMethodNotAllowed,
		)
		return
	}

	var err error
	limit, offset := 20, 0

	if r.URL.Query().Has("limit") {
		limit, err = strconv.Atoi(r.PathValue("limit"))

		if err != nil || limit <= 0 {
			http.Error(
				w,
				"invalid request",
				http.StatusBadRequest,
			)
			return
		}
	}

	if r.URL.Query().Has("limit") {
		offset, err = strconv.Atoi(r.PathValue("offset"))

		if err != nil {
			log.Printf("get notes: %v", err)
			http.Error(
				w,
				"invalid request",
				http.StatusBadRequest,
			)
			return
		}
	}

	const query = `
		select
			id,
			title,
			body,
			pinned,
			created_at
		from notes
		order by id
		limit $1
		offset $2;
	`

	rows, err := h.db.QueryContext(
		r.Context(),
		query,
		limit,
		offset,
	)

	if err != nil {
		http.Error(
			w,
			"internal server error",
			http.StatusInternalServerError,
		)
		return
	}

	defer rows.Close()

	var notes []Note

	for rows.Next() {
		var note Note
		err := rows.Scan(
			&note.ID,
			&note.Title,
			&note.Body,
			&note.Pinned,
			&note.CreatedAt,
		)
		if err != nil {
			http.Error(
				w,
				"internal server error",
				http.StatusInternalServerError,
			)
			return
		}
	}

	if rows.Err() != nil {
		http.Error(
			w,
			"internal server error",
			http.StatusInternalServerError,
		)
		return
	}

	data, err := json.Marshal(notes)

	if err != nil {
		http.Error(
			w,
			"internal server error",
			http.StatusInternalServerError,
		)
		return
	}

	w.Write(data)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) ReplaceNote(w http.ResponseWriter, r *http.Request) {
	// TODO: Parse ID and JSON, UPDATE all mutable fields, then write 200 or 404.
	http.Error(w, "not implemented", http.StatusNotImplemented)
}

func (h *Handler) PatchNote(w http.ResponseWriter, r *http.Request) {
	// TODO: Track field presence, UPDATE supplied fields atomically, then write 200 or 404.
	http.Error(w, "not implemented", http.StatusNotImplemented)
}

func (h *Handler) DeleteNote(w http.ResponseWriter, r *http.Request) {
	// TODO: Parse ID, DELETE with r.Context(), check RowsAffected, then write 204 or 404.
	http.Error(w, "not implemented", http.StatusNotImplemented)
}
