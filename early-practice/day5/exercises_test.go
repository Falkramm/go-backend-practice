package day5

import (
	"errors"
	"strings"
	"testing"
)

func TestRun(t *testing.T) {
	if err := Run(func() {}); err != nil {
		t.Fatalf("без паники err = %v, ожидался nil", err)
	}
	err := Run(func() { panic("boom") })
	if err == nil {
		t.Fatal("паника должна была стать ошибкой")
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Fatalf("в ошибке нет значения паники: %q", err)
	}
}

func TestLoadProfileWrapping(t *testing.T) {
	// happy path
	s, err := LoadProfile(1)
	if err != nil || s != "user-1" {
		t.Fatalf("LoadProfile(1) = (%q, %v), ожидалось (user-1, nil)", s, err)
	}

	// сентинел должен быть виден СКВОЗЬ обёртку
	_, err = LoadProfile(0)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("errors.Is(err, ErrNotFound) = false — цепочка порвана (%%v вместо %%w?): %v", err)
	}
	// и контекст должен быть добавлен
	if !strings.Contains(err.Error(), "load profile") {
		t.Fatalf("нет контекста 'load profile' в ошибке: %q", err)
	}

	// тип с данными должен доставаться сквозь обёртку
	_, err = LoadProfile(-5)
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("errors.As не нашёл ValidationError в цепочке: %v", err)
	}
	if ve.Field != "id" {
		t.Fatalf("ve.Field = %q, ожидалось id", ve.Field)
	}
}

func TestHandleStatus(t *testing.T) {
	cases := map[int]int{1: 200, 0: 404, -1: 400}
	for id, want := range cases {
		if got := HandleStatus(id); got != want {
			t.Errorf("HandleStatus(%d) = %d, ожидалось %d", id, got, want)
		}
	}
}

// fakeCloser считает вызовы Close и может падать.
type fakeCloser struct {
	name   string
	fail   bool
	closed bool
}

func (f *fakeCloser) Close() error {
	f.closed = true
	if f.fail {
		return errors.New(f.name + ": close failed")
	}
	return nil
}

func TestCloseAll(t *testing.T) {
	a := &fakeCloser{name: "a"}
	b := &fakeCloser{name: "b", fail: true}
	c := &fakeCloser{name: "c", fail: true}

	err := CloseAll(a, b, c)

	for _, fc := range []*fakeCloser{a, b, c} {
		if !fc.closed {
			t.Fatalf("%s не был закрыт — падение одного Close не должно останавливать остальные", fc.name)
		}
	}
	if err == nil {
		t.Fatal("две ошибки закрытия потеряны")
	}
	if !strings.Contains(err.Error(), "b:") || !strings.Contains(err.Error(), "c:") {
		t.Fatalf("в ошибке должны быть ОБЕ причины, got: %q", err)
	}

	if err := CloseAll(a); err != nil {
		t.Fatalf("без падений err = %v, ожидался nil", err)
	}
}
