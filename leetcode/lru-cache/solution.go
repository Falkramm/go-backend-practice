package lrucache

import "container/list"

type entry struct {
	key   int
	value int
	it    *list.Element
}

type LRUCache struct {
	capacity int
	mp       map[int]entry
	order    *list.List
}

func Constructor(capacity int) LRUCache {
	return LRUCache{
		capacity: capacity,
		mp:       make(map[int]entry),
		order:    list.New(),
	}
}

func (this *LRUCache) Get(key int) int {
	v, ok := this.mp[key]
	if !ok {
		return -1
	}

	this.order.MoveToFront(v.it)
	return v.value
}

func (this *LRUCache) Put(key int, value int) {
	if v, ok := this.mp[key]; ok {
		this.mp[key] = entry{
			key:   key,
			value: value,
			it:    v.it,
		}
		this.order.MoveToFront(v.it)
		return
	}

	if len(this.mp) == this.capacity {
		back := this.order.Back()
		delete(this.mp, back.Value.(int))
		this.order.Remove(back)
	}

	this.mp[key] = entry{
		key:   key,
		value: value,
		it:    this.order.PushFront(key),
	}
}

/**
 * Your LRUCache object will be instantiated and called as such:
 * obj := Constructor(capacity);
 * param_1 := obj.Get(key);
 * obj.Put(key,value);
 */
