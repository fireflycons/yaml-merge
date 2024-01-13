package main

import (
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	yaml "gopkg.in/yaml.v3"
)

func unmarshal(yml string) interface{} {
	var contents interface{}
	var nilResult map[string]interface{}
	d := yaml.NewDecoder(strings.NewReader(yml))
	if err := d.Decode(&contents); err == io.EOF {
		return nilResult
	} else if err != nil {
		panic(err)
	}

	return contents
}

func TestMerge(t *testing.T) {

	testcases := []struct {
		name          string
		input1        string
		input2        string
		expectError   bool
		errorContains string
		toJson        bool
		strict        bool
		output        map[string]interface{}
	}{
		{
			name:        "merge simple maps",
			input1:      `{"one": 1, "two": 2}`,
			input2:      `{"one": 42, "three": 3}`,
			expectError: false,
			output: map[string]interface{}{
				"one":   42,
				"two":   2,
				"three": 3,
			},
		},
		{
			name:        "merge simple maps output JSON",
			input1:      `{"one": 1, "two": 2}`,
			input2:      `{"one": 42, "three": 3}`,
			expectError: false,
			toJson:      true,
			output: map[string]interface{}{
				"one":   42,
				"two":   2,
				"three": 3,
			},
		},
		{
			name:        "merge simple sequences",
			input1:      `{"foo": [1, 2, 3]}}`,
			input2:      `{"foo": [4, 5, 6]}`,
			expectError: false,
			output: map[string]interface{}{
				"foo": []interface{}{
					4, 5, 6,
				},
			},
		},
		{
			name: "test n",
			// Assert that this value is treated as string and not boolean false
			input1:      `marker: n`,
			input2:      ``,
			expectError: false,
			output: map[string]interface{}{
				"marker": "n",
			},
		},
		{
			name:          "duplicate key",
			input1:        `{"one": 1, "two": 2, "one": 99}`,
			input2:        `{"one": 42, "three": 3}`,
			expectError:   true,
			errorContains: "already defined",
		},
		{
			name:        "non-strict",
			input1:      `{"one": 1, "two": 2}`,
			input2:      `{"one": [1, 2], "three": 3}`,
			expectError: false,
			output: map[string]interface{}{
				"one": []interface{}{
					1,
					2,
				},
				"two":   2,
				"three": 3,
			},
		},
		{
			name:          "strict",
			input1:        `{"one": 1, "two": 2}`,
			input2:        `{"one": [1, 2], "three": 3}`,
			strict:        true,
			expectError:   true,
			errorContains: "can't merge a sequence into a scalar",
		},
		{
			name:          "input-error",
			input1:        `{"one": 1, "two": 2`,
			input2:        `{"one": [1, 2], "three": 3}`,
			expectError:   true,
			errorContains: "couldn't decode source",
		},
		{
			name:   "empty inputs",
			input1: ``,
			input2: ``,
			output: nil,
		},
		{
			name:   "first input empty",
			input1: ``,
			input2: `{"one": 1, "two": 2}`,
			output: map[string]interface{}{
				"one": 1,
				"two": 2,
			},
		},
		{
			name:   "second input empty",
			input1: `{"one": 1, "two": 2}`,
			input2: ``,
			output: map[string]interface{}{
				"one": 1,
				"two": 2,
			},
		},
		{
			name:        "null value",
			input1:      `{"one": 1, "two": 2}`,
			input2:      `{"one": 42, "two": null}`,
			expectError: false,
			output: map[string]interface{}{
				"one": 42,
				"two": nil,
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			output := &strings.Builder{}
			err := mergeDocuments(tc.strict, tc.toJson, output, strings.NewReader(tc.input1), strings.NewReader(tc.input2))
			if tc.expectError {
				require.ErrorContains(t, err, tc.errorContains)
			} else {
				require.NoError(t, err)
				if tc.toJson {
					require.True(t, output.String()[0] == '{' || output.String()[0] == '[')
				}
				require.Equal(t, tc.output, unmarshal(output.String()))
			}
		})
	}
}
