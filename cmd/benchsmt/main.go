package main

import (
	"crypto/sha512"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"sort"
	"strconv"
	"testing"
	"time"

	"github.com/AlexXiong97/gosmt"
	"github.com/montanaflynn/stats"
)

type run struct {
	f     func(b *testing.B)
	s     func() string
	cache string
}

var (
	maxSMT          = 20
	keyUpdateDSsize = 15
	bMin            = 0.5
	bMax            = 0.9
	bDelta          = 0.1
	updateSize      = 256
	filename        = "benchsmt." + time.Now().String()
	data            []gosmt.D
	repeat          = 4
)

type res struct {
	index int
	ms    float64
}

func main() {
	flag.Parse()
	if len(flag.Args()) == 0 {
		log.Fatal("need to specify maximum size of SMT")
	}
	m, err := strconv.Atoi(flag.Arg(0))
	if err != nil {
		log.Fatal("the first argument has to be an int")
	}
	maxSMT = m
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	// generate testdata only once
	for i := uint(1); i <= uint(maxSMT); i++ {
		var d gosmt.D
		size := 1 << i
		for j := 0; j < size; j++ {
			d = append(d, hash(randKey(make([]byte, 32))))
		}
		sort.Sort(gosmt.D(d))
		data = append(data, d)
	}

	var benchAP []run
	for i := 0; i < maxSMT; i++ {
		for j := bMin; j <= bMax; j += bDelta {
			benchAP = append(benchAP, run{
				f: makeAuditPathBench(data[i],
					gosmt.NewCacheBranchMinus(j)),
				cache: fmt.Sprintf("B-%.1f", j),
			})
		}
		benchAP = append(benchAP, run{
			f: makeAuditPathBench(data[i],
				gosmt.CacheBranch(make(map[string][]byte))),
			cache: "B",
		})
		benchAP = append(benchAP, run{
			f: makeAuditPathBench(data[i],
				gosmt.CacheBranchPlus(make(map[string][]byte))),
			cache: "B+",
		})
	}

	var benchUpdate []run
	for i := 0; i < maxSMT; i++ {
		for j := bMin; j <= bMax; j += bDelta {
			benchUpdate = append(benchUpdate, run{
				f:     makeUpdateBench(data[i], gosmt.NewCacheBranchMinus(j)),
				cache: fmt.Sprintf("B-%.1f", j),
			})
		}
		benchUpdate = append(benchUpdate, run{
			f: makeUpdateBench(data[i],
				gosmt.CacheBranch(make(map[string][]byte))),
			cache: "B",
		})
		benchUpdate = append(benchUpdate, run{
			f: makeUpdateBench(data[i],
				gosmt.CacheBranchPlus(make(map[string][]byte))),
			cache: "B+",
		})
	}

	var benchUpdateKey []run
	for i := 0; i < maxSMT; i++ {
		size := 1 << uint(i+1)
		for j := bMin; j <= bMax; j += bDelta {
			benchUpdateKey = append(benchUpdateKey, run{
				f: makeUpdateKeyBench(size, data[keyUpdateDSsize-1],
					gosmt.NewCacheBranchMinus(j)),
				cache: fmt.Sprintf("B-%.1f", j),
			})
		}
		benchUpdateKey = append(benchUpdateKey, run{
			f: makeUpdateKeyBench(size, data[keyUpdateDSsize-1],
				gosmt.CacheBranch(make(map[string][]byte))),
			cache: "B",
		})
		benchUpdateKey = append(benchUpdateKey, run{
			f: makeUpdateKeyBench(size, data[keyUpdateDSsize-1],
				gosmt.CacheBranchPlus(make(map[string][]byte))),
			cache: "B+",
		})
	}

	var benchCacheSize []run
	for i := 0; i < maxSMT; i++ {
		for j := bMin; j <= bMax; j += bDelta {
			benchCacheSize = append(benchCacheSize, run{
				s: makeCacheSizeBench(data[i],
					gosmt.NewCacheBranchMinus(j)),
				cache: fmt.Sprintf("B-%.1f", j),
			})
		}
		benchCacheSize = append(benchCacheSize, run{
			s: makeCacheSizeBench(data[i],
				gosmt.CacheBranch(make(map[string][]byte))),
			cache: "B",
		})
		benchCacheSize = append(benchCacheSize, run{
			s: makeCacheSizeBench(data[i],
				gosmt.CacheBranchPlus(make(map[string][]byte))),
			cache: "B+",
		})
	}

	do(fmt.Sprintf("update time (ms) for 2^i keys in 2^%d SMT", keyUpdateDSsize),
		benchUpdateKey, file)
	do(fmt.Sprintf("update time (ms) for %d keys", updateSize), benchUpdate, file)
	do("cache size (MiB)", benchCacheSize, file)
	do("audit path generation time (ms)", benchAP, file)

}

func do(exp string, bench []run, file *os.File) {
	flog(fmt.Sprintf("####### experiment: %s #######", exp), file)
	expCount := len(bench) / maxSMT
	header := "SMT size 2^x"
	for i := 0; i < expCount; i++ {
		header += fmt.Sprintf(", %s", bench[i].cache)
	}
	flog(header, file)

	for i := 0; i < maxSMT; i++ {
		r := fmt.Sprintf("%d", i+1)
		for j := 0; j < expCount; j++ {
			if bench[i].s != nil {
				r += fmt.Sprintf(", %s", bench[i*expCount+j].s())
			} else {
				results := make([]float64, repeat)
				for round := 0; round < repeat; round++ {
					results[round] = float64(testing.Benchmark(bench[i*expCount+j].f).NsPerOp()) / float64(1000*1000) // ns to ms
				}
				avg, err := stats.LoadRawData(results).Mean()
				if err != nil {
					panic(err)
				}
				r += fmt.Sprintf(", %.4f", avg)
			}
		}
		flog(r, file)
	}
}

func flog(str string, f *os.File) {
	_, err := f.WriteString(str + "\n")
	if err != nil {
		panic(err)
	}
	log.Println(str)
}

func makeAuditPathBench(data gosmt.D,
	cache gosmt.Cache) func(b *testing.B) {
	return func(b *testing.B) {
		s := gosmt.NewSMT([]byte{0x42}, cache, hash)
		s.Update(data, gosmt.Key(data), s.N, s.Base, gosmt.Set)

		// create N keys
		keys := make([][]byte, b.N)
		for i := 0; i < b.N; i++ {
			keys[i] = randKey(make([]byte, s.N/8))
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			s.AuditPath(data, s.N, s.Base, keys[i])
		}
	}
}

func makeUpdateBench(data gosmt.D,
	cache gosmt.Cache) func(b *testing.B) {
	return func(b *testing.B) {
		s := gosmt.NewSMT([]byte{0x42}, cache, hash)
		s.Update(data, gosmt.Key(data), s.N, s.Base, gosmt.Set)

		// create updateSize keys
		keys := make([][]byte, updateSize)
		for i := 0; i < updateSize; i++ {
			keys[i] = randKey(make([]byte, s.N/8))
		}
		newdata := make([][]byte, len(data), len(data)+len(keys))
		copy(newdata, data)
		newdata = append(newdata, keys...)
		sort.Sort(gosmt.D(newdata))

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			s.Update(newdata, keys, s.N, s.Base, gosmt.Set)
			b.StopTimer()
			s.Update(data, keys, s.N, s.Base, gosmt.Empty)
			b.StartTimer()
		}
	}
}

func makeUpdateKeyBench(size int, data gosmt.D,
	cache gosmt.Cache) func(b *testing.B) {
	return func(b *testing.B) {
		s := gosmt.NewSMT([]byte{0x42}, cache, hash)
		s.Update(data, gosmt.Key(data), s.N, s.Base, gosmt.Set)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			b.StopTimer()

			keys := make([][]byte, size)
			for i := 0; i < size; i++ {
				keys[i] = randKey(make([]byte, s.N/8))
			}
			newdata := make([][]byte, len(data), len(data)+len(keys))
			copy(newdata, data)
			newdata = append(newdata, keys...)
			sort.Sort(gosmt.D(newdata))

			b.StartTimer()
			s.Update(newdata, keys, s.N, s.Base, gosmt.Set)

			b.StopTimer()
			// cleanup, remove the keys we just inserted
			s.Update(data, keys, s.N, s.Base, gosmt.Empty)
			b.StartTimer()
		}
	}
}

func makeCacheSizeBench(data gosmt.D,
	cache gosmt.Cache) func() string {
	return func() string {
		s := gosmt.NewSMT([]byte{0x42}, cache, hash)
		s.Update(data, gosmt.Key(data), s.N, s.Base, gosmt.Set)

		return fmt.Sprintf("%.4f",
			float64(s.CacheEntries()*int(s.N))/float64(1024*1024))
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
