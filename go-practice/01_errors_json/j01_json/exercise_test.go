package j01

// TODO: Самостоятельно вызови json.Marshal и проверь имена полей id, name и email.
// TODO: Проверь отсутствие пустого email в JSON.
// TODO: Вызови json.Unmarshal и проверь восстановление значений User.
// TODO: Проверь ошибку при чтении повреждённого JSON.

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestUnmarshal(t *testing.T) {
	user := User{
		ID:    10,
		Name:  "Example Student",
		Email: "student@example.com",
	}

	data, err := json.Marshal(user)
	if err != nil {
		t.Fatalf("test unmarshal: %v", err)
	}
	t.Logf("marshal data: %v", string(data))

	var res User
	if err := json.Unmarshal(data, &res); err != nil {
		t.Fatalf("test unmarshal: %v", err)
	}

	if res != user {
		t.Fatalf("inccorect unmarshal")
	}
}

func TestUnmarshalBuff(t *testing.T) {
	user := User{
		ID:    10,
		Name:  "Example Student",
		Email: "student@example.com",
	}

	var data bytes.Buffer

	enc := json.NewEncoder(&data)

	if err := enc.Encode(user); err != nil {
		t.Fatalf("test unmarshal: %v", err)
	}

	t.Logf("marshal data: %v", data.String())

	var res User

	dec := json.NewDecoder(&data)

	if err := dec.Decode(&res); err != nil {
		t.Fatalf("test unmarshal: %v", err)
	}

	if res != user {
		t.Fatalf("inccorect unmarshal")
	}
}
