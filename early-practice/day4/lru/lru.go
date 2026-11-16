// LRU-кеш: все операции за O(1).
//
// Структура-классика: мапа key -> узел двусвязного списка.
// Список хранит порядок использования: голова — самое свежее,
// хвост — кандидат на выселение.
//
//	cd early-practice/day4/lru
//	go test -v
package lru

import "sync"

type Node struct {
	key, value int
	prev, next *Node
}

type LRU struct {
	capacity   int
	m          map[int]*Node
	head, tail *Node
	mu         sync.Mutex
}

// NewLRU создаёт кеш на capacity элементов. capacity >= 1.
func NewLRU(capacity int) *LRU {
	return &LRU{
		capacity: capacity,
		m:        make(map[int]*Node),
	}
}

// unlink выдёргивает узел из списка, поддерживая head/tail.
func (c *LRU) unlink(n *Node) {
	if n.prev != nil {
		n.prev.next = n.next
	} else {
		c.head = n.next
	}
	if n.next != nil {
		n.next.prev = n.prev
	} else {
		c.tail = n.prev
	}
	n.prev, n.next = nil, nil
}

// pushFront вставляет узел в голову списка.
func (c *LRU) pushFront(n *Node) {
	n.next = c.head
	if c.head != nil {
		c.head.prev = n
	}
	c.head = n
	if c.tail == nil {
		c.tail = n
	}
}

// upNode поднимает узел в "самые свежие".
func (c *LRU) upNode(n *Node) {
	if n == c.head {
		return
	}
	c.unlink(n)
	c.pushFront(n)
}

// extract выселяет самый давно не использованный узел (хвост).
func (c *LRU) extract() {
	delete(c.m, c.tail.key)
	c.unlink(c.tail)
}

// Get возвращает значение по ключу и поднимает ключ в "самые свежие".
// Нет ключа -> (0, false).
func (c *LRU) Get(key int) (int, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	n, ok := c.m[key]
	if !ok {
		return 0, false
	}
	c.upNode(n)
	return n.value, true
}

// Put кладёт пару key/value. Если ключ есть — обновляет значение
// (и поднимает в свежие). Если нет и кеш полон — выселяет самый
// давно не использованный ключ.
func (c *LRU) Put(key, value int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if n, ok := c.m[key]; ok {
		n.value = value
		c.upNode(n)
		return
	}

	if len(c.m) == c.capacity {
		c.extract()
	}

	n := &Node{key: key, value: value}
	c.pushFront(n)
	c.m[key] = n
}

// Len возвращает текущее число элементов.
func (c *LRU) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.m)
}
