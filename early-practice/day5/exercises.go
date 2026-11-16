// День 5 — практика: defer, panic/recover, ошибки.
//
//	cd early-practice/day5
//	go test -v
package day5

import (
	"errors"
	"fmt"
)

// ============ 1. Recover-обёртка (мини-версия HTTP-мидлвари) ============

// Run вызывает f и превращает её панику в error вида "panic: <значение>".
// Без паники — возвращает nil. Подсказка: именованный результат + defer.
func Run(f func()) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(error); ok {
				err = fmt.Errorf("panic: %w", e)
			} else {
				err = fmt.Errorf("panic: %v", r)
			}

		}
	}()
	f()

	return nil
}

// ============ 2. Трёхслойные ошибки: repo → service → handler ============

// Сентинел "не найдено" уровня домена.
var ErrNotFound = errors.New("not found")

// ValidationError — ошибка с данными: какое поле не прошло проверку.
type ValidationError struct {
	Field string
}

func (e *ValidationError) Error() string {
	return "invalid field: " + e.Field
}

// repoGetUser — нижний слой (менять не нужно).
// id < 0      -> ValidationError{Field: "id"}
// id == 0     -> ErrNotFound
// иначе       -> "user-<id>", nil
func repoGetUser(id int) (string, error) {
	switch {
	case id < 0:
		return "", &ValidationError{Field: "id"}
	case id == 0:
		return "", ErrNotFound
	}
	return "user-1", nil
}

// LoadProfile — сервисный слой: зови repoGetUser и добавляй контекст
// "load profile: ..." так, чтобы errors.Is/As наверху ПРОДОЛЖАЛИ работать.
func LoadProfile(id int) (string, error) {
	user, err := repoGetUser(id)
	if err != nil {
		return user, fmt.Errorf("load profile: %w", err)
	}
	return user, err
}

// HandleStatus — «хендлер»: по ошибке из LoadProfile верни HTTP-статус:
// нет ошибки -> 200, ErrNotFound в цепочке -> 404,
// ValidationError в цепочке -> 400, всё прочее -> 500.
func HandleStatus(id int) int {
	_, err := LoadProfile(id)
	var vError *ValidationError
	switch {
	case errors.Is(err, ErrNotFound):
		return 404
	case errors.As(err, &vError):
		return 400
	case err != nil:
		return 500
	default:
		return 200
	}
}

// ============ 3. errors.Join: закрыть всё и ничего не потерять ============

// Closer — то, что умеет закрываться (может вернуть ошибку).
type Closer interface {
	Close() error
}

// CloseAll закрывает ВСЕ переданные closers (даже если первые падают!)
// и возвращает все ошибки закрытия одной ошибкой. Ошибок нет -> nil.
func CloseAll(closers ...Closer) error {
	errs := make([]error, 0, len(closers))
	for _, closer := range closers {
		errs = append(errs, closer.Close())
	}
	return errors.Join(errs...)
}
