package gosmt

import (
	"bytes"
	"crypto/rand"
	"math/big"
	"strconv"
)

// Cache specifies a caching approach.
type Cache interface {
	Exists(height uint64, base []byte) bool
	Get(height uint64, base []byte) []byte
	HashCache(left, right []byte, height uint64, base, split []byte,
		interiorHash func(left, right []byte, height uint64, base []byte) []byte,
		defaultHashes [][]byte) []byte
	Entries() int
}

// CacheNothing caches nothing.
type CacheNothing int

// Exists checks if a value exists in the cache.
func (c CacheNothing) Exists(height uint64, base []byte) bool { return false }

// Get returns a value that exists from the cache.
func (c CacheNothing) Get(height uint64, base []byte) []byte { return nil }

// HashCache hashes the provided values and maybe caches.
func (c CacheNothing) HashCache(left, right []byte, height uint64, base, split []byte,
	interiorHash func(left, right []byte, height uint64, base []byte) []byte,
	defaultHashes [][]byte) []byte {
	return interiorHash(left, right, height, base)
}

// Entries returns the number of entries in the cache.
func (c CacheNothing) Entries() int {
	return 0
}

// CacheBranch caches every branch where both children have non-default values.
type CacheBranch map[string][]byte

// Exists checks if a value exists in the cache.
func (c CacheBranch) Exists(height uint64, base []byte) bool {
	_, exists := c[strconv.Itoa(int(height))+string(base)]
	return exists
}

// Get returns a value that exists from the cache.
func (c CacheBranch) Get(height uint64, base []byte) []byte {
	return c[strconv.Itoa(int(height))+string(base)]
}

// HashCache hashes the provided values and maybe caches.
func (c CacheBranch) HashCache(left, right []byte, height uint64, base, split []byte,
	interiorHash func(left, right []byte, height uint64, base []byte) []byte,
	defaultHashes [][]byte) []byte {
	h := interiorHash(left, right, height, base)
	if !bytes.Equal(defaultHashes[height-1], left) && !bytes.Equal(defaultHashes[height-1], right) {
		c[strconv.Itoa(int(height))+string(base)] = h
	} else {
		delete(c, strconv.Itoa(int(height))+string(base))
	}
	return h
}

// Entries returns the number of entries in the cache.
func (c CacheBranch) Entries() int {
	return len(c)
}

// CacheBranchPlus caches the two children of every branch where both children have non-default values.
type CacheBranchPlus map[string][]byte

// Exists checks if a value exists in the cache.
func (c CacheBranchPlus) Exists(height uint64, base []byte) bool {
	_, exists := c[strconv.Itoa(int(height))+string(base)]
	return exists
}

// Get returns a value that exists from the cache.
func (c CacheBranchPlus) Get(height uint64, base []byte) []byte {
	return c[strconv.Itoa(int(height))+string(base)]
}

// HashCache hashes the provided values and maybe caches.
func (c CacheBranchPlus) HashCache(left, right []byte, height uint64, base, split []byte,
	interiorHash func(left, right []byte, height uint64, base []byte) []byte,
	defaultHashes [][]byte) []byte {
	h := interiorHash(left, right, height, base)

	if !bytes.Equal(defaultHashes[height-1], left) && !bytes.Equal(defaultHashes[height-1], right) {
		c[strconv.Itoa(int(height-1))+string(base)] = left
		c[strconv.Itoa(int(height-1))+string(split)] = right
	} else {
		delete(c, strconv.Itoa(int(height-1))+string(base))
		delete(c, strconv.Itoa(int(height-1))+string(split))
	}

	return h
}

// Entries returns the number of entries in the cache.
func (c CacheBranchPlus) Entries() int {
	return len(c)
}

// CacheBranchMinus caches every branch where both children have non-default values with the
// provided probability [0,1]).
type CacheBranchMinus struct {
	data        map[string][]byte
	probability float64
}

// NewCacheBranchMinus creates a new CacheBranchMinus with the provided caching probability.
func NewCacheBranchMinus(prob float64) *CacheBranchMinus {
	c := new(CacheBranchMinus)
	c.data = make(map[string][]byte)
	c.probability = prob
	return c
}

// Exists checks if a value exists in the cache.
func (c CacheBranchMinus) Exists(height uint64, base []byte) bool {
	_, exists := c.data[strconv.Itoa(int(height))+string(base)]
	return exists
}

// Get returns a value that exists from the cache.
func (c CacheBranchMinus) Get(height uint64, base []byte) []byte {
	return c.data[strconv.Itoa(int(height))+string(base)]
}

// HashCache hashes the provided values and maybe caches.
func (c CacheBranchMinus) HashCache(left, right []byte, height uint64, base, split []byte,
	interiorHash func(left, right []byte, height uint64, base []byte) []byte,
	defaultHashes [][]byte) []byte {
	h := interiorHash(left, right, height, base)
	if randLess(c.probability) &&
		!bytes.Equal(defaultHashes[height-1], left) && !bytes.Equal(defaultHashes[height-1], right) {
		c.data[strconv.Itoa(int(height))+string(base)] = h
	} else {
		delete(c.data, strconv.Itoa(int(height))+string(base))
	}
	return h
}

// Entries returns the number of entries in the cache.
func (c CacheBranchMinus) Entries() int {
	return len(c.data)
}

func randLess(x float64) bool {
	b, err := rand.Int(rand.Reader, big.NewInt(100))
	if err != nil {
		panic(err)
	}
	return float64(b.Int64())/float64(100) < x
}
