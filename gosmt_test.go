package gosmt

import (
	"bytes"
	"math/rand"
	"sort"
	"testing"
)

func TestCaching(t *testing.T) {
	rounds := 4
	roundSize := 12

	var s []*SMT

	s = append(s, NewSMT([]byte{0x42}, CacheNothing(1), hash))
	s = append(s, NewSMT([]byte{0x42}, NewCacheBranchMinus(0.5), hash))
	s = append(s, NewSMT([]byte{0x42},
		CacheBranch(make(map[string][]byte)), hash))
	s = append(s, NewSMT([]byte{0x42},
		CacheBranchPlus(make(map[string][]byte)), hash))

	var data D
	var keys Key

	// for each round,
	// - first add roundSize keys,
	// - then remove roundSize/2 random keys,
	// checking that each approach to caching is consistent
	for round := 0; round < rounds; round++ {
		roots := make([][]byte, len(s))

		keys = getFreshData(roundSize)
		data = append(data, keys...)
		sort.Sort(D(data))

		for i := 0; i < len(s); i++ {
			// update, then make sure we get the same root from RootHash
			roots[i] = s[i].Update(data, keys, s[i].N, s[i].Base, Set)
			r := s[i].RootHash(data, s[i].N, s[i].Base)
			if !bytes.Equal(roots[i], r) {
				t.Fatal("roots mismatch")
			}

			key := hash([]byte("non-member"))
			ap := s[i].AuditPath(data, s[i].N, s[i].Base, key)
			if !s[i].VerifyAuditPath(ap, key, Empty, roots[i]) {
				t.Fatalf("failed to verify valid proof")
			}
		}

		// make sure all caching strategies produce the same root
		for i := 1; i < len(s); i++ {
			if !bytes.Equal(roots[i], roots[i-1]) {
				t.Fatalf("roots mismatch")
			}

			if s[i].CacheEntries() <= s[0].CacheEntries() {
				t.Fatal("only CacheNothing should have no entries")
			}
		}

		// remove n random keys from the tail of data (we know they're (still) sorted)
		n := roundSize / 2
		removedKeys := Key(data[len(data)-n:])
		data = data[:len(data)-n]

		for i := 0; i < len(s); i++ {
			// update, then make sure we get the same root from RootHash
			roots[i] = s[i].Update(data, removedKeys, s[i].N, s[i].Base, Empty)
			r := s[i].RootHash(data, s[i].N, s[i].Base)
			if !bytes.Equal(roots[i], r) {
				t.Fatalf("roots mismatch for i = %d", i)
			}

			key := hash([]byte("non-member"))
			ap := s[i].AuditPath(data, s[i].N, s[i].Base, key)
			if !s[i].VerifyAuditPath(ap, key, Empty, roots[i]) {
				t.Fatalf("failed to verify valid proof")
			}
		}

		// make sure all caching strategies produce the same root
		for i := 1; i < len(s); i++ {
			if !bytes.Equal(roots[i], roots[i-1]) {
				t.Fatalf("roots mismatch")
			}

			if s[i].CacheEntries() <= s[0].CacheEntries() {
				t.Fatal("only CacheNothing should have no entries")
			}
		}
	}
}

func getFreshData(size int) Key {
	var data Key
	for i := 0; i < size; i++ {
		key := make([]byte, 32)
		_, err := rand.Read(key)
		if err != nil {
			panic(err)
		}
		data = append(data, hash(key))
	}
	sort.Sort(Key(data))
	return data
}
