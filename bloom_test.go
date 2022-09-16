// DCSO go bloom filter
// Copyright (c) 2017, DCSO GmbH

package bloom

import (
	"bytes"
	"io/ioutil"
	"log"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFingerprinting(t *testing.T) {
	filter := Initialize(100000, 0.01)
	fp := make([]uint64, 7)
	expected := [7]uint64{20311, 36825, 412501, 835777, 658914, 853361, 307361}
	filter.Fingerprint([]byte("bar"), fp)
	for i, v := range fp {
		if v != expected[i] {
			t.Errorf("Wrong fingerprint: %d vs. %d", v, expected[i])
			break
		}
	}
}

func TestInitialization(t *testing.T) {
	filter := Initialize(10000, 0.001)
	if filter.k != 10 {
		t.Error("k does not match expectation!")
	}
	if filter.m != 143775 {
		t.Error("m does not match expectation: ", filter.m)
	}
	if filter.M != uint64(math.Ceil(float64(filter.m)/64)) {
		t.Error("M does not match expectation: ", filter.M)
	}
	for i := uint64(0); i < filter.M; i++ {
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
		b.M != a.M ||
		!bytes.Equal(b.Data, a.Data) {
		return false
	}
	for i := uint64(0); i < a.M; i++ {
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
	capacity := uint64(100000)
	p := float64(0.01)
	samples := uint64(1000)
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
	capacity := uint64(100000)
	p := float64(0.001)
	samples := uint64(1000)
	filter, _ := GenerateExampleFilter(capacity, p, samples)

	var buf bytes.Buffer

	filter.Write(&buf)

	var newFilter BloomFilter

	newFilter.Read(&buf)

	checkFilters(filter, newFilter, t)
}

func TestSerializationWriteFail(t *testing.T) {
	capacity := uint64(100000)
	p := float64(0.001)
	samples := uint64(1000)
	filter, _ := GenerateExampleFilter(capacity, p, samples)

	dir, err := ioutil.TempDir("", "bloomtest")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(dir)

	tmpfn := filepath.Join(dir, "tmpfile")
	tmpfile, err := os.OpenFile(tmpfn, os.O_CREATE|os.O_RDONLY, 0000)
	if err != nil {
		t.Fatal(err)
	}
	defer tmpfile.Close()

	err = filter.Write(tmpfile)
	if err == nil {
		t.Error("writing to read-only file should fail")
	}
}

func TestSerializationReadFail(t *testing.T) {
	var newFilter BloomFilter

	dir, err := ioutil.TempDir("", "bloomtest")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(dir)

	tmpfn := filepath.Join(dir, "tmpfile")
	tmpfile, err := os.OpenFile(tmpfn, os.O_CREATE, 0777)
	if err != nil {
		t.Fatal(err)
	}
	defer tmpfile.Close()

	err = newFilter.Read(tmpfile)
	if err == nil {
		t.Error("reading from empty file should fail")
	}
}

func GenerateTestValue(length uint64) []byte {
	value := make([]byte, length)
	for i := uint64(0); i < length; i++ {
		value[i] = byte(rand.Int() % 256)
	}
	return value
}

func GenerateExampleFilter(capacity uint64, p float64, samples uint64) (BloomFilter, [][]byte) {
	filter := Initialize(capacity, p)
	filter.Data = []byte("foobar")
	testValues := make([][]byte, 0, samples)
	for i := uint64(0); i < samples; i++ {
		testValue := GenerateTestValue(100)
		testValues = append(testValues, testValue)
		filter.Add(testValue)
	}
	return filter, testValues
}

func GenerateDisjointExampleFilter(capacity uint64, p float64, samples uint64, other BloomFilter) (BloomFilter, [][]byte) {
	filter := Initialize(capacity, p)
	testValues := make([][]byte, 0, samples)
	for i := uint64(0); i < samples; {
		testValue := GenerateTestValue(100)
		if !other.Check(testValue) {
			testValues = append(testValues, testValue)
			filter.Add(testValue)
			i++
		}
	}
	return filter, testValues
}

// This tests the checking of values against a given filter
func TestChecking(t *testing.T) {
	capacity := uint64(100000)
	p := float64(0.001)
	samples := uint64(100000)
	filter, testValues := GenerateExampleFilter(capacity, p, samples)
	fingerprint := make([]uint64, filter.k)
	for _, value := range testValues {
		filter.Fingerprint(value, fingerprint)
		if !filter.CheckFingerprint(fingerprint) {
			t.Error("Did not find test value in filter!")
		}
	}
}

// This tests the checking of values against a given filter after resetting it
func TestReset(t *testing.T) {
	capacity := uint64(100000)
	p := float64(0.001)
	samples := uint64(100000)
	filter, testValues := GenerateExampleFilter(capacity, p, samples)
	filter.Reset()
	fingerprint := make([]uint64, filter.k)
	for _, value := range testValues {
		filter.Fingerprint(value, fingerprint)
		if filter.CheckFingerprint(fingerprint) {
			t.Error("Did not find test value in filter!")
		}
	}
}

// This tests the checking of values against a given filter
// see https://en.wikipedia.org/wiki/Bloom_filter#Probability_of_false_positives
func TestFalsePositives(t *testing.T) {
	capacity := uint64(10000)
	p := float64(0.001)
	fillingFactor := 0.9
	N := uint64(float64(capacity) * fillingFactor)
	filter, _ := GenerateExampleFilter(capacity, p, N)
	pAcceptable := math.Pow(1-math.Exp(-float64(filter.k)*float64(N)/float64(filter.m)), float64(filter.k))
	fingerprint := make([]uint64, filter.k)
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

func TestJoiningRegularMisdimensioned(t *testing.T) {
	a := Initialize(100000, 0.0001)
	b := Initialize(10000, 0.0001)
	err := a.Join(&b)
	if err == nil {
		t.Error("joining filters with different capacity should fail")
	}
	if !strings.Contains(err.Error(), "different dimensions") {
		t.Error("wrong error message returned")
	}
	a = Initialize(100000, 0.0001)
	b = Initialize(100000, 0.001)
	err = a.Join(&b)
	if err == nil {
		t.Error("joining filters with different FP prob should fail")
	}
	if !strings.Contains(err.Error(), "different dimensions") {
		t.Error("wrong error message returned")
	}
	a = Initialize(100000, 0.0001)
	b = Initialize(100000, 0.0001)
	b.k = 1
	err = a.Join(&b)
	if err == nil {
		t.Error("joining filters with different number of hash funcs should fail")
	}
	if !strings.Contains(err.Error(), "different dimensions") {
		t.Error("wrong error message returned")
	}
	a = Initialize(100000, 0.0001)
	b = Initialize(100000, 0.0001)
	b.m = 1
	err = a.Join(&b)
	if err == nil {
		t.Error("joining filters with different number of bits should fail")
	}
	if !strings.Contains(err.Error(), "different dimensions") {
		t.Error("wrong error message returned")
	}
	a = Initialize(100000, 0.0001)
	b = Initialize(100000, 0.0001)
	b.M = 1
	err = a.Join(&b)
	if err == nil {
		t.Error("joining filters with different int array size should fail")
	}
	if !strings.Contains(err.Error(), "different dimensions") {
		t.Error("wrong error message returned")
	}
}

func TestAccessors(t *testing.T) {
	a, _ := GenerateExampleFilter(100000, 0.0001, 10000)
	if a.MaxNumElements() != 100000 {
		t.Error("unexpected capacity in filter")
	}
	if a.NumBits() != 1917011 {
		t.Error("unexpected number of bits in filter")
	}
	if a.NumHashFuncs() != 14 {
		t.Error("unexpected number of hash funcs in filter")
	}
	if a.FalsePositiveProb() != 0.0001 {
		t.Error("unexpected FP prob in filter")
	}
}

func TestJoiningRegular(t *testing.T) {
	a, aval := GenerateExampleFilter(100000, 0.0001, 10000)
	b, bval := GenerateDisjointExampleFilter(100000, 0.0001, 20000, a)
	c, _ := GenerateDisjointExampleFilter(100000, 0.0001, 85000, b)
	for _, v := range bval {
		if a.Check(v) {
			t.Errorf("value not missing in joined filter: %s", string(v))
		}
	}
	if a.N != 10000 {
		t.Error("unexpected number of elements in filter")
	}
	if b.N != 20000 {
		t.Error("unexpected number of elements in filter")
	}
	err := a.Join(&b)
	if a.N != 30000 {
		t.Errorf("unexpected number of elements in filter")
	}
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range aval {
		if !a.Check(v) {
			t.Errorf("value not found in joined filter: %s", string(v))
		}
	}
	for _, v := range bval {
		if !a.Check(v) {
			t.Errorf("value not found in joined filter: %s", string(v))
		}
	}
	expected := "addition of member counts would overflow"
	actual := b.Join(&c)
	if actual == nil {
		t.Errorf("Expected error %v not triggered", expected)
	} else {
		if actual.Error() != expected {
			t.Errorf("Error actual = %v, and Expected = %v.", actual, expected)
		}
	}
}

// This benchmarks the checking of values against a given filter
func BenchmarkChecking(b *testing.B) {
	capacity := uint64(1e9)
	p := float64(0.001)
	samples := uint64(100000)
	filter, testValues := GenerateExampleFilter(capacity, p, samples)
	fingerprint := make([]uint64, filter.k)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		value := testValues[rand.Int()%len(testValues)]
		filter.Fingerprint(value, fingerprint)
		if !filter.CheckFingerprint(fingerprint) {
			b.Error("Did not find test value in filter!")
		}
	}
}

// This benchmarks the checking without using a fixed fingerprint variable (instead a temporary variable is created each time)
func BenchmarkSimpleChecking(b *testing.B) {
	capacity := uint64(1e9)
	p := float64(0.001)
	samples := uint64(100000)
	filter, testValues := GenerateExampleFilter(capacity, p, samples)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		value := testValues[rand.Int()%len(testValues)]
		if !filter.Check(value) {
			b.Error("Did not find test value in filter!")
		}
	}
}
