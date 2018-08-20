package gosmt

import "crypto/sha512"

// thank you https://play.golang.org/p/sycUxCZyxf.

// bitIsSet checks whether bit of a bit string (stored as a byte string)
// at position i (range:[1, N] and big-endian) is set to 1 .
func bitIsSet(bits []byte, i uint64) bool { return bits[i/8]&(1<<uint(7-i%8)) != 0 }

// bitSet sets bit at position i (range:[1, N] and big-endian) to 1.
func bitSet(bits []byte, i uint64) { bits[i/8] |= 1 << uint(7-i%8) }

// bitSplit returns a new bit string (stored as byte string) whose bit
// at position i (range:[1, N] and big-endian) is set to 1.
// This is usually used to get the split parameter of a node in SMT,
// given its base and height.
func bitSplit(bits []byte, i uint64) (split []byte) {
	split = make([]byte, len(bits))
	copy(split, bits)
	bitSet(split, i)
	return
}

// hash returns a hashed digest on a list of concatenated bytes strings.
// SHA512/256 truncated SHA512 to 256 bits, and is as safe as SHA-256,
// but faster on 64-bit architecture.
func hash(data ...[]byte) []byte {
	hasher := sha512.New512_256()
	for i := 0; i < len(data); i++ {
		hasher.Write(data[i])
	}
	return hasher.Sum(nil)
}
