# Bloom - A highly efficient bloom filter implementation for Go

Bloom is a simple tool that provides a very efficient implementation of Bloom filters for the go language.
It provides a command line tool that can be used to easily create Bloom filters with desired capacity
and false positive probability. Values can be added to filters through standard input, which makes it
easy to use the tool in a pipeline workflow.

# Usage

To create a new bloom filter with a desired capacity and false positive probability, you can use the `create` command:

    #will create a gzipped Bloom filter with 100.000 capacity and a 0.1 % false positive probability
    bloom --gzip create -p 0.001 -n 100000 test.bloom.gz

To insert values, you can use the `insert` command and pipe some input to it (each line will be treated as one value):

    cat values | bloom --gzip insert test.bloom.gz

You can also interactively add values to the filter by specifying the `--interactive` command line option:

    bloom --gzip --interactive insert test.bloom.gz

To check if a given value or a list of values is in the filter, you can use the `check` command:

    cat values | bloom --gzip check test.bloom.gz

This will return a list of all values in the filter.

# Installation & Usage

To install the command line tool:

    make install

To run the tests:

    make test

To run the benchmarks:

    make bench
