package lru

import "testing"

func mustGet(t *testing.T, c *LRU, key, want int) {
	t.Helper()
	v, ok := c.Get(key)
	if !ok || v != want {
		t.Fatalf("Get(%d) = (%d, %v), ожидалось (%d, true)", key, v, ok, want)
	}
}

func mustMiss(t *testing.T, c *LRU, key int) {
	t.Helper()
	if v, ok := c.Get(key); ok {
		t.Fatalf("Get(%d) = (%d, true), ожидался промах — ключ должен быть выселен/отсутствовать", key, v)
	}
}

func TestBasicPutGet(t *testing.T) {
	c := NewLRU(2)
	c.Put(1, 100)
	c.Put(2, 200)
	mustGet(t, c, 1, 100)
	mustGet(t, c, 2, 200)
	if c.Len() != 2 {
		t.Fatalf("Len = %d, ожидалось 2", c.Len())
	}
}

func TestUpdateExisting(t *testing.T) {
	c := NewLRU(2)
	c.Put(1, 100)
	c.Put(1, 111) // обновление, НЕ новый элемент
	mustGet(t, c, 1, 111)
	if c.Len() != 1 {
		t.Fatalf("Len = %d после двойного Put одного ключа, ожидалось 1", c.Len())
	}
}

func TestEvictsOldest(t *testing.T) {
	c := NewLRU(2)
	c.Put(1, 100)
	c.Put(2, 200)
	c.Put(3, 300) // переполнение: выселяется 1 (самый давний)
	mustMiss(t, c, 1)
	mustGet(t, c, 2, 200)
	mustGet(t, c, 3, 300)
}

func TestGetRefreshesRecency(t *testing.T) {
	c := NewLRU(2)
	c.Put(1, 100)
	c.Put(2, 200)
	mustGet(t, c, 1, 100) // 1 стал свежим, теперь самый давний — 2
	c.Put(3, 300)         // выселиться должен 2, а не 1!
	mustGet(t, c, 1, 100)
	mustMiss(t, c, 2)
	mustGet(t, c, 3, 300)
}

func TestPutRefreshesRecency(t *testing.T) {
	c := NewLRU(2)
	c.Put(1, 100)
	c.Put(2, 200)
	c.Put(1, 111) // обновление тоже поднимает 1 в свежие
	c.Put(3, 300) // выселяется 2
	mustGet(t, c, 1, 111)
	mustMiss(t, c, 2)
}

func TestCapacityOne(t *testing.T) {
	c := NewLRU(1)
	c.Put(1, 100)
	mustGet(t, c, 1, 100)
	c.Put(2, 200)
	mustMiss(t, c, 1)
	mustGet(t, c, 2, 200)
	if c.Len() != 1 {
		t.Fatalf("Len = %d, ожидалось 1", c.Len())
	}
}

func TestLongSequence(t *testing.T) {
	c := NewLRU(3)
	for i := 0; i < 100; i++ {
		c.Put(i, i*10)
	}
	// должны остаться только 97, 98, 99
	mustMiss(t, c, 96)
	mustGet(t, c, 97, 970)
	mustGet(t, c, 98, 980)
	mustGet(t, c, 99, 990)
	if c.Len() != 3 {
		t.Fatalf("Len = %d, ожидалось 3", c.Len())
	}
}
