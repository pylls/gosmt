package main

import (
	"crypto/sha512"
	"fmt"
	"log"
	"math/rand"
	"testing"

	"github.com/pylls/balloon/hashtreap"
)

type run struct {
	f     func(b *testing.B)
	cache string
	size  int
}

var (
	maxSize         = 20
	hashOutputLen   = 32
	updateSize      = 256
	keyUpdateDSsize = 15
)

func main() {
	log.Printf("update time (ms) for keys in 2^%d HT:", keyUpdateDSsize)
	for i := uint(1); i <= uint(maxSize); i++ {
		result := testing.Benchmark(makeUpdateKeysBench(1<<uint(keyUpdateDSsize),
			1<<i))
		log.Printf("%d (%d), %f\n", i, 1<<i, toMS(result.NsPerOp()))
	}
	fmt.Println("")

	log.Printf("update time (ms) for %d keys:", keyUpdateDSsize)
	for i := uint(1); i <= uint(maxSize); i++ {
		result := testing.Benchmark(makeUpdateBench(1 << i))
		log.Printf("%d (%d), %f\n", i, 1<<i, toMS(result.NsPerOp()))
	}
	fmt.Println("")

	log.Println("membership queries:")
	for i := uint(1); i <= uint(maxSize); i++ {
		result := testing.Benchmark(makeAuditPathBench(1 << i))
		log.Printf("%d (%d), %f\n", i, 1<<i, toMS(result.NsPerOp()))
	}
	fmt.Println("")

	log.Println("size:")
	for i := uint(1); i <= uint(maxSize); i++ {
		size := 1 << i
		ht := hashtreap.NewHashTreap()
		for j := 0; j < size; j++ {
			k := randKey(make([]byte, hashOutputLen))
			n, err := ht.Add(k, k)
			if err != nil {
				panic(err)
			}
			ht = n
		}
		ht.Update()

		htsize := ht.Size() * (256*4 + 64*3)
		log.Printf("%d (%d), %.2f\n", i, 1<<i, float64(htsize)/float64(1024*1024))
	}
	fmt.Println("")
}

func toMS(ns int64) float64 {
	return float64(ns) / float64(1000*1000)
}

func makeAuditPathBench(size int) func(b *testing.B) {
	return func(b *testing.B) {
		// create HT
		ht := hashtreap.NewHashTreap()
		for i := 0; i < size; i++ {
			k := randKey(make([]byte, hashOutputLen))
			n, err := ht.Add(k, k)
			if err != nil {
				panic(err)
			}
			ht = n
		}
		ht.Update()

		key := randKey(make([]byte, hashOutputLen))
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			ht.MembershipQuery(key)
		}
	}
}

func makeUpdateBench(size int) func(b *testing.B) {
	return func(b *testing.B) {
		// create HT
		ht := hashtreap.NewHashTreap()
		for i := 0; i < size; i++ {
			k := randKey(make([]byte, hashOutputLen))
			n, err := ht.Add(k, k)
			if err != nil {
				panic(err)
			}
			ht = n
		}
		ht.Update()

		// create updateSize keys
		keys := make([][]byte, updateSize)
		for i := 0; i < updateSize; i++ {
			keys[i] = randKey(make([]byte, hashOutputLen))
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			tHT := ht
			for j := 0; j < len(keys); j++ {
				tHT, _ = tHT.Add(keys[j], keys[j])
			}
			tHT.Update()
		}
	}
}

func makeUpdateKeysBench(size, keyCount int) func(b *testing.B) {
	return func(b *testing.B) {
		// create HT
		ht := hashtreap.NewHashTreap()
		for i := 0; i < size; i++ {
			k := randKey(make([]byte, hashOutputLen))
			n, err := ht.Add(k, k)
			if err != nil {
				panic(err)
			}
			ht = n
		}
		ht.Update()

		// create keyCount keys
		keys := make([][]byte, keyCount)
		for i := 0; i < keyCount; i++ {
			keys[i] = randKey(make([]byte, hashOutputLen))
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			tHT := ht
			for j := 0; j < len(keys); j++ {
				tHT, _ = tHT.Add(keys[j], keys[j])
			}
			tHT.Update()
		}
	}
}

func randKey(key []byte) []byte {
	_, err := rand.Read(key)
	if err != nil {
		panic(err)
	}
	return key
}

func hash(data ...[]byte) []byte {
	hasher := sha512.New512_256()
	for i := 0; i < len(data); i++ {
		hasher.Write(data[i])
	}
	return hasher.Sum(nil)
}
