package lpmtrie

// inspired by: https://github.com/torvalds/linux/blob/master/kernel/bpf/lpm_trie.c

import (
	"encoding/binary"
	"errors"
	"math/bits"
	"sync/atomic"
	"unsafe"
)

const (
	MaxPrefixLenIPv4 = 32
	MaxPrefixLenIPv6 = 128
)

// Key is the key of the trie.
// A key is a byte array with a prefix length.
// The prefix length is the number of bits for the key.
// The length of the byte array must be same with eighth of trie's max prefix length.
type Key struct {
	PrefixLen int
	Data      []byte
}

// LpmTrie is a trie data structure which implements Longest Prefix Match algorithm.
type LpmTrie interface {
	// Size returns the number of entries in the trie.
	Size() int64

	// Lookup lookups the value of the key by LPM algo.
	// The length of key's data must be same with eighth of trie's max prefix length,
	// or it will panic.
	Lookup(key Key) (interface{}, bool)

	// Update updates the value of the key by LPM algo.
	// If the key is not found, it inserts the key-value pair.
	// The length of key's data must be same with eighth of trie's max prefix length,
	// or it will panic.
	Update(key Key, val interface{}) (updated bool)

	// Delete deletes the key-value pair by LPM algo.
	// The length of key's data must be same with eighth of trie's max prefix length,
	// or it will panic.
	Delete(key Key) (deleted bool)

	// Range iterates over the key-value pairs in the trie by in-order.
	Range(fn func(key Key, val interface{}) bool)
}

type nodeValue struct {
	v interface{}
}

var prunedValue = nodeValue{}

type lpmTrieNode struct {
	Key
	child [2]unsafe.Pointer // *lpmTrieNode
	value atomic.Value
	im    atomic.Value
}

var nilNode = (*lpmTrieNode)(nil)

func newLpmTrieNode(key Key, value interface{}) *lpmTrieNode {
	var n lpmTrieNode
	n.Key = key
	n.storeValue(nodeValue{v: value})
	storePointer(&n.child[0], nilNode)
	storePointer(&n.child[1], nilNode)
	n.im.Store(false)
	return &n
}

func (n *lpmTrieNode) storeValue(v nodeValue) {
	n.value.Store(v)
}

func (n *lpmTrieNode) loadValue() interface{} {
	return n.value.Load().(nodeValue).v
}

// isIm returns if the node is intermediate one.
func (n *lpmTrieNode) isIm() bool {
	return n.im.Load().(bool)
}

func (n *lpmTrieNode) setIm(v bool) {
	n.im.Store(v)
}

func loadPointer(ptr *unsafe.Pointer) *lpmTrieNode {
	return (*lpmTrieNode)(atomic.LoadPointer(ptr))
}

func storePointer(ptr *unsafe.Pointer, node *lpmTrieNode) {
	atomic.StorePointer(ptr, unsafe.Pointer(node))
}

type lpmTrie struct {
	root         unsafe.Pointer // *lpmTrieNode
	size         int64
	maxPrefixLen int
	keySize      int
}

var _ LpmTrie = (*lpmTrie)(nil)

func New(maxPrefixLen int) (LpmTrie, error) {
	if maxPrefixLen <= 0 || maxPrefixLen%8 != 0 {
		return nil, errors.New("maxPrefixLen must be positive and be times of 8")
	}

	var t lpmTrie
	storePointer(&t.root, nilNode)
	t.maxPrefixLen = maxPrefixLen
	t.keySize = maxPrefixLen / 8
	return &t, nil
}

func (t *lpmTrie) Size() int64 {
	return atomic.LoadInt64(&t.size)
}

func (t *lpmTrie) isValidKey(key Key) bool {
	return 0 <= key.PrefixLen && key.PrefixLen <= t.maxPrefixLen && len(key.Data) == t.keySize
}

func (t *lpmTrie) checkKey(key Key) {
	if !t.isValidKey(key) {
		panic("lpmtrie: invalid key")
	}
}

func extractBit(data []byte, index int) byte {
	return (data[index/8] >> (7 - (index % 8))) & 0x01
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (t *lpmTrie) longestPrefixMatch(node *lpmTrieNode, key Key) int {
	limit := min(node.PrefixLen, key.PrefixLen)
	prefixlen, i := 0, 0
	be := binary.BigEndian

	for ; t.keySize >= i+4; i += 4 {
		diff := be.Uint32(node.Data[i:i+4]) ^ be.Uint32(key.Data[i:i+4])

		prefixlen += bits.LeadingZeros32(diff)
		if prefixlen >= limit {
			return limit
		}
		if diff != 0 {
			return prefixlen
		}
	}

	if t.keySize >= i+2 {
		diff := be.Uint16(node.Data[i:i+2]) ^ be.Uint16(key.Data[i:i+2])

		prefixlen += bits.LeadingZeros16(diff)
		if prefixlen >= limit {
			return limit
		}
		if diff != 0 {
			return prefixlen
		}

		i += 2
	}

	if t.keySize >= i+1 {
		diff := node.Data[i] ^ key.Data[i]

		prefixlen += bits.LeadingZeros8(diff)
		if prefixlen >= limit {
			return limit
		}
	}

	return prefixlen
}

func (t *lpmTrie) Lookup(key Key) (interface{}, bool) {
	t.checkKey(key)

	var found *lpmTrieNode

	for node := loadPointer(&t.root); node != nil; node = loadPointer(&node.child[extractBit(key.Data, node.PrefixLen)]) {
		matchlen := t.longestPrefixMatch(node, key)
		if matchlen == t.maxPrefixLen {
			return node.loadValue(), true
		}

		if matchlen < node.PrefixLen {
			break
		}

		if !node.isIm() {
			found = node
		}
	}

	if found == nil {
		return nil, false
	}

	return found.loadValue(), true
}

func (t *lpmTrie) Update(key Key, val interface{}) (updated bool) {
	t.checkKey(key)

	atomic.AddInt64(&t.size, 1)

	newnode := newLpmTrieNode(key, val)
	slot := &t.root

	matchlen := 0
	node := loadPointer(slot)
	for ; node != nil; node = loadPointer(slot) {
		matchlen = t.longestPrefixMatch(node, key)
		if node.PrefixLen != matchlen ||
			node.PrefixLen == key.PrefixLen ||
			node.PrefixLen == t.maxPrefixLen {
			break
		}

		slot = &node.child[extractBit(key.Data, node.PrefixLen)]
	}

	if node == nil {
		storePointer(slot, newnode)
		return false
	}

	if node.PrefixLen == matchlen {
		newnode.child = node.child

		if !node.isIm() {
			atomic.AddInt64(&t.size, -1)
		}

		storePointer(slot, newnode)
		return true
	}

	imNode := newLpmTrieNode(key, nil)
	imNode.PrefixLen = matchlen
	imNode.setIm(true)

	nextBit := extractBit(key.Data, matchlen)
	if nextBit != 0 {
		storePointer(&imNode.child[0], node)
		storePointer(&imNode.child[1], newnode)
	} else {
		storePointer(&imNode.child[0], newnode)
		storePointer(&imNode.child[1], node)
	}

	storePointer(slot, imNode)

	return false
}

func (t *lpmTrie) Delete(key Key) (deleted bool) {
	t.checkKey(key)

	var parent *lpmTrieNode
	trim := &t.root
	trim2 := trim
	matchlen := 0
	node := loadPointer(trim)
	for ; node != nil; node = loadPointer(trim) {
		matchlen = t.longestPrefixMatch(node, key)

		if node.PrefixLen != matchlen ||
			node.PrefixLen == key.PrefixLen {
			break
		}

		parent = node
		trim2 = trim
		trim = &node.child[extractBit(key.Data, node.PrefixLen)]
	}

	if node == nil ||
		node.PrefixLen != key.PrefixLen ||
		node.PrefixLen != matchlen ||
		node.isIm() {
		return false
	}

	atomic.AddInt64(&t.size, -1)

	if loadPointer(&node.child[0]) != nil &&
		loadPointer(&node.child[1]) != nil {
		node.storeValue(prunedValue) // Note: free the value
		node.setIm(true)             // mark as intermediate node
		return true
	}

	if parent != nil &&
		parent.isIm() &&
		loadPointer(&node.child[0]) == nil &&
		loadPointer(&node.child[1]) == nil {
		if node == loadPointer(&parent.child[0]) {
			storePointer(trim2, loadPointer(&parent.child[1]))
		} else {
			storePointer(trim2, loadPointer(&parent.child[0]))
		}
		return true
	}

	if loadPointer(&node.child[0]) != nil {
		storePointer(trim, loadPointer(&node.child[0]))
	} else if loadPointer(&node.child[1]) != nil {
		storePointer(trim, loadPointer(&node.child[1]))
	} else {
		storePointer(trim, nil)
	}
	return true
}

func (t *lpmTrie) Range(fn func(key Key, val interface{}) bool) {
	_ = t.traverse(&t.root, fn)
}

func (t *lpmTrie) traverse(root *unsafe.Pointer, fn func(key Key, val interface{}) bool) (terminated bool) {
	node := loadPointer(root)
	if node == nil {
		return false
	}

	if t.traverse(&node.child[0], fn) {
		return true
	}

	if !node.isIm() && !fn(node.Key, node.loadValue()) {
		return true
	}

	return t.traverse(&node.child[1], fn)
}
