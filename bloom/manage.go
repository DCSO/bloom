// DCSO Threat Intelligence Engine
// Copyright (c) 2017, DCSO GmbH

package main

import (
	"bufio"
	"strings"
	"fmt"
	"strconv"
	"github.com/dcso/bloom"
	"gopkg.in/urfave/cli.v1"
	"os"
	"path/filepath"
)

type BloomParams struct {
	gzip bool
	interactive bool
	split bool
	printEachMatch bool
	delimiter string
	fields []int
}

func exitWithError(message string) {
	fmt.Fprintf(os.Stderr, "Error: %s \n", message)
	os.Exit(-1)
}

func splitLine(line string, delimiter string, fields []int) []string {
	result := strings.Split(line, delimiter)
	if len(fields) > 0 {
		relevantResult := make([]string, len(fields))
		for _, i := range fields {
			if i >= len(result) {
				continue
			}
			relevantResult = append(relevantResult, result[i])
		}
		return relevantResult
	}
	return result
}

func readValuesIntoFilter(filter *bloom.BloomFilter, bloomParams BloomParams) {
	//we determine if the program is run interactively or within a pipe
	stat, _ := os.Stdin.Stat()
	var isTerminal = (stat.Mode() & os.ModeCharDevice) != 0
	//if we are not in an interactive session and this is a terminal, we quit
	if !bloomParams.interactive && isTerminal {
		return
	}
	if bloomParams.interactive {
		fmt.Println("Interactive mode: Enter a blank line [by pressing ENTER] to exit (values will not be stored otherwise).")
	}
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" && bloomParams.interactive {
			break
		}
		if bloomParams.split {
			values := splitLine(line, bloomParams.delimiter, bloomParams.fields)
			for _, value := range values {
				filter.Add([]byte(value))
			}
		} else {
			filter.Add([]byte(line))
		}
	}

}

func insertIntoFilter(path string, bloomParams BloomParams) {
	filter, err := bloom.LoadFilter(path, bloomParams.gzip)
	if err != nil {
		exitWithError(err.Error())
	}
	readValuesIntoFilter(filter, bloomParams)
	err = bloom.WriteFilter(filter, path, bloomParams.gzip)
	if err != nil {
		exitWithError(err.Error())
	}
}

func checkAgainstFilter(path string, bloomParams BloomParams) {
	filter, err := bloom.LoadFilter(path, bloomParams.gzip)
	if err != nil {
		exitWithError(err.Error())
	}
	scanner := bufio.NewScanner(os.Stdin)
	if bloomParams.interactive {
		fmt.Println("Interactive mode: Enter a blank line [by pressing ENTER] to exit.")
	}
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" && bloomParams.interactive {
			break
		}
		var valuesToCheck []string
		if bloomParams.split {
			valuesToCheck = splitLine(line, bloomParams.delimiter, bloomParams.fields)
		} else {
			valuesToCheck = make([]string, 1)
			valuesToCheck[0] = line
		}
		printed := false
		prefix := ""
		if bloomParams.interactive {
			prefix = ">"
		}
		for _, value := range(valuesToCheck) {
			if filter.Check([]byte(value)) {
				if bloomParams.printEachMatch {
					fmt.Printf("%s%s\n", prefix, value)
				} else {
					if !printed {
						fmt.Printf("%s%s\n", prefix, line)
					}
					printed = true
				}
			}
		}
		}
}

func createFilter(path string, n uint32, p float64, bloomParams BloomParams) {
	filter := bloom.Initialize(n, p)
	readValuesIntoFilter(&filter, bloomParams)
	err := bloom.WriteFilter(&filter, path, bloomParams.gzip)
	if err != nil {
		exitWithError(err.Error())
	}
}

func parseBloomParams(c *cli.Context) BloomParams {
	var bloomParams BloomParams
	bloomParams.gzip = c.GlobalBool("gzip")
	bloomParams.interactive = c.GlobalBool("interactive")
	bloomParams.split = c.GlobalBool("split")
	bloomParams.delimiter = c.GlobalString("delimiter")
	bloomParams.printEachMatch = c.GlobalBool("each")
	if c.GlobalString("fields") != "" {
		fields := strings.Split(c.GlobalString("fields"), bloomParams.delimiter)
		fieldNumbers := make([]int, len(fields))
		for i, field := range fields {
			num, err := strconv.Atoi(field)
			if err != nil {
				exitWithError(fmt.Sprintf("Invalid field value (should be a number): %s", field))
			}
			fieldNumbers[i] = num
		}
		bloomParams.fields = fieldNumbers
	}
	return bloomParams
}

func main() {

	app := cli.NewApp()
	app.Name = "Bloom Filter"
	app.Usage = "Utility to work with bloom filters"
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "gzip, gz",
			Usage: "compress bloom file with gzip",
		},
		cli.BoolFlag{
			Name:  "interactive, i",
			Usage: "interactively add values to the filter",
		},
		cli.BoolFlag{
			Name:  "split, s",
			Usage: "split the input string",
		},
		cli.BoolFlag{
			Name:  "each, e",
			Usage: "print each match of a splitted string individually",
		},
		cli.StringFlag{
			Name:  "delimiter, d",
			Value: ",",
			Usage: "delimiter to use for splitting",
		},
		cli.StringFlag{
			Name:  "fields, f",
			Value: "",
			Usage: "fields of splitted output to use in filter (a single number or a comma-separated list of numbers, zero-indexed)",
		},
	}
	app.Commands = []cli.Command{
		{
			Name:    "create",
			Aliases: []string{"c"},
			Flags: []cli.Flag{
				cli.Float64Flag{Name: "p", Value: 0.01, Usage: "The desired false positive probability."},
				cli.IntFlag{Name: "n", Value: 10000, Usage: "The desired capacity."},
			},
			Usage: "Create a new Bloom filter and store it in the given filename.",
			Action: func(c *cli.Context) error {
				path := c.Args().First()
				bloomParams := parseBloomParams(c)
				if path == "" {
					exitWithError("No filename given.")
				}
				path, err := filepath.Abs(path)
				if err != nil {
					return err
				}
				n := c.Int("n")
				p := c.Float64("p")
				if n < 0 {
					exitWithError("n cannot be negative.")
				}
				if p < 0 || p > 1 {
					exitWithError("p must be between 0 and 1.")
				}
				createFilter(path, uint32(n), p, bloomParams)
				return nil
			},
		},
		{
			Name:    "insert",
			Aliases: []string{"i"},
			Flags:   []cli.Flag{},
			Usage:   "Inserts new values into an existing Bloom filter.",
			Action: func(c *cli.Context) error {
				path := c.Args().First()
				bloomParams := parseBloomParams(c)
				if path == "" {
					exitWithError("No filename given.")
				}
				path, err := filepath.Abs(path)
				if err != nil {
					return err
				}
				insertIntoFilter(path, bloomParams)
				return nil
			},
		},
		{
			Name:    "check",
			Aliases: []string{"c"},
			Flags:   []cli.Flag{},
			Usage:   "Checks values against an existing Bloom filter.",
			Action: func(c *cli.Context) error {
				path := c.Args().First()
				bloomParams := parseBloomParams(c)
				if path == "" {
					exitWithError("No filename given.")
				}
				path, err := filepath.Abs(path)
				if err != nil {
					return err
				}
				checkAgainstFilter(path, bloomParams)
				return nil
			},
		},
	}

	app.Run(os.Args)

}
