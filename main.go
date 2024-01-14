// Copyright (c) 2024 Firefly Consulting Ltd.
// Portions Copyright (c) 2019 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	yaml "gopkg.in/yaml.v3"
)

type (
	// YAML/JSON has three fundamental types. When unmarshaled into interface{},
	// they're represented like this.
	mapping  = map[string]interface{}
	sequence = []interface{}
	// The third type is scalar which is simply interface{}
	// however can be detected by not being one of the other two.
)

// Interface that can represent JSON or YAML encoder.
type encoder interface {
	Encode(interface{}) error
}

// mergeDocuments deep-merges any number of YAML/JSON sources, with later sources taking
// priority over earlier ones.
//
// Maps are deep-merged. For example,
//
//	{"one": 1, "two": 2} + {"one": 42, "three": 3}
//	== {"one": 42, "two": 2, "three": 3}
//
// Sequences are replaced. For example,
//
//	{"foo": [1, 2, 3]} + {"foo": [4, 5, 6]}
//	== {"foo": [4, 5, 6]}
//
// In non-strict mode, attempting to merge
// mismatched types (e.g., merging a sequence into a map) replaces the old
// value with the new.
//
// Enabling strict mode returns errors in the above case.
func mergeDocuments(strict, toJson bool, dest io.Writer, sources ...io.Reader) error {
	var merged interface{}
	var hasContent bool

	for i, r := range sources {
		// JSON is YAML so doesn't matter what the input is.
		d := yaml.NewDecoder(r)

		var contents interface{}
		if err := d.Decode(&contents); err == io.EOF {
			// Skip empty and comment-only sources, which we should handle
			// differently from explicit nils.
			continue
		} else if err != nil {
			return fmt.Errorf("couldn't decode source (input file #%d): %v", i, err)
		}

		hasContent = true
		pair, err := merge(merged, contents, strict)
		if err != nil {
			return err // error is already descriptive enough
		}
		merged = pair
	}

	if !hasContent {
		// No sources had any content. To distinguish this from a source with just
		// an explicit top-level null, return an empty buffer.
		return nil
	}

	var enc encoder

	if toJson {
		enc = json.NewEncoder(dest)
		enc.(*json.Encoder).SetIndent("", "    ")
	} else {
		enc = yaml.NewEncoder(dest)
	}

	if err := enc.Encode(merged); err != nil {
		return fmt.Errorf("couldn't re-serialize merged documents: %v", err)
	}
	return nil
}

// merge performs the merge of element 'from' into element 'into'.
func merge(into, from interface{}, strict bool) (interface{}, error) {

	switch {
	case into == nil:
		// No change
		return from, nil
	case from == nil:
		// Allow higher-priority document to explicitly nil out lower-priority entries.
		return nil, nil
	case isScalar(into) && isScalar(from):
		// Both elements are a scalar entry
		return from, nil
	case isSequence(into) && isSequence(from):
		// Both elements are a sequence
		return from, nil
	case isMapping(into) && isMapping(from):
		// Both elements are a map
		return mergeMapping(into.(mapping), from.(mapping), strict)
	case !strict:
		// value types don't match, so no merge is possible. For backward
		// compatibility, ignore mismatches unless we're in strict mode and return
		// the higher-priority value.
		return from, nil
	default:
		return nil, fmt.Errorf("can't merge a %s into a %s", describe(from), describe(into))
	}
}

// mergeMapping recursively merges map `from` into map `into`.
func mergeMapping(into, from mapping, strict bool) (mapping, error) {
	// Output map will be at least the same number of keys as the `into` doc
	merged := make(mapping, len(into))

	// Copy `into` doc to output doc
	for k, v := range into {
		merged[k] = v
	}

	// Enumerate keys of `from` doc, replacing
	// matching keys of `into` with values from `from`
	for k := range from {
		// Recursively merge this value
		m, err := merge(merged[k], from[k], strict)

		if err != nil {
			return nil, err
		}

		merged[k] = m
	}

	return merged, nil
}

// isMapping reports whether a type is a mapping in YAML, represented as a
// map[interface{}]interface{}.
func isMapping(i interface{}) bool {
	_, is := i.(mapping)
	return is
}

// isSequence reports whether a type is a sequence in YAML, represented as an
// []interface{}.
func isSequence(i interface{}) bool {
	_, is := i.(sequence)
	return is
}

// isScalar reports whether a type is a scalar value in YAML.
func isScalar(i interface{}) bool {
	return !isMapping(i) && !isSequence(i)
}

// describe describes the element type of i.
func describe(i interface{}) string {
	if isMapping(i) {
		return "mapping"
	}
	if isSequence(i) {
		return "sequence"
	}
	return "scalar"
}

func usage() {
	fmt.Printf("Usage: %s [OPTIONS] file1 file2 [...filen]\n\n", os.Args[0])
	fmt.Printf("Merge two or more YAML or JSON documents together.\nDocuments are merged in the order they appear on the command line\nand the result output defaults to YAML.\n\n")
	flag.PrintDefaults()
}

func main() {
	var strict, toJson, verbose bool
	var outputFilename string

	output := os.Stdout

	flag.BoolVar(&strict, "s", false, "Set strict mode (value types for any given key must be the same)")
	flag.StringVar(&outputFilename, "o", "", "Output file (stdout if not present)")
	flag.BoolVar(&toJson, "j", false, "Output JSON instead of YAML. Auto-enabled if output file has .json extension")
	flag.BoolVar(&verbose, "v", false, "Verbose (messages written to stderr)")
	flag.Usage = usage
	flag.Parse()

	readers := make([]io.Reader, 0, len(flag.Args()))

	if verbose {
		fmt.Fprintf(os.Stderr, "Files to be merged in this order:\n\n")
	}

	// Remaining non-flag arguments are files to merge.
	count := 0
	for _, fileArg := range flag.Args() {
		// Allow commma separated list of files as single arg
		for _, f := range strings.Split(fileArg, ",") {
			if rd, err := os.Open(f); err == nil {
				readers = append(readers, rd)
				defer func(i int) {
					readers[i].(*os.File).Close()
				}(count)

				count++
				if verbose {
					fmt.Fprintf(os.Stderr, "  %02d. %s\n", count, f)
				}

			} else {
				fmt.Fprintf(os.Stderr, "cannot open %s for reading: %v\n", f, err)
				os.Exit(1)
			}
		}
	}

	if outputFilename != "" {
		wd, err := os.OpenFile(outputFilename, os.O_CREATE, 0644)

		if err != nil {
			fmt.Fprintf(os.Stderr, "cannot open %s for writing: %v\n", outputFilename, err)
			os.Exit(1)
		}

		if strings.ToLower(filepath.Ext(outputFilename)) == ".json" {
			toJson = true
		}

		if verbose {
			fmt.Fprintf(os.Stderr, "\nOutput to %s\n", outputFilename)
		}

		output = wd
		defer output.Close()
	}

	err := mergeDocuments(strict, toJson, output, readers...)

	if err != nil {
		fmt.Fprintf(os.Stderr, "merge error: %v\n", err)
		os.Exit(1)
	}
}
