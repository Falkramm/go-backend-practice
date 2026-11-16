package e01

// TODO: Напиши проверки для входов "18", "0", "-1" и "abc".
// TODO: Проверь значение и nil error для корректного возраста.
// TODO: Распознай ErrNegativeAge через errors.Is.
// TODO: Для "abc" извлеки *strconv.NumError через errors.As.

import (
	"errors"
	"testing"
)

var testCases = []struct {
	name  string
	input string
	want  int
	err   error
}{
	{
		"positive",
		"12",
		12,
		nil,
	},

	{
		"positive 2",
		"1241",
		1241,
		nil,
	},
	{
		"positive 3",
		"0012",
		12,
		nil,
	},
	{
		"negative",
		"-123",
		0,
		ErrNegativeAge,
	},
	{
		"zero",
		"000000",
		0,
		nil,
	},
	{
		"zero 1",
		"0",
		0,
		nil,
	},
}

func TestParseAge(t *testing.T) {
	for _, testCase := range testCases {
		t.Run(
			testCase.name,
			func(t *testing.T) {
				i, err := ParseAge(testCase.input)
				if i != testCase.want {
					t.Errorf("got: %v; want: %v;", i, testCase.want)
				}

				if (err != nil || testCase.err != nil) && !errors.Is(err, testCase.err) {
					t.Errorf("error got: %v; want: %v;", err, testCase.err)
				}
			},
		)
	}
}
