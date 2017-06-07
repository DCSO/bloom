// DCSO Threat Intelligence Engine
// Copyright (c) 2017, DCSO GmbH

package main

import (
	"bufio"
	"fmt"
	"github.com/dcso/bloom"
	"gopkg.in/urfave/cli.v1"
	"os"
	"path/filepath"
)

func exitWithError(message string) {
	fmt.Fprintf(os.Stderr, "Error: %s \n", message)
	os.Exit(-1)
}

func readValuesIntoFilter(filter *bloom.BloomFilter, interactive bool) {
	//we determine if the program is run interactively or within a pipe
	stat, _ := os.Stdin.Stat()
	var isTerminal = (stat.Mode() & os.ModeCharDevice) != 0
	//if we are not in an interactive session and this is a terminal, we quit
	if !interactive && isTerminal {
		return
	}
	if interactive {
		fmt.Println("Interactive mode: Enter a blank line [by pressing ENTER] to exit (values will not be stored otherwise).")
	}
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" && interactive {
			break
		}
		filter.Add([]byte(scanner.Text()))
	}

}

func insertIntoFilter(path string, gzip bool, interactive bool) {
	filter, err := bloom.LoadFilter(path, gzip)
	if err != nil {
		exitWithError(err.Error())
	}
	readValuesIntoFilter(filter, interactive)
	err = bloom.WriteFilter(filter, path, gzip)
	if err != nil {
		exitWithError(err.Error())
	}
}

func checkAgainstFilter(path string, gzip bool, interactive bool) {
	filter, err := bloom.LoadFilter(path, gzip)
	if err != nil {
		exitWithError(err.Error())
	}
	scanner := bufio.NewScanner(os.Stdin)
	if interactive {
		fmt.Println("Interactive mode: Enter a blank line [by pressing ENTER] to exit.")
	}
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" && interactive {
			break
		}
		if filter.Check([]byte(scanner.Text())) {
			if interactive {
				//in interactive mode we add a > to distinguish the output from the input
				fmt.Printf(">%s\n", scanner.Text())
			} else {
				fmt.Println(scanner.Text())
			}
		}
	}
}

func createFilter(path string, n uint32, p float64, gzip bool, interactive bool) {
	filter := bloom.Initialize(n, p)
	readValuesIntoFilter(&filter, interactive)
	err := bloom.WriteFilter(&filter, path, gzip)
	if err != nil {
		exitWithError(err.Error())
	}
}

func main() {

	app := cli.NewApp()
	app.Name = "Bloom Filter"
	app.Usage = "Utility to work with bloom filters"
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "gzip",
			Usage: "compress bloom file with gzip",
		},
		cli.BoolFlag{
			Name:  "interactive",
			Usage: "interactively add values to the filter",
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
				gzip := c.GlobalBool("gzip")
				interactive := c.GlobalBool("interactive")
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
				createFilter(path, uint32(n), p, gzip, interactive)
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
				gzip := c.GlobalBool("gzip")
				interactive := c.GlobalBool("interactive")
				if path == "" {
					exitWithError("No filename given.")
				}
				path, err := filepath.Abs(path)
				if err != nil {
					return err
				}
				insertIntoFilter(path, gzip, interactive)
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
				gzip := c.GlobalBool("gzip")
				interactive := c.GlobalBool("interactive")
				if path == "" {
					exitWithError("No filename given.")
				}
				path, err := filepath.Abs(path)
				if err != nil {
					return err
				}
				checkAgainstFilter(path, gzip, interactive)
				return nil
			},
		},
	}

	app.Run(os.Args)

}
