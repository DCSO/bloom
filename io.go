// DCSO go bloom filter
// Copyright (c) 2017, DCSO GmbH

package bloom

import (
	"bufio"
	"bytes"
	gz "compress/gzip"
	"io"
	"os"
)

// LoadFromBytes reads a binary Bloom filter representation from a byte array
// and returns a BloomFilter struct pointer based on it.
// If 'gzip' is true, then compressed input will be expected.
func LoadFromBytes(input []byte, gzip bool) (*BloomFilter, error) {
	return LoadFromReader(bytes.NewReader(input), gzip)
}

// LoadFilter reads a binary Bloom filter representation from a file
// and returns a BloomFilter struct pointer based on it.
// If 'gzip' is true, then compressed input will be expected.
func LoadFilter(path string, gzip bool) (*BloomFilter, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return LoadFromReader(file, gzip)
}

// LoadFromReader reads a binary Bloom filter representation from an io.Reader
// and returns a BloomFilter struct pointer based on it.
// If 'gzip' is true, then compressed input will be expected.
func LoadFromReader(inReader io.Reader, gzip bool) (*BloomFilter, error) {
	var err error
	var reader io.Reader
	var gzipReader *gz.Reader
	var ioReader *bufio.Reader

	if gzip {
		gzipReader, err = gz.NewReader(inReader)
		if err != nil {
			return nil, err
		}
		defer gzipReader.Close()
		reader = gzipReader
	} else {
		ioReader = bufio.NewReader(inReader)
		reader = ioReader
	}

	var filter BloomFilter
	if err = filter.Read(reader); err != nil {
		return nil, err
	}

	return &filter, nil
}

// WriteFilter writes a binary Bloom filter representation for a given struct
// to a file. If 'gzip' is true, then a compressed file will be written.
func WriteFilter(filter *BloomFilter, path string, gzip bool) error {

	file, err := os.Create(path)

	if err != nil {
		return err
	}

	defer file.Close()

	file.Seek(0, 0)

	var writer io.Writer
	var gzipWriter *gz.Writer
	var ioWriter *bufio.Writer

	if gzip {
		gzipWriter = gz.NewWriter(file)
		defer gzipWriter.Close()
		writer = gzipWriter
	} else {
		ioWriter = bufio.NewWriter(file)
		writer = ioWriter
	}

	err = filter.Write(writer)

	if err != nil {
		return err
	}

	if gzip {
		gzipWriter.Flush()
	} else {
		ioWriter.Flush()
	}

	file.Sync()

	return nil
}
