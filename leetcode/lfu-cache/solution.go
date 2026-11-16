package lfucache

import "container/list"

type entry struct {
	key, value, fr int
}

type LFUCache struct {
	capacity int
	minFr    int
	frMp     map[int]*list.List
	dataMp   map[int]*list.Element
}

func Constructor(capacity int) LFUCache {
	return LFUCache{
		capacity: capacity,
		frMp:     make(map[int]*list.List),
		dataMp:   make(map[int]*list.Element),
	}
}

func (c *LFUCache) Get(key int) int {
	el, ok := c.dataMp[key]
	if !ok {
		return -1
	}
	return c.touch(el).value
}

func (c *LFUCache) Put(key, value int) {
	if c.capacity <= 0 {
		return
	}
	if el, ok := c.dataMp[key]; ok {
		c.touch(el).value = value
		return
	}

	if len(c.dataMp) == c.capacity {
		bucket := c.frMp[c.minFr]
		victim := bucket.Remove(bucket.Back()).(*entry)
		delete(c.dataMp, victim.key)
		if bucket.Len() == 0 {
			delete(c.frMp, c.minFr)
		}
	}

	c.push(&entry{key: key, value: value, fr: 1})
	c.minFr = 1
}

func (c *LFUCache) touch(el *list.Element) *entry {
	item := el.Value.(*entry)
	bucket := c.frMp[item.fr]
	bucket.Remove(el)
	if bucket.Len() == 0 {
		delete(c.frMp, item.fr)
		if item.fr == c.minFr {
			c.minFr++
		}
	}
	item.fr++
	c.push(item)
	return item
}

func (c *LFUCache) push(item *entry) {
	bucket := c.frMp[item.fr]
	if bucket == nil {
		bucket = list.New()
		c.frMp[item.fr] = bucket
	}
	c.dataMp[item.key] = bucket.PushFront(item)
}
