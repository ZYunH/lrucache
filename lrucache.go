package lrucache

import (
	"github.com/ZYunH/goutils"
	"sync"
)

type LRUCache struct {
	m       map[string]*node
	root    *node
	maxSize int
	hits    int
	misses  int

	lock        sync.RWMutex
	_buf        []byte
	_bufNodePtr *node
}

type node struct {
	key   string
	value interface{}
	prev  *node
	next  *node
}

func New(maxSize int) *LRUCache {
	if maxSize <= 0 {
		panic("maxSize must be greater than 0, use map instead of LRUCache in case maxSize == 0")
	}
	root := &node{}
	root.next = root
	root.prev = root
	return &LRUCache{m: make(map[string]*node, maxSize), root: root, _buf: make([]byte, 0, 64), maxSize: maxSize}
}

func deepCopyNode(n node) node {
	return node{key: n.key, value: n.value, prev: n.prev, next: n.next}
}

func (c *LRUCache) Set(key, value interface{}) bool {
	k := goutils.BytesToStringNew(InterfaceToBytesWithBuf(c._buf, key))

	c.lock.Lock()
	c._bufNodePtr = c.m[k]
	if c._bufNodePtr == nil { // This means the k not in the map
		if len(c.m) < c.maxSize-1 {
			// Cache is not full, insert a new node
			_node := &node{}
			_node.key = k
			_node.value = value
			_node.next = c.root
			_node.prev = c.root.prev
			c.m[k] = _node

			c.root.prev.next = _node
			c.root.prev = _node

			c.lock.Unlock()
			return false
		} else {
			// Cache is full, replace the oldest one with the new node,
			// in this case, we just replace the original root with the
			// new root, and makes the original root.next become new root.
			delete(c.m, c.root.key)
			c.root.key = k
			c.root.value = value
			c.m[k] = c.root
			c.root = c.root.next

			c.lock.Unlock()
			return true
		}
	} else {
		// Hits a key, we do nothing in this case, since the key and
		// value already exists.
	}
	c.lock.Unlock()
	return false
}

func (c *LRUCache) Get(key interface{}) (interface{}, bool) {
	k := goutils.BytesToString(InterfaceToBytesWithBuf(c._buf, key))
	c.lock.Lock()
	c._bufNodePtr = c.m[k]
	if c._bufNodePtr == nil {
		// This means the k not in the map
		c.misses++
		c.lock.Unlock()
		return nil, false
	} else {
		// Hits a key, drop it from the original location, and insert it
		// to the location between root.prev and root(The latest one in cache)
		c.hits++
		c._bufNodePtr.prev.next = c._bufNodePtr.next
		c._bufNodePtr.next.prev = c._bufNodePtr.prev
		c.root = c.root.next
		c._bufNodePtr.prev = c.root.prev
		c._bufNodePtr.next = c.root

		c.root.prev.next = c._bufNodePtr
		c.root.prev = c._bufNodePtr

		c.lock.Unlock()
		return c._bufNodePtr.value, true
	}
}
