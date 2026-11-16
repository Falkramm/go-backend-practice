package j01

// TODO: Создай User с полями ID int, Name string и Email string.
// TODO: Добавь JSON-теги id, name и email,omitempty.

type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email,omitempty"`
}
