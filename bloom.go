// DCSO go bloom filter
// Copyright (c) 2017, DCSO GmbH

//Implements a simple and highly efficient variant of the Bloom filter that uses only two hash functions.

package bloom

import (
	"encoding/binary"
	"errors"
	"fmt"
	"hash/fnv"
	"io"
	"io/ioutil"
	"math"
)

// SetError represents an error with a given message related to set operations.
type SetError struct {
	msg string
}

// BloomFilter represents a Bloom filter, a data structure for quickly checking
// for set membership, with a specific desired capacity and false positive
// probability.
type BloomFilter struct {
	//bit array
	v []uint64
	//desired maximum number of elements
	n uint64
	//desired false positive probability
	p float64
	//number of hash functions
	k uint64
	//number of bits
	m uint64
	//number of elements in the filter
	N uint64
	//number of 64-bit integers (generated automatically)
	M uint64
	//arbitrary data that we can attach to the filter
	Data []byte
}

// Read loads a filter from a reader object.
func (s *BloomFilter) Read(input io.Reader) error {
	bs8 := make([]byte, 8)

	if _, err := io.ReadFull(input, bs8); err != nil {
		return err
	}

	flags := binary.LittleEndian.Uint64(bs8)

	if flags & 0xFF != 1 {
		return fmt.Errorf("Invalid version bit (should be 1)")
	}

	if _, err := io.ReadFull(input, bs8); err != nil {
		return err
	}

	s.n = binary.LittleEndian.Uint64(bs8)

	if _, err := io.ReadFull(input, bs8); err != nil {
		return err
	}

	s.p = math.Float64frombits(binary.LittleEndian.Uint64(bs8))

	if _, err := io.ReadFull(input, bs8); err != nil {
		return err
	}

	s.k = binary.LittleEndian.Uint64(bs8)

	if _, err := io.ReadFull(input, bs8); err != nil {
		return err
	}

	s.m = binary.LittleEndian.Uint64(bs8)

	if _, err := io.ReadFull(input, bs8); err != nil {
		return err
	}

	s.N = binary.LittleEndian.Uint64(bs8)

	s.M = uint64(math.Ceil(float64(s.m) / 64.0))

	s.v = make([]uint64, s.M)

	for i := uint64(0); i < s.M; i++ {
		n, err := io.ReadFull(input, bs8)
		if err != nil {
			return err
		}
		if n != 8 {
			return fmt.Errorf("Cannot read from file: %d, position: %d, %d", n, i*8, len(bs8))
		}
		s.v[i] = binary.LittleEndian.Uint64(bs8)
	}

	b, err := ioutil.ReadAll(input)

	if err != nil {
		return err
	}

	s.Data = b

	return nil

}

// NumHashFuncs returns the number of hash functions used in the Bloom filter.
func (s *BloomFilter) NumHashFuncs() uint64 {
	return s.k
}

// MaxNumElements returns the maximal supported number of elements in the Bloom
// filter (capacity).
func (s *BloomFilter) MaxNumElements() uint64 {
	return s.n
}

// NumBits returns the number of bits used in the Bloom filter.
func (s *BloomFilter) NumBits() uint64 {
	return s.m
}

// FalsePositiveProb returns the chosen false positive probability for the
// Bloom filter.
func (s *BloomFilter) FalsePositiveProb() float64 {
	return s.p
}

// Write writes the binary representation of a Bloom filter to an io.Writer.
func (s *BloomFilter) Write(output io.Writer) error {
	bs8 := make([]byte, 8)

	// we write the version bit
	binary.LittleEndian.PutUint64(bs8, 1)
	output.Write(bs8)

	binary.LittleEndian.PutUint64(bs8, s.n)
	output.Write(bs8)
	binary.LittleEndian.PutUint64(bs8, math.Float64bits(s.p))
	output.Write(bs8)
	binary.LittleEndian.PutUint64(bs8, s.k)
	output.Write(bs8)
	binary.LittleEndian.PutUint64(bs8, s.m)
	output.Write(bs8)
	binary.LittleEndian.PutUint64(bs8, s.N)
	output.Write(bs8)

	for i := uint64(0); i < s.M; i++ {
		binary.LittleEndian.PutUint64(bs8, s.v[i])
		n, err := output.Write(bs8)
		if n != 8 {
			return errors.New("Cannot write to file!")
		}
		if err != nil {
			return err
		}
	}
	if s.Data != nil {
		output.Write(s.Data)
	}
	return nil
}

// Reset clears the Bloom filter of all elements.
func (s *BloomFilter) Reset() {
	for i := uint64(0); i < s.M; i++ {
		s.v[i] = 0
	}
	s.N = 0
}

// Fingerprint returns the fingerprint of a given value, as an array of index
// values.
func (s *BloomFilter) Fingerprint(value []byte, fingerprint []uint64) {

	hv := fnv.New64()
	hv.Write(value)
	hn := hv.Sum64()
	hm := hn

	for i := uint64(0); i < s.k; i++ {
		hn = hn+hm*i
		hm = hn
		fingerprint[i] = uint64(hn % s.m)
	}
}

// Add adds a byte array element to the Bloom filter.
func (s *BloomFilter) Add(value []byte) {
	var k, l uint64
	newValue := false
	fingerprint := make([]uint64, s.k)
	s.Fingerprint(value, fingerprint)
	for i := uint64(0); i < s.k; i++ {
		k = uint64(fingerprint[i] / 64)
		l = uint64(fingerprint[i] % 64)
		v := uint64(1 << l)
		if (s.v[k] & v) == 0 {
			newValue = true
		}
		s.v[k] |= v
	}
	if newValue {
		s.N++
	}
}

// Join adds the items of another Bloom filter with identical dimensions to
// the receiver. That is, all elements that are described in the
// second filter will also described by the receiver, and the number of elements
// of the receiver will grow by the number of elements in the added filter.
// Note that it is implicitly assumed that both filters are disjoint! Otherwise
// the number of elements in the joined filter must _only_ be considered an
// upper bound and not an exact value!
// Joining two differently dimensioned filters may yield unexpected results and
// hence is not allowed. An error will be returned in this case, and the
// receiver will be left unaltered.
func (s *BloomFilter) Join(s2 *BloomFilter) error {
	var i uint64
	if s.n != s2.n {
		return fmt.Errorf("filters have different dimensions (n = %d vs. %d))",
			s.n, s2.n)
	}
	if s.p != s2.p {
		return fmt.Errorf("filters have different dimensions (p = %f vs. %f))",
			s.p, s2.p)
	}
	if s.k != s2.k {
		return fmt.Errorf("filters have different dimensions (k = %d vs. %d))",
			s.k, s2.k)
	}
	if s.m != s2.m {
		return fmt.Errorf("filters have different dimensions (m = %d vs. %d))",
			s.m, s2.m)
	}
	if s.M != s2.M {
		return fmt.Errorf("filters have different dimensions (M = %d vs. %d))",
			s.M, s2.M)
	}
	for i = 0; i < s.M; i++ {
		s.v[i] |= s2.v[i]
	}
	if s.N+s2.N < s.N {
		return fmt.Errorf("addition of member counts would overflow")
	}
	s.N += s2.N

	return nil
}

// Check returns true if the given value may be in the Bloom filter, false if it
// is definitely not in it.
func (s *BloomFilter) Check(value []byte) bool {
	fingerprint := make([]uint64, s.k)
	s.Fingerprint(value, fingerprint)
	return s.CheckFingerprint(fingerprint)
}

// CheckFingerprint returns true if the given fingerprint occurs in the Bloom
// filter, false if it does not.
func (s *BloomFilter) CheckFingerprint(fingerprint []uint64) bool {
	var k, l uint64
	for i := uint64(0); i < s.k; i++ {
		k = uint64(fingerprint[i] / 64)
		l = uint64(fingerprint[i] % 64)
		if (s.v[k] & (1 << l)) == 0 {
			return false
		}
	}
	return true
}

// Initialize returns a new, empty Bloom filter with the given capacity (n)
// and FP probability (p).
func Initialize(n uint64, p float64) BloomFilter {
	var bf BloomFilter
	bf.n = n
	bf.p = p
	bf.m = uint64(math.Abs(math.Ceil(float64(n) * math.Log(float64(p)) / (math.Pow(math.Log(2.0), 2.0)))))
	bf.M = uint64(math.Ceil(float64(bf.m) / 64.0))
	bf.k = uint64(math.Ceil(math.Log(2) * float64(bf.m) / float64(n)))
	bf.v = make([]uint64, bf.M)
	return bf
}
