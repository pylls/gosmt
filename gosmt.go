package gosmt

import (
	"bytes"
	"encoding/binary"
	"sort"
)

// constants (has to be var in Go, slices evaluated at runtime)
var (
	// Empty is an empty key.
	Empty = []byte{0x0}
	// Set is a set key.
	Set = []byte{0x1}
)

// D is our data structure to authenticate
// D [key][]byte: for each key value, D stores the leaf node value ([]byte)
type D [][]byte

// for sorting
func (d D) Len() int           { return len(d) }
func (d D) Swap(i, j int)      { d[i], d[j] = d[j], d[i] }
func (d D) Less(i, j int) bool { return bytes.Compare(d[i], d[j]) == -1 }

// Split splits d based on Split index s.
func (d D) Split(s []byte) (l, r D) {
	// the smallest index i where d[i] >= s
	i := sort.Search(d.Len(), func(i int) bool {
		return bytes.Compare(d[i], s) >= 0
	})
	return d[:i], d[i:]
}

// SMT is a sparse Merkle tree.
type SMT struct {
	c             []byte // tree-wide constant, an empty leaf will have a default value of hash(c)
	cache         Cache  // Cache interface could be implemented by different caching strategies
	Base          []byte // key of left-most leaf of a subtree, fixed in size.
	hash          func(data ...[]byte) []byte
	N             uint64   // output length, in bits, of hash
	defaultHashes [][]byte // [height][]byte, one default byte string per height (range:[0, N]), leaf node has height of 0, root node has height of N.
}

// NewSMT creates a new SMT. SMT instantiation requires a default empty leaf constant c, a caching strategy cache (e.g. CacheBranch, CacheBranchPlus), and a particular hash function (e.g. SHA256)
func NewSMT(c []byte, cache Cache, hash func(data ...[]byte) []byte) *SMT {
	s := new(SMT)
	s.cache = cache
	s.hash = hash
	s.c = c
	s.N = uint64(len(hash([]byte("smt"))) * 8) // hash any string to get output length
	s.Base = make([]byte, s.N/8)

	s.defaultHashes = make([][]byte, s.N+1)
	s.defaultHashes[0] = s.leafHash(Empty, nil)
	for i := 1; i <= int(s.N); i++ {
		s.defaultHashes[i] = hash(s.defaultHashes[i-1], s.defaultHashes[i-1])
	}
	return s
}

// Update updates keys to the value.
// TODO: split the type of d and keys
func (s *SMT) Update(d, keys D, height uint64, base, value []byte) []byte {
	if height == 0 {
		return s.leafHash(value, base)
	}
	split := bitSplit(base, s.N-height)
	ld, rd := d.Split(split)
	lkeys, rkeys := keys.Split(split)

	// When there's a key falling within the range of left/right subtree, meaning
	// a leaf node in left/right branch should be updated to a new value, then update
	// the root hash of left/right subtree recursively;
	// When no leaf node in left/right subtree shall be updated, then directly return
	// its root hash and update upwards recursively.
	switch {
	case lkeys.Len() == 0 && rkeys.Len() > 0:
		return s.cache.HashCache(s.RootHash(ld, height-1, base),
			s.Update(rd, keys, height-1, split, value),
			height, base, split, s.interiorHash, s.defaultHashes)
	case lkeys.Len() > 0 && rkeys.Len() == 0:
		return s.cache.HashCache(s.Update(ld, keys, height-1, base, value),
			s.RootHash(rd, height-1, split),
			height, base, split, s.interiorHash, s.defaultHashes)
	default:
		return s.cache.HashCache(s.Update(ld, lkeys, height-1, base, value),
			s.Update(rd, rkeys, height-1, split, value),
			height, base, split, s.interiorHash, s.defaultHashes)
	}
}

// AuditPath generates an audit path.
func (s *SMT) AuditPath(d D, height uint64, base, key []byte) [][]byte {
	if height == 0 {
		return nil
	}
	split := bitSplit(base, s.N-height)
	l, r := d.Split(split)

	if !bitIsSet(key, s.N-height) { // if k_j == 0
		return append(s.AuditPath(l, height-1, base, key),
			s.RootHash(r, height-1, split))
	}
	return append(s.AuditPath(r, height-1, split, key),
		s.RootHash(l, height-1, base))
}

// VerifyAuditPath verifies an audit path.
func (s *SMT) VerifyAuditPath(ap [][]byte, key, value, root []byte) bool {
	return bytes.Equal(root,
		s.auditPathCalc(ap, s.N, make([]byte, s.N/8), key, value))
}

func (s *SMT) auditPathCalc(ap [][]byte, height uint64,
	base, key, value []byte) []byte {
	if height == 0 {
		return s.leafHash(value, base)
	}
	split := bitSplit(base, s.N-height)
	if !bitIsSet(key, s.N-height) { // if k_j == 0
		return s.interiorHash(s.auditPathCalc(ap, height-1, base, key, value),
			ap[height-1], height, base)
	}
	return s.interiorHash(ap[height-1],
		s.auditPathCalc(ap, height-1, split, key, value), height, base)
}

// RootHash returns the root hash of a subtree with certain height.
func (s *SMT) RootHash(d D, height uint64, base []byte) []byte {
	switch {
	case s.cache.Exists(height, base):
		return s.cache.Get(height, base)
	case d.Len() == 0:
		return s.defaultHash(height)
	case d.Len() == 1 && height == 0:
		return s.leafHash(Set, base)
	case d.Len() > 0 && height == 0:
		panic("this should never happen (unsorted D or broken split?)")
	default:
		split := bitSplit(base, s.N-height)
		l, r := d.Split(split)
		return s.interiorHash(s.RootHash(l, height-1, base),
			s.RootHash(r, height-1, split), height, base)
	}
}

func (s *SMT) defaultHash(height uint64) []byte {
	return s.defaultHashes[height]
}

// leafHash returns the leaf value of SMT.
func (s *SMT) leafHash(a, base []byte) []byte {
	if bytes.Equal(a, Empty) {
		return s.hash(s.c)
	}
	return s.hash(s.c, base)
}

// interiorHash returns the non-leaf node value of SMT.
func (s *SMT) interiorHash(left, right []byte,
	height uint64, base []byte) []byte {
	if bytes.Equal(left, right) {
		return s.hash(left, right)
	}
	buf := new(bytes.Buffer)
	if binary.Write(buf, binary.BigEndian, height) != nil {
		panic("failed to encode height")
	}
	return s.hash(left, right, base, buf.Bytes())
}

// CacheEntries returns the number of cache entries.
func (s *SMT) CacheEntries() int {
	return s.cache.Entries()
}
