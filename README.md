# Bloom

### A highly efficient bloom filter implementation for Go

[![GoDoc](https://godoc.org/github.com/DCSO/bloom?status.svg)](http://godoc.org/github.com/DCSO/bloom)
![Build Status](https://github.com/DCSO/bloom/actions/workflows/go.yml/badge.svg)

Bloom is a simple tool that provides a very efficient implementation of Bloom filters for the go language.
It provides a command line tool that can be used to easily create Bloom filters with desired capacity
and false positive probability. Values can be added to filters through standard input, which makes it
easy to use the tool in a pipeline workflow.

# Usage

    NAME:
       Bloom Filter - Utility to work with bloom filters

    USAGE:
       bloom [global options] command [command options] [arguments...]

    VERSION:
       0.2.2

    COMMANDS:
         create, cr         Create a new Bloom filter and store it in the given filename.
         insert, i          Inserts new values into an existing Bloom filter.
         join, j, merge, m  Joins two Bloom filters into one.
         check, c           Checks values against an existing Bloom filter.
         set-data, sd       Sets the data associated with the Bloom filter.
         get-data, gd       Prints the data associated with the Bloom filter.
         show, s            Shows various details about a given Bloom filter.
         help, h            Shows a list of commands or help for one command

    GLOBAL OPTIONS:
       --gzip, --gz                      compress bloom file with gzip
       --interactive, -i                 interactively add values to the filter
       --split, -s                       split the input string
       --each, -e                        print each match of a split string individually
       --delimiter value, -d value       delimiter to use for splitting (default: ",")
       --fields value, -f value          fields of split output to use in filter (a single number or a comma-separated list of numbers, zero-indexed)
       --print-fields value, --pf value  fields of split output to print for a successful match (a single number or a comma-separated list of numbers, zero-indexed).
       --help, -h                        show help
       --version, -v                     print the version


# Examples

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

# Advanced Usage

Sometimes it is useful to attach additional information to a string that we want to check against the Bloom filter,
such as a timestamp or the original line content. To make passing along this additional information easier within
a shell context, the Bloom tool provides an option for splitting the input string by a given delimiter and checking
the filter against the resulting field values. Example:

    # will check the Bloom filter for the values foo, bar and baz
    cat "foo,bar,baz" | bloom -s filter.bloom

    # uses a different delimiter (--magic-delimiter--)
    cat "foo--ac5ba--bar--ac5ba--baz" | bloom  -d "--ac5ba--" -s filter.bloom

    # will check the Bloom filter against the second field value only
    cat "foo,bar,baz" | bloom -f 1 -s filter.bloom

    # will check the Bloom filter against the second and third field values only
    cat "foo,bar,baz" | bloom -f 1,2 -s filter.bloom

    # will print one line for each field value that matched against the filter
    cat "foo,bar,baz" | bloom -e -s filter.bloom

    # will print the last field value for each line whose fields matched against the filter
    cat "foo,bar,baz" | bloom -e -s --pf -1 filter.bloom

This functionality is especially handy when using CSV data, as it allows you to filter CSV rows by checking individual
columns against the filter without having to use external tools to split and reassemble the lines.

# Installation

## Installation on Debian-based systems

Debian [command line tool](https://tracker.debian.org/pkg/golang-github-dcso-bloom) (only available in stretch-backports, buster and sid):

    sudo apt install golang-github-dcso-bloom-cli

## Installation from source

These need to be run from within the GOPATH source directory for this project. To install the command line tool:

    make install

To run the tests:

    make test

To run the benchmarks:

    make bench

# Cross-Compiling

To compile a binary, simply specify the target architecture and go:

    #Windows, 64 bit
    env GOOS=windows GOARCH=amd64 go build -v -o bloom.exe github.com/DCSO/bloom
    #Windows, 32 bit
    env GOOS=windows GOARCH=i386 go build -v -o /tmp/bloom github.com/DCSO/bloom
