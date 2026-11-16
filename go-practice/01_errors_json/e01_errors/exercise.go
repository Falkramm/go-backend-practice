package e01

import (
	"errors"
	"fmt"
	"strconv"
)

// TODO: Объяви sentinel ErrNegativeAge через errors.New.
// TODO: Реализуй ParseAge(input string) (int, error) через strconv.Atoi.
// TODO: Для неотрицательного возраста верни число и nil.
// TODO: Для отрицательного возраста верни ErrNegativeAge.
// TODO: Ошибку преобразования оберни через fmt.Errorf с %w.

var (
	ErrNegativeAge = errors.New("negative age")
)

func ParseAge(input string) (int, error) {
	i, err := strconv.Atoi(input)
	if err != nil {
		return 0, fmt.Errorf("parse age: %w", err)
	}

	if i < 0 {
		return 0, ErrNegativeAge
	}

	return i, nil
}
