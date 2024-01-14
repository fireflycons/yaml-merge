# yaml-merge

![build](https://img.shields.io/github/actions/workflow/status/fireflycons/yaml-merge/test.yml)

Tool for merging several YAML or JSON documents into a single document. Can be useful in some CI/CD situations where you might have a layered configuration structure.

Merged document can be output as either YAML or JSON.

The command is a standalone binary with no dependencies.

## Usage

```text
Usage: yaml-merge [OPTIONS] file1 file2 [...filen]

Merge two or more YAML or JSON documents together.
Documents are merged in the order they appear on the command line
and the result output defaults to YAML.

  -j    Output JSON instead of YAML. Auto-enabled if output file has .json extension
  -o string
        Output file (stdout if not present)
  -s    Set strict mode (value types for any given key must be the same)
  -v    Verbose (messages written to stderr)
```

## Merge strategy

Maps are deep-merged. For example,

```
	{"one": 1, "two": 2} + {"one": 42, "three": 3}
	== {"one": 42, "two": 2, "three": 3}
```

Sequences are replaced. For example,

```
	{"foo": [1, 2, 3]} + {"foo": [4, 5, 6]}
	== {"foo": [4, 5, 6]}
```

In non-strict mode, attempting to merge
mismatched types (e.g., merging a sequence into a map) replaces the old
value with the new. In strict mode, an error is reported.
