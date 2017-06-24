package main

import (
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"

	"group_imports"
	"strconv"
)

type grouper struct {
	// The group numbers of prefixed packages.
	prefixes map[int]string

	// The group numbers of standard packages and unidentified packages.
	std, other int

	// The next integer to assign
	next int
}

func newGrouper() *grouper {
	return &grouper{
		prefixes: make(map[int]string),
		std:      0,
		other:    1,
		next:     2,
	}
}

func (g *grouper) Group(pkg string) int {
	for n, prefix := range g.prefixes {
		if strings.HasPrefix(pkg, prefix) {
			return n
		}
	}

	// A dot distinguishes non-standard packages.
	if strings.Contains(pkg, ".") {
		return g.other
	} else {
		return g.std
	}
}

func (g *grouper) wasSet() bool {
	return g.next > 2
}

func (g *grouper) String() string {
	parts := []string{}
	remain := len(g.prefixes)
	for i := 0; i <= g.std || i <= g.other || remain > 0; i++ {
		if g.std == i {
			parts = append(parts, "std")
		} else if g.other == i {
			parts = append(parts, "other")
		} else if p, ok := g.prefixes[i]; ok {
			parts = append(parts, fmt.Sprintf("prefix=%s", p))
			remain--
		}
	}
	return strings.Join(parts, ",")
}

var rePrefix = regexp.MustCompile(`^prefix=(.*)$`)

func (g *grouper) Set(s string) error {
	parts := strings.Split(s, ",")
	for _, p := range parts {
		if p == "std" {
			g.std = g.next
		} else if p == "other" {
			g.other = g.next
		} else if match := rePrefix.FindStringSubmatch(p); match != nil {
			g.prefixes[g.next] = match[1]
		} else {
			return fmt.Errorf("Unknown order specification '%s'", p)
		}
		g.next++
	}
	return nil
}

const (
	statusError       = 1
	statusHelp        = 2
	statusInvalidFile = 3
)

func validateOne(proc *group_imports.Processor, file string) (validErr *group_imports.ValidationError, err error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return proc.Validate(file, f)
}

func validate(gr *grouper, files []string) {
	proc := group_imports.NewProcessor(gr)
	invalid := false

	for _, file := range files {
		validErr, err := validateOne(proc, file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err.Error())
			os.Exit(statusError)
		}
		if validErr != nil {
			invalid = true
			fmt.Fprintf(os.Stdout, "%s:%d: %s at %s\n", file, validErr.Line,
				validErr.Message, strconv.Quote(validErr.ImportPath))
		}
	}

	if invalid {
		os.Exit(statusInvalidFile)
	}
}

func main() {
	rewrite := false
	gr := newGrouper()

	flag.Usage = func() {
		// Hard to get flag to format long usage well, so just put everything here.
		fmt.Fprintln(os.Stderr,
			`group-imports: Enforce import grouping in Go source files.

Exits with status 3 if import grouping is violated.

Usage: group-imports [OPTIONS] FILE...

  -rewrite
      Instead of checking import grouping, rewrite the source files with
      the correct grouping. Default: false.

  -order SPEC[,SPEC...]
      Modify the import grouping strategy by listing the desired groups in
      order. Group specifications include:

      - std: Standard library imports
      - prefix=PREFIX: Imports whose path starts with PREFIX
      - other: Imports that match no other specification

      These groups can be specified in one comma-separated argument, or
      multiple arguments. Default: std,other
`,
		)
	}

	flag.BoolVar(&rewrite, "rewrite", false, "")
	flag.Var(gr, "order", "")

	flag.Parse()
	if flag.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "No file provided.")
		flag.Usage()
		os.Exit(statusHelp)
	}

	if rewrite {
		// TODO
	} else {
		validate(gr, flag.Args())
	}
}
