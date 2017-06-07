// DCSO Threat Intelligence Engine
// Copyright (c) 2017, DCSO GmbH

package bloom

import "os"
import "math"
import "bytes"
import "io/ioutil"
import "testing"
import "math/rand"

func TestInitialization(t *testing.T) {
	filter := Initialize(10000, 0.001)
	if filter.k != 10 {
		t.Error("k does not match expectation!")
	}
	if filter.m != 143775 {
		t.Error("m does not match expectation: ", filter.m)
	}
	if filter.M != uint32(math.Ceil(float64(filter.m)/64)) {
		t.Error("M does not match expectation: ", filter.M)
	}
	for i := uint32(0); i < filter.M; i++ {
		if filter.v[i] != 0 {
			t.Error("Filter value is not initialized to zero!")
		}
	}
}

func checkFilters(a BloomFilter, b BloomFilter, t *testing.T) bool {
	if b.n != a.n ||
		b.p != a.p ||
		b.k != a.k ||
		b.m != a.m ||
		b.M != a.M {
		return false
	}
	for i := uint32(0); i < a.M; i++ {
		if a.v[i] != b.v[i] {
			return false
		}
	}
	return true
}

func serializeToBuffer(filter BloomFilter) (*BloomFilter, error) {
	var buf bytes.Buffer
	filter.Write(&buf)
	var newFilter BloomFilter
	newFilter.Read(&buf)
	return &newFilter, nil
}

func serializeToDisk(filter BloomFilter) (*BloomFilter, error) {
	tempFile, err := ioutil.TempFile("", "filter")
	if err != nil {
		return nil, err
	}
	defer os.Remove(tempFile.Name())
	filter.Write(tempFile)
	tempFile.Sync()
	tempFile.Seek(0, 0)
	var newFilter BloomFilter
	err = newFilter.Read(tempFile)
	if err != nil {
		return nil, err
	}
	return &newFilter, nil
}

func TestSerialization(t *testing.T) {
	capacity := uint32(100000)
	p := float64(0.01)
	samples := uint32(1000)
	filter, _ := GenerateExampleFilter(capacity, p, samples)

	newFilter, err := serializeToBuffer(filter)
	if err != nil {
		t.Error("Cannot serialize filter to buffer!")
		return
	}

	if !checkFilters(filter, *newFilter, t) {
		t.Error("Filters do not match!")
	}

	newFilter, err = serializeToDisk(filter)

	if err != nil {
		t.Error("Cannot serialize filter to file!")
		return
	}

	if !checkFilters(filter, *newFilter, t) {
		t.Error("Filters do not match!")
	}

	filter.Add(GenerateTestValue(100))
	newFilter.Add(GenerateTestValue(100))
	newFilter, err = serializeToDisk(filter)
	if err != nil {
		t.Error("Cannot serialize filter to disk!")
		return
	}

	if !checkFilters(filter, *newFilter, t) {
		t.Error("Filters do not match!")
	}

	filter.Add(GenerateTestValue(100))
	newFilter.Add(GenerateTestValue(100))
	newFilter, err = serializeToDisk(filter)
	if err != nil {
		t.Error("Cannot serialize filter to disk!")
		return
	}

	if !checkFilters(filter, *newFilter, t) {
		t.Error("Filters do not match!")
	}

	checkFilters(filter, *newFilter, t)
}

func TestSerializationToDisk(t *testing.T) {
	capacity := uint32(100000)
	p := float64(0.001)
	samples := uint32(1000)
	filter, _ := GenerateExampleFilter(capacity, p, samples)

	var buf bytes.Buffer

	filter.Write(&buf)

	var newFilter BloomFilter

	newFilter.Read(&buf)

	checkFilters(filter, newFilter, t)
}

func GenerateTestValue(length uint32) []byte {
	value := make([]byte, length)
	for i := uint32(0); i < length; i++ {
		value[i] = byte(rand.Int() % 256)
	}
	return value
}

func GenerateExampleFilter(capacity uint32, p float64, samples uint32) (BloomFilter, [][]byte) {
	filter := Initialize(capacity, p)
	testValues := make([][]byte, 0, samples)
	for i := uint32(0); i < samples; i++ {
		testValue := GenerateTestValue(100)
		testValues = append(testValues, testValue)
		filter.Add(testValue)
	}
	return filter, testValues
}

//This tests the checking of values against a given filter
func TestChecking(t *testing.T) {
	capacity := uint32(100000)
	p := float64(0.001)
	samples := uint32(100000)
	filter, testValues := GenerateExampleFilter(capacity, p, samples)
	fingerprint := make([]uint32, filter.k)
	for _, value := range testValues {
		filter.Fingerprint(value, fingerprint)
		if !filter.CheckFingerprint(fingerprint) {
			t.Error("Did not find test value in filter!")
		}
	}
}

//This tests the checking of values against a given filter after resetting it
func TestReset(t *testing.T) {
	capacity := uint32(100000)
	p := float64(0.001)
	samples := uint32(100000)
	filter, testValues := GenerateExampleFilter(capacity, p, samples)
	filter.Reset()
	fingerprint := make([]uint32, filter.k)
	for _, value := range testValues {
		filter.Fingerprint(value, fingerprint)
		if filter.CheckFingerprint(fingerprint) {
			t.Error("Did not find test value in filter!")
		}
	}
}

//This tests the checking of values against a given filter
//see https://en.wikipedia.org/wiki/Bloom_filter#Probability_of_false_positives
func TestFalsePositives(t *testing.T) {
	capacity := uint32(10000)
	p := float64(0.001)
	fillingFactor := 0.9
	N := uint32(float64(capacity) * fillingFactor)
	filter, _ := GenerateExampleFilter(capacity, p, N)
	pAcceptable := math.Pow(1-math.Exp(-float64(filter.k)*float64(N)/float64(filter.m)), float64(filter.k))
	fingerprint := make([]uint32, filter.k)
	cnt := 0.0
	matches := 0.0
	for {
		cnt++
		value := GenerateTestValue(100)
		filter.Fingerprint(value, fingerprint)
		if filter.CheckFingerprint(fingerprint) {
			matches++
		}
		if cnt > float64(capacity)*10 {
			break
		}
	}
	//this might still fail sometimes...
	//we allow for a probability that is two times higher than the normally acceptable probability
	if matches/cnt > pAcceptable*2 {
		t.Error("False positive probability is too high at ", matches/cnt*100, "% vs ", pAcceptable*100, "%")
	}
}

//This benchmarks the checking of values against a given filter
func BenchmarkChecking(b *testing.B) {
	capacity := uint32(1e9)
	p := float64(0.001)
	samples := uint32(100000)
	filter, testValues := GenerateExampleFilter(capacity, p, samples)
	fingerprint := make([]uint32, filter.k)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		value := testValues[rand.Int()%len(testValues)]
		filter.Fingerprint(value, fingerprint)
		if !filter.CheckFingerprint(fingerprint) {
			b.Error("Did not find test value in filter!")
		}
	}
}

//This benchmarks the checking without using a fixed fingerprint variable (instead a temporary variable is created each time)
func BenchmarkSimpleChecking(b *testing.B) {
	capacity := uint32(1e9)
	p := float64(0.001)
	samples := uint32(100000)
	filter, testValues := GenerateExampleFilter(capacity, p, samples)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		value := testValues[rand.Int()%len(testValues)]
		if !filter.Check(value) {
			b.Error("Did not find test value in filter!")
		}
	}
}
