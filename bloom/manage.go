// DCSO go bloom filter
// Copyright (c) 2017, DCSO GmbH

package main

import (
	"bytes"
	"bufio"
	"fmt"
	"github.com/DCSO/bloom"
	"gopkg.in/urfave/cli.v1"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// BloomParams represents the parameters of the 'bloom' command line tool.
type BloomParams struct {
	gzip           bool
	interactive    bool
	split          bool
	printEachMatch bool
	delimiter      string
	fields         []int
	printFields    []int
}

func exitWithError(message string) {
	fmt.Fprintf(os.Stderr, "Error: %s \n", message)
	os.Exit(-1)
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
			values := strings.Split(line, bloomParams.delimiter)
			for i, value := range values {
				j := i - len(values)

				if len(bloomParams.fields) > 0 {
					if !contains(bloomParams.fields, i) && !contains(bloomParams.fields, j) {
						continue
					}
				}

				filter.Add([]byte(value))
			}
		} else {
			filter.Add([]byte(line))
		}
	}
}

func readInputIntoData(filter *bloom.BloomFilter, bloomParams BloomParams) {
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
	dataBuffer := bytes.NewBuffer([]byte(""))
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 && bloomParams.interactive {
			break
		}
		dataBuffer.Write(line)
		dataBuffer.Write([]byte("\n"))
	}
	filter.Data = dataBuffer.Bytes()
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

func updateFilterData(path string, bloomParams BloomParams) {
	filter, err := bloom.LoadFilter(path, bloomParams.gzip)
	if err != nil {
		exitWithError(err.Error())
	}
	readInputIntoData(filter, bloomParams)
	err = bloom.WriteFilter(filter, path, bloomParams.gzip)
	if err != nil {
		exitWithError(err.Error())
	}
}

func getFilterData(path string, bloomParams BloomParams) {
	filter, err := bloom.LoadFilter(path, bloomParams.gzip)
	if err != nil {
		exitWithError(err.Error())
	}
	fmt.Print(string(filter.Data))
}

func contains(s []int, e int) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
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
			valuesToCheck = strings.Split(line, bloomParams.delimiter)
		} else {
			valuesToCheck = make([]string, 1)
			valuesToCheck[0] = line
		}
		printed := false
		prefix := ""
		if bloomParams.interactive {
			prefix = ">"
		}
		for i, value := range valuesToCheck {
			j := i - len(valuesToCheck)
			//we only check fields that are in the "fields" parameters (if defined)
			if len(bloomParams.fields) > 0 {
				if !contains(bloomParams.fields, i) && !contains(bloomParams.fields, j) {
					continue
				}
			}

			if filter.Check([]byte(value)) {
				if bloomParams.printEachMatch {
					fmt.Printf("%s%s\n", prefix, value)
				} else {
					if !printed {
						if len(bloomParams.printFields) > 0 {
							values := make([]string, 0, len(bloomParams.printFields))
							for _, i := range bloomParams.printFields {
								j := i
								if j < 0 {
									j = j + len(valuesToCheck)
								}
								if j >= len(valuesToCheck) || j < 0 {
									continue
								}
								values = append(values, valuesToCheck[j])
							}
							fmt.Printf("%s%s\n", prefix, strings.Join(values, bloomParams.delimiter))
						} else {
							fmt.Printf("%s%s\n", prefix, line)
						}
					}
					printed = true
				}
			}
		}
	}
}

func printStats(path string, bloomParams BloomParams) {
	filter, err := bloom.LoadFilter(path, bloomParams.gzip)
	if err != nil {
		exitWithError(err.Error())
	}
	fmt.Printf("File:\t\t\t%s\n", path)
	fmt.Printf("Capacity:\t\t%d\n", filter.MaxNumElements())
	fmt.Printf("Elements present:\t%d\n", filter.N)
	fmt.Printf("FP probability:\t\t%f\n", filter.FalsePositiveProb())
	fmt.Printf("Bits:\t\t\t%d\n", filter.NumBits())
	fmt.Printf("Hash functions:\t\t%d\n", filter.NumHashFuncs())
}

func createFilter(path string, n uint32, p float64, bloomParams BloomParams) {
	filter := bloom.Initialize(n, p)
	readValuesIntoFilter(&filter, bloomParams)
	err := bloom.WriteFilter(&filter, path, bloomParams.gzip)
	if err != nil {
		exitWithError(err.Error())
	}
}

func joinFilters(path string, pathToAdd string, bloomParams BloomParams) {
	filter, err := bloom.LoadFilter(path, bloomParams.gzip)
	if err != nil {
		exitWithError(err.Error())
	}
	filter2, err := bloom.LoadFilter(pathToAdd, bloomParams.gzip)
	if err != nil {
		exitWithError(err.Error())
	}
	err = filter.Join(filter2)
	if err != nil {
		exitWithError(err.Error())
	}
	err = bloom.WriteFilter(filter, path, bloomParams.gzip)
	if err != nil {
		exitWithError(err.Error())
	}
}

func parseFieldIndexes(s string) ([]int, error) {
	fields := strings.Split(s, ",")
	fieldNumbers := make([]int, len(fields))
	for i, field := range fields {
		num, err := strconv.Atoi(field)
		if err != nil {
			return nil, err
		}
		fieldNumbers[i] = num
	}
	return fieldNumbers, nil
}

func parseBloomParams(c *cli.Context) BloomParams {
	var bloomParams BloomParams
	var err error
	bloomParams.gzip = c.GlobalBool("gzip")
	bloomParams.interactive = c.GlobalBool("interactive")
	bloomParams.split = c.GlobalBool("split")
	bloomParams.delimiter = c.GlobalString("delimiter")
	bloomParams.printEachMatch = c.GlobalBool("each")
	if c.GlobalString("fields") != "" {
		bloomParams.fields, err = parseFieldIndexes(c.GlobalString("fields"))
		if err != nil {
			exitWithError(err.Error())
		}
	}
	if c.GlobalString("print-fields") != "" {
		bloomParams.printFields, err = parseFieldIndexes(c.GlobalString("print-fields"))
		if err != nil {
			exitWithError(err.Error())
		}
		//if printFields is set we also set printEachMatch
		if len(bloomParams.printFields) > 0 {
			bloomParams.printEachMatch = false
		}
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
		cli.StringFlag{
			Name:  "print-fields, pf",
			Value: "",
			Usage: "fields of splitted output to print for a successful match (a single number or a comma-separated list of numbers, zero-indexed).",
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
			Name:    "join",
			Aliases: []string{"j", "merge", "m"},
			Flags:   []cli.Flag{},
			Usage:   "Joins two Bloom filters into one.",
			Action: func(c *cli.Context) error {
				if len(c.Args()) != 2 {
					exitWithError("Two filenames are required.")
				}
				bloomParams := parseBloomParams(c)
				path := c.Args().First()
				if path == "" {
					exitWithError("No first filename given.")
				}
				path, err := filepath.Abs(path)
				if err != nil {
					return err
				}
				pathToAdd := c.Args().Get(1)
				if pathToAdd == "" {
					exitWithError("No second filename given.")
				}
				pathToAdd, err = filepath.Abs(pathToAdd)
				if err != nil {
					return err
				}
				joinFilters(path, pathToAdd, bloomParams)
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
		{
			Name:    "set-data",
			Aliases: []string{"c"},
			Flags:   []cli.Flag{},
			Usage:   "Sets the data associated with the Bloom filter.",
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
				updateFilterData(path, bloomParams)
				return nil
			},
		},
		{
			Name:    "get-data",
			Aliases: []string{"c"},
			Flags:   []cli.Flag{},
			Usage:   "Prints the data associated with the Bloom filter.",
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
				getFilterData(path, bloomParams)
				return nil
			},
		},
		{
			Name:    "show",
			Aliases: []string{"s"},
			Flags:   []cli.Flag{},
			Usage:   "Shows various details about a given Bloom filter.",
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
				printStats(path, bloomParams)
				return nil
			},
		},
	}
	app.Version = "0.1.1"

	app.Run(os.Args)
}
