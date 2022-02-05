package lpmtrie

import (
	"bytes"
	"testing"
)

func TestNilNode(t *testing.T) {
	if nilNode != nil {
		t.Errorf("nilNode should be nil")
	}
}

func TestExtractBit(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		bit  int
	}{
		{
			"bit on 0",
			[]byte{0b10000000, 0, 0, 0},
			0,
		},
		{
			"bit on 1",
			[]byte{0b01000000, 0, 0, 0},
			1,
		},
		{
			"bit on 2",
			[]byte{0b00100000, 0, 0, 0},
			2,
		},
		{
			"bit on 3",
			[]byte{0b00010000, 0, 0, 0},
			3,
		},
		{
			"bit on 4",
			[]byte{0b00001000, 0, 0, 0},
			4,
		},
		{
			"bit on 5",
			[]byte{0b00000100, 0, 0, 0},
			5,
		},
		{
			"bit on 6",
			[]byte{0b00000010, 0, 0, 0},
			6,
		},
		{
			"bit on 7",
			[]byte{0b00000001, 0, 0, 0},
			7,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if b := extractBit(tt.data, tt.bit); b != 1 {
				t.Errorf("expected bit %d to be 1, got %d", tt.bit, b)
			}
		})
	}
}

func TestLongestPrefixMatch(t *testing.T) {
	key1 := Key{PrefixLen: 32}
	key2 := Key{PrefixLen: 32}
	node1 := lpmTrieNode{Key: key1}

	lt, _ := New(key1.PrefixLen)
	trie := lt.(*lpmTrie)

	tests := []struct {
		name            string
		data1           []byte
		data2           []byte
		expectPrefixlen int
	}{
		{
			"two zero",
			[]byte{0, 0, 0, 0},
			[]byte{0, 0, 0, 0},
			32,
		},
		{
			"0 prefixlen",
			[]byte{0, 0, 0, 0},
			[]byte{0b10000000, 0, 0, 0},
			0,
		},
		{
			"1 prefixlen",
			[]byte{0b11000000, 0, 0, 0},
			[]byte{0b10000000, 0, 0, 0},
			1,
		},
		{
			"2 prefixlen",
			[]byte{0b10100000, 0, 0, 0},
			[]byte{0b10000000, 0, 0, 0},
			2,
		},
		{
			"3 prefixlen",
			[]byte{0b10010000, 0, 0, 0},
			[]byte{0b10000000, 0, 0, 0},
			3,
		},
		{
			"4 prefixlen",
			[]byte{0b10001000, 0, 0, 0},
			[]byte{0b10000000, 0, 0, 0},
			4,
		},
		{
			"5 prefixlen",
			[]byte{0b10000100, 0, 0, 0},
			[]byte{0b10000000, 0, 0, 0},
			5,
		},
		{
			"6 prefixlen",
			[]byte{0b10000010, 0, 0, 0},
			[]byte{0b10000000, 0, 0, 0},
			6,
		},
		{
			"7 prefixlen",
			[]byte{0b10000001, 0, 0, 0},
			[]byte{0b10000000, 0, 0, 0},
			7,
		},
		{
			"8 prefixlen",
			[]byte{0, 0, 0, 0},
			[]byte{0, 0b10000000, 0, 0},
			8,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key1.Data = tt.data1
			key2.Data = tt.data2
			node1.Key = key1

			if pl := trie.longestPrefixMatch(&node1, key2); pl != tt.expectPrefixlen {
				t.Errorf("expected longest prefix match to be %d, got %d", tt.expectPrefixlen, pl)
			}
		})
	}
}

func TestLPMandExtractBit(t *testing.T) {
	const plen = 32
	var trie *lpmTrie

	reset := func() {
		lt, _ := New(plen)
		trie = lt.(*lpmTrie)
	}
	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			"the first bit",
			func(t *testing.T) {
				key1 := Key{plen, []byte{0b10000000, 0, 0, 0}}
				key2 := Key{plen, []byte{0b01000000, 0, 0, 0}}

				key := Key{plen, []byte{0b11000000, 0, 0, 0}}
				node := lpmTrieNode{Key: key}

				matchlen1 := trie.longestPrefixMatch(&node, key1)
				if matchlen1 != 1 {
					t.Fatalf("expected longest prefix match1 to be 1, got %d", matchlen1)
				}
				matchlen2 := trie.longestPrefixMatch(&node, key2)
				if matchlen2 != 0 {
					t.Fatalf("expected longest prefix match2 to be 0, got %d", matchlen2)
				}

				if b := extractBit(key1.Data, matchlen1); b != 0 {
					t.Fatalf("key1, expected bit %d to be 0, got %d", matchlen1, b)
				}

				if b := extractBit(key2.Data, matchlen2); b != 0 {
					t.Fatalf("key2, expected bit %d to be 0, got %d", matchlen2, b)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reset()
			tt.run(t)
		})
	}
}

func printTrie(trie *lpmTrie, t *testing.T) {
	trie.Range(func(key Key, val interface{}) bool {
		t.Logf("%+v: %+v\n", key, val)
		return true
	})
}

var _ = printTrie

func TestLookup(t *testing.T) {
	const plen = 32
	var trie *lpmTrie

	reset := func() {
		lt, _ := New(plen)
		trie = lt.(*lpmTrie)
	}

	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			"empty",
			func(t *testing.T) {
				_, ok := trie.Lookup(Key{PrefixLen: plen, Data: []byte{0, 0, 0, 0}})
				if ok {
					t.Errorf("expected lookup to fail")
				}
			},
		},
		{
			"one node",
			func(t *testing.T) {
				key := Key{PrefixLen: plen, Data: []byte{0, 0, 0, 0}}
				node := lpmTrieNode{Key: key}
				trie.Update(key, &node)

				_, ok := trie.Lookup(key)
				if !ok {
					t.Errorf("expected lookup to succeed")
				}
			},
		},
		{
			"two nodes",
			func(t *testing.T) {
				key := Key{PrefixLen: plen, Data: []byte{0, 0, 0, 0}}
				trie.Update(key, 1)
				key = Key{plen, []byte{0b10000000, 0, 0, 0}}
				trie.Update(key, 2)

				// printTrie(trie, t)

				v, ok := trie.Lookup(key)
				if !ok || v.(int) != 2 {
					t.Errorf("expected lookup to succeed")
				}

				key.Data = []byte{0, 0, 0, 0}
				v, ok = trie.Lookup(key)
				if !ok || v.(int) != 1 {
					t.Errorf("expected lookup to succeed")
				}

				key.Data = []byte{0b01000000, 0, 0, 0}
				_, ok = trie.Lookup(key)
				if ok {
					t.Errorf("expected lookup to fail")
				}
			},
		},
		{
			"intermediate node",
			func(t *testing.T) {
				key := Key{plen, []byte{0b10100000, 0, 0, 0}}
				trie.Update(key, 1)
				key = Key{plen, []byte{0b00100000, 0, 0, 0}}
				trie.Update(key, 2)

				key = Key{plen, []byte{0b11000000, 0, 0, 0}}
				_, ok := trie.Lookup(key)
				if ok {
					t.Errorf("expected lookup to fail")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reset()
			tt.run(t)
		})
	}
}

func TestUpdate(t *testing.T) {
	const plen = 32
	var trie *lpmTrie

	reset := func() {
		lt, _ := New(plen)
		trie = lt.(*lpmTrie)
	}

	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			"empty",
			func(t *testing.T) {
				rt := loadPointer(&trie.root)
				if rt != nil {
					t.Errorf("expected root to be nil")
				}
			},
		},
		{
			"root node",
			func(t *testing.T) {
				key := Key{PrefixLen: plen, Data: []byte{0, 0, 0, 0}}
				trie.Update(key, 1)

				rt := loadPointer(&trie.root)
				if rt == nil || rt.loadValue().(int) != 1 {
					t.Errorf("expected root node to be 1")
				}
			},
		},
		{
			"intermediate node",
			func(t *testing.T) {
				key := Key{plen, []byte{0b10100000, 0, 0, 0}}
				trie.Update(key, 1)
				key = Key{plen, []byte{0b00100000, 0, 0, 0}}
				trie.Update(key, 2)

				rt := loadPointer(&trie.root)
				if !rt.isIm() {
					t.Errorf("expected root to be intermediate")
				}
				if rt.loadValue() != nil {
					t.Errorf("expected root's value to be nil")
				}

				left := loadPointer(&rt.child[0])
				if left == nil || left.loadValue().(int) != 2 {
					t.Errorf("expected left child to be 2")
				}

				right := loadPointer(&rt.child[1])
				if right == nil || right.loadValue().(int) != 1 {
					t.Errorf("expected right child to be 1")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reset()
			tt.run(t)
		})
	}
}

func TestDelete(t *testing.T) {
	const plen = 32
	var trie *lpmTrie

	reset := func() {
		lt, _ := New(plen)
		trie = lt.(*lpmTrie)
	}

	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			"empty",
			func(t *testing.T) {
				key := Key{PrefixLen: plen, Data: []byte{0, 0, 0, 0}}
				deleted := trie.Delete(key)
				if deleted {
					t.Errorf("expected delete to fail")
				}
			},
		},
		{
			"root node",
			func(t *testing.T) {
				key := Key{PrefixLen: plen, Data: []byte{0, 0, 0, 0}}
				trie.Update(key, 1)

				deleted := trie.Delete(key)
				if !deleted {
					t.Fatalf("expected delete to succeed")
				}

				rt := loadPointer(&trie.root)
				if rt != nil {
					t.Errorf("expected root node to be nil")
				}
			},
		},
		{
			"intermediate node",
			func(t *testing.T) {
				key := Key{plen, []byte{0b10100000, 0, 0, 0}}
				trie.Update(key, 1)
				key = Key{plen, []byte{0b00100000, 0, 0, 0}}
				trie.Update(key, 2)

				// printTrie(trie, t)

				rt := loadPointer(&trie.root)
				if !rt.isIm() {
					t.Errorf("expected root to be intermediate")
				}

				key = Key{plen, []byte{0b11000000, 0, 0, 0}}
				deleted := trie.Delete(key)
				if deleted {
					t.Errorf("expected delete to fail")
				}
			},
		},
		{
			"one node",
			func(t *testing.T) {
				key := Key{plen, []byte{0b10100000, 0, 0, 0}}
				trie.Update(key, 1)
				key = Key{plen, []byte{0b00100000, 0, 0, 0}}
				trie.Update(key, 2)

				deleted := trie.Delete(key)
				if !deleted {
					t.Errorf("expected delete to succeed")
				}

				rt := loadPointer(&trie.root)
				if loadPointer(&rt.child[0]) != nil {
					t.Errorf("expected left child to be nil")
				}
			},
		},
		{
			"two nodes",
			func(t *testing.T) {
				key := Key{plen, []byte{0b10100000, 0, 0, 0}}
				trie.Update(key, 1)
				key = Key{plen, []byte{0b00100000, 0, 0, 0}}
				trie.Update(key, 2)

				deleted := trie.Delete(key)
				if !deleted {
					t.Errorf("expected delete to succeed")
				}

				rt := loadPointer(&trie.root)
				if rt.loadValue().(int) != 1 {
					t.Fatalf("expected root node to be 1")
				}

				key = Key{plen, []byte{0b10100000, 0, 0, 0}}
				deleted = trie.Delete(key)
				if !deleted {
					t.Errorf("expected delete to succeed")
				}

				rt = loadPointer(&trie.root)
				if rt != nil {
					t.Errorf("expected root node to be nil")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reset()
			tt.run(t)
		})
	}
}

func TestSize(t *testing.T) {
	const plen = 32
	var trie *lpmTrie

	reset := func() {
		lt, _ := New(plen)
		trie = lt.(*lpmTrie)
	}

	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			"empty",
			func(t *testing.T) {
				if trie.Size() != 0 {
					t.Errorf("expected size to be 0")
				}
			},
		},
		{
			"one node",
			func(t *testing.T) {
				key := Key{PrefixLen: plen, Data: []byte{0, 0, 0, 0}}
				trie.Update(key, 1)

				if trie.Size() != 1 {
					t.Errorf("expected size to be 1")
				}
			},
		},
		{
			"two nodes",
			func(t *testing.T) {
				key := Key{plen, []byte{0b10100000, 0, 0, 0}}
				trie.Update(key, 1)
				key = Key{plen, []byte{0b11110000, 0, 0, 0}}
				trie.Update(key, 2)

				if trie.Size() != 2 {
					t.Errorf("expected size to be 2")
				}
			},
		},
		{
			"intermediate node",
			func(t *testing.T) {
				key := Key{plen, []byte{0b10100000, 0, 0, 0}}
				trie.Update(key, 1)
				key = Key{plen, []byte{0b00100000, 0, 0, 0}}
				trie.Update(key, 2)

				if trie.Size() != 2 {
					t.Errorf("expected size to be 2")
				}
			},
		},
		{
			"delete one node",
			func(t *testing.T) {
				key := Key{PrefixLen: plen, Data: []byte{0, 0, 0, 0}}
				trie.Update(key, 1)

				if trie.Size() != 1 {
					t.Errorf("expected size to be 1")
				}

				trie.Delete(key)
				if trie.Size() != 0 {
					t.Errorf("expected size to be 0")
				}
			},
		},
		{
			"delete two nodes",
			func(t *testing.T) {
				key := Key{plen, []byte{0b10100000, 0, 0, 0}}
				trie.Update(key, 1)
				key = Key{plen, []byte{0b11110000, 0, 0, 0}}
				trie.Update(key, 2)

				if trie.Size() != 2 {
					t.Errorf("expected size to be 2")
				}

				trie.Delete(key)
				if trie.Size() != 1 {
					t.Errorf("expected size to be 1")
				}

				key = Key{plen, []byte{0b10100000, 0, 0, 0}}
				trie.Delete(key)
				if trie.Size() != 0 {
					t.Errorf("expected size to be 0")
				}
			},
		},
		{
			"delete intermediate node",
			func(t *testing.T) {
				key := Key{plen, []byte{0b10100000, 0, 0, 0}}
				trie.Update(key, 1)
				key = Key{plen, []byte{0b11100000, 0, 0, 0}}
				trie.Update(key, 2)

				if trie.Size() != 2 {
					t.Errorf("expected size to be 2")
				}

				key = Key{plen, []byte{0b10000000, 0, 0, 0}}
				trie.Delete(key)
				if trie.Size() != 2 {
					t.Errorf("expected size to be 2")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reset()
			tt.run(t)
		})
	}
}

func TestRange(t *testing.T) {
	const plen = 32
	var trie *lpmTrie

	reset := func() {
		lt, _ := New(plen)
		trie = lt.(*lpmTrie)
	}

	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			"empty",
			func(t *testing.T) {
				trie.Range(func(key Key, val interface{}) bool {
					t.Errorf("expected no keys")
					return true
				})
			},
		},
		{
			"one node",
			func(t *testing.T) {
				key := Key{plen, []byte{0b10100000, 0, 0, 0}}
				trie.Update(key, 1)

				trie.Range(func(key Key, val interface{}) bool {
					if key.PrefixLen != plen || !bytes.Equal(key.Data, []byte{0b10100000, 0, 0, 0}) {
						t.Errorf("expected key to be %v", key)
					}
					if val.(int) != 1 {
						t.Errorf("expected value to be 1")
					}
					return true
				})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reset()
			tt.run(t)
		})
	}
}
