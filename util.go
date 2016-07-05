package gosmt

import "crypto/sha512"

// thank you https://play.golang.org/p/sycUxCZyxf.
func bitIsSet(bits []byte, i uint64) bool { return bits[i/8]&(1<<uint(7-i%8)) != 0 }
func bitSet(bits []byte, i uint64)        { bits[i/8] |= 1 << uint(7-i%8) }
func bitSplit(bits []byte, i uint64) (split []byte) {
	split = make([]byte, len(bits))
	copy(split, bits)
	bitSet(split, i)
	return
}

func hash(data ...[]byte) []byte {
	hasher := sha512.New512_256()
	for i := 0; i < len(data); i++ {
		hasher.Write(data[i])
	}
	return hasher.Sum(nil)
}
