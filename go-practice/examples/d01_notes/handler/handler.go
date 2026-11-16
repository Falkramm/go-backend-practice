package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
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
	var input struct {
		Title  string  `json:"title"`
		Body   *string `json:"body"`
		Pinned bool    `json:"pinned"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(input.Title) == "" {
		http.Error(w, "title is required", http.StatusBadRequest)
		return
	}

	var note Note
	err := h.db.QueryRowContext(r.Context(), `
		INSERT INTO notes (title, body, pinned) VALUES ($1, $2, $3)
		RETURNING id, title, body, pinned, created_at`, input.Title, input.Body, input.Pinned,
	).Scan(&note.ID, &note.Title, &note.Body, &note.Pinned, &note.CreatedAt)
	if err != nil {
		log.Printf("%s %s: %v", r.Method, r.URL.Path, err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Location", "/notes/"+strconv.FormatInt(note.ID, 10))
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(note)
}

func (h *Handler) GetNote(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		http.Error(w, "invalid note ID", http.StatusBadRequest)
		return
	}

	var note Note
	err = h.db.QueryRowContext(r.Context(), `
		SELECT id, title, body, pinned, created_at FROM notes WHERE id = $1`, id,
	).Scan(&note.ID, &note.Title, &note.Body, &note.Pinned, &note.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "note not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("%s %s: %v", r.Method, r.URL.Path, err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(note)
}

func (h *Handler) GetNotes(w http.ResponseWriter, r *http.Request) {
	limit, offset := 20, 0
	var err error
	query := r.URL.Query()
	if query.Has("limit") {
		limit, err = strconv.Atoi(query.Get("limit"))
		if err != nil || limit < 1 || limit > 100 {
			http.Error(w, "invalid limit", http.StatusBadRequest)
			return
		}
	}
	if query.Has("offset") {
		offset, err = strconv.Atoi(query.Get("offset"))
		if err != nil || offset < 0 {
			http.Error(w, "invalid offset", http.StatusBadRequest)
			return
		}
	}

	rows, err := h.db.QueryContext(r.Context(), `
		SELECT id, title, body, pinned, created_at
		FROM notes ORDER BY id LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		log.Printf("%s %s: %v", r.Method, r.URL.Path, err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	notes := make([]Note, 0)
	for rows.Next() {
		var note Note
		if err := rows.Scan(&note.ID, &note.Title, &note.Body, &note.Pinned, &note.CreatedAt); err != nil {
			log.Printf("%s %s: %v", r.Method, r.URL.Path, err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		notes = append(notes, note)
	}
	if err := rows.Err(); err != nil {
		log.Printf("%s %s: %v", r.Method, r.URL.Path, err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(notes)
}

func (h *Handler) ReplaceNote(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		http.Error(w, "invalid note ID", http.StatusBadRequest)
		return
	}
	var input struct {
		Title  string  `json:"title"`
		Body   *string `json:"body"`
		Pinned bool    `json:"pinned"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(input.Title) == "" {
		http.Error(w, "title is required", http.StatusBadRequest)
		return
	}

	var note Note
	err = h.db.QueryRowContext(r.Context(), `
		UPDATE notes SET title = $2, body = $3, pinned = $4 WHERE id = $1
		RETURNING id, title, body, pinned, created_at`, id, input.Title, input.Body, input.Pinned,
	).Scan(&note.ID, &note.Title, &note.Body, &note.Pinned, &note.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "note not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("%s %s: %v", r.Method, r.URL.Path, err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(note)
}

func (h *Handler) PatchNote(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		http.Error(w, "invalid note ID", http.StatusBadRequest)
		return
	}
	var input *struct {
		Title  *string         `json:"title"`
		Body   json.RawMessage `json:"body"`
		Pinned *bool           `json:"pinned"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil || input == nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if input.Title != nil && strings.TrimSpace(*input.Title) == "" {
		http.Error(w, "title is required", http.StatusBadRequest)
		return
	}
	var body *string
	if input.Body != nil {
		if err := json.Unmarshal(input.Body, &body); err != nil {
			http.Error(w, "invalid body", http.StatusBadRequest)
			return
		}
	}

	var note Note
	err = h.db.QueryRowContext(r.Context(), `
		UPDATE notes SET
			title = COALESCE($2, title),
			body = CASE WHEN $3 THEN $4::text ELSE body END,
			pinned = COALESCE($5, pinned)
		WHERE id = $1
		RETURNING id, title, body, pinned, created_at`, id, input.Title, input.Body != nil, body, input.Pinned,
	).Scan(&note.ID, &note.Title, &note.Body, &note.Pinned, &note.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "note not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("%s %s: %v", r.Method, r.URL.Path, err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(note)
}

func (h *Handler) DeleteNote(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		http.Error(w, "invalid note ID", http.StatusBadRequest)
		return
	}

	result, err := h.db.ExecContext(r.Context(), "DELETE FROM notes WHERE id = $1", id)
	if err != nil {
		log.Printf("%s %s: %v", r.Method, r.URL.Path, err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	affected, err := result.RowsAffected()
	if err != nil {
		log.Printf("%s %s: %v", r.Method, r.URL.Path, err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if affected == 0 {
		http.Error(w, "note not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
