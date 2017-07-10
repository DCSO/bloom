// DCSO go bloom filter
// Copyright (c) 2017, DCSO GmbH

package bloom

import (
  "os"
  "io/ioutil"
  "net/http"
  "testing"

  httpmock "gopkg.in/jarcoal/httpmock.v1"
)

func checkResults(t *testing.T, bf *BloomFilter) {
  for _, v := range []string{"foo", "bar", "baz"} {
    if !bf.Check([]byte(v)) {
      t.Fatal("value %s expected in filter but wasn't found", v)
    }
  }
  if bf.Check([]byte("")) {
    t.Fatal("empty value not expected in filter but was found")
  }
  if bf.Check([]byte("12345")) {
    t.Fatal("missing value not expected in filter but was found")
  }
}

func TestFromReaderFile(t *testing.T) {
  f, err := os.Open("testdata/test.bloom")
  if err != nil {
		t.Fatal(err)
	}
  defer f.Close()
  bf, err := LoadFromReader(f, false)
  checkResults(t, bf)
}

func TestFromReaderHttp(t *testing.T) {
  httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	testBloomFile, err := ioutil.ReadFile("testdata/test.bloom")
	if err != nil {
		t.Fatal(err)
	}
	httpmock.RegisterResponder("GET", "https://localhost:9998/test.bloom",
	httpmock.NewBytesResponder(200, testBloomFile))
  response, err := http.Get("https://localhost:9998/test.bloom");
  if err != nil {
		t.Fatal(err)
	}
  defer response.Body.Close()
  bf, err := LoadFromReader(response.Body, false)
  checkResults(t, bf)
}

func TestFromBytes(t *testing.T) {
	testBytes, err := ioutil.ReadFile("testdata/test.bloom")
	if err != nil {
		t.Fatal(err)
	}
  bf, err := LoadFromBytes(testBytes, false)
  checkResults(t, bf)
}

func TestFromFile(t *testing.T) {
  bf, err := LoadFilter("testdata/test.bloom", false)
  if err != nil {
		t.Fatal(err)
	}
  checkResults(t, bf)
}
