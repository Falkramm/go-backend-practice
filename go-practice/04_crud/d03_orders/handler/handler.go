package handler

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"strconv"
)

type Product struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	PriceCents int64  `json:"price_cents"`
	Stock      int64  `json:"stock"`
}

type OrderItem struct {
	ProductID  int64  `json:"product_id"`
	Qty        int64  `json:"qty"`
	PriceCents *int64 `json:"price_cents"`
}

type Order struct {
	ID     int64       `json:"id"`
	Status string      `json:"status"`
	Items  []OrderItem `json:"items"`
}

type ItemInput struct {
	ProductID int64 `json:"product_id"`
	Qty       int64 `json:"qty"`
}

type OrderInput struct {
	Items []ItemInput `json:"items"`
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
	mux.HandleFunc("GET /orders/{id}", h.GetOrder)
}

func (h *Handler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	// TODO: Декодировать OrderInput и создать черновик с позициями в транзакции.
	if r.Method != http.MethodPost {
		http.Error(
			w,
			"invalid method",
			http.StatusMethodNotAllowed,
		)
		log.Printf("mathod got: %v; want: %v", r.Method, http.MethodPost)
		return
	}

	var input struct {
		Items []Order `json:"items"`
	}

	dec := json.NewDecoder(r.Body)

	if err := dec.Decode(&input); err != nil {
		http.Error(
			w,
			"invalid request",
			http.StatusBadRequest,
		)
		log.Printf("invalid request: %v", err)
		return
	}

	tx, err := h.db.BeginTx(
		r.Context(),
		nil,
	)
	if err != nil {
		http.Error(
			w,
			"internal server error",
			http.StatusInternalServerError,
		)
		log.Printf("create order: %v", err)
		return
	}

	defer tx.Rollback()

	for _, item := range input.Items {
		const query = `
			insert into orders(id,  status, items)
			values
				($1, $2, $3)
			on conflict (id) do update
				set
					status = EXCLUDE.status,
					items = EXCLUDE.items;
		`
		_, err := tx.ExecContext(
			r.Context(),
			query,
			item.ID,
			item.Status,
			item.Items,
		)

		if err != nil {
			http.Error(
				w,
				"internal server error",
				http.StatusInternalServerError,
			)
			log.Printf("create order: %v", err)
			return
		}

	}

	tx.Commit()

	w.WriteHeader(http.StatusCreated)

}

func (h *Handler) GetOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(
			w,
			"invalid method",
			http.StatusMethodNotAllowed,
		)
		log.Printf("method got: %v; want: %v", r.Method, http.MethodGet)
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
			status
		from orders
		where id = $1;
	`

	var order Order

	tx, err := h.db.BeginTx(r.Context(), nil)

	if err != nil {
		http.Error(
			w,
			"invalid request",
			http.StatusBadRequest,
		)
		return
	}

	err = tx.QueryRowContext(
		r.Context(),
		query,
		id,
	).Scan(
		&order.ID,
		&order.Status,
	)

	if err != nil {
		http.Error(
			w,
			"internal server error",
			http.StatusInternalServerError,
		)
		return
	}

	data, err := json.Marshal(order)

	if err != nil {
		http.Error(
			w,
			"internal server error",
			http.StatusInternalServerError,
		)
		return
	}

	w.Write(data)

	w.Header().Set("content-type", "application/json")
	w.WriteHeader(http.StatusOK)

}

func (h *Handler) GetOrders(w http.ResponseWriter, r *http.Request) {
	u, err := url.Parse("/test_req")
	var data []byte

	req, err := http.NewRequest(
		http.MethodPost,
		u.String(),
		bytes.NewReader(data),
	)

	resp, err := http.DefaultClient.Do(req)

	defer resp.Body.Close()

	var res Order
	dec := json.NewDecoder(r.Body)

	dec.Decode(&res)

}

func (h *Handler) ReplaceOrder(w http.ResponseWriter, r *http.Request) {
	// TODO: Атомарно заменить позиции черновика из OrderInput.
	http.Error(w, "not implemented", http.StatusNotImplemented)
}

func (h *Handler) DeleteOrder(w http.ResponseWriter, r *http.Request) {
	// TODO: Атомарно удалить черновик вместе с позициями.
	http.Error(w, "not implemented", http.StatusNotImplemented)
}

func (h *Handler) ConfirmOrder(w http.ResponseWriter, r *http.Request) {
	// TODO: В транзакции проверить остатки, зафиксировать цены и подтвердить заказ.
	http.Error(w, "not implemented", http.StatusNotImplemented)
}
