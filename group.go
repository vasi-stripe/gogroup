package gogroup

import (
	"fmt"
	"go/parser"
	"go/token"
	"io"
	"sort"
	"strconv"

	"bufio"
	"bytes"
	"io/ioutil"

	"golang.org/x/tools/imports"
)

// An import statement with a group.
type groupedImport struct {
	// The zero-based starting and ending lines in the file.
	// The endLine is the last line of this statement, not the line after.
	startLine, endLine int

	// The import package path.
	path string

	// The import group.
	group int
}

// Allow sorting grouped imports.
type groupedImports []*groupedImport

func (gs groupedImports) Len() int {
	return len(gs)
}
func (gs groupedImports) Swap(i, j int) {
	gs[i], gs[j] = gs[j], gs[i]
}
func (gs groupedImports) Less(i, j int) bool {
	if gs[i].group < gs[j].group {
		return true
	}
	if gs[i].group == gs[j].group && gs[i].path < gs[j].path {
		return true
	}
	return false
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s (line %s)", e.Message, e.ImportPath, e.Line)
}

func validationError(g *groupedImport, msg string) *ValidationError {
	return &ValidationError{
		Message:    msg,
		ImportPath: g.path,
		Line:       g.startLine,
	}
}

const (
	errstrStatementOrder     = "Import out of order within import group"
	errstrStatementExtraLine = "Extra empty line inside import group"
	errstrStatementGroup     = "Import in incorrect group"
	errstrGroupOrder         = "Import groups out of order"
	errstrGroupExtraLine     = "Extra empty line between import groups"
)

// Validate an import group.
func (gs groupedImports) validate() *ValidationError {
	if len(gs) < 2 {
		// Always valid!
		return nil
	}

	var prev *groupedImport
	for _, g := range gs {
		if prev != nil {
			emptyLines := g.startLine - prev.endLine - 1

			if g.group == prev.group {
				if emptyLines > 0 {
					return validationError(g, errstrStatementExtraLine)
				} else if g.path < prev.path {
					return validationError(g, errstrStatementOrder)
				}
			} else if emptyLines == 0 {
				// This could also be a missing empty line.
				return validationError(g, errstrStatementGroup)
			} else if g.group < prev.group {
				return validationError(g, errstrGroupOrder)
			} else if emptyLines > 1 {
				return validationError(g, errstrGroupExtraLine)
			}

		}
		prev = g
	}
	return nil

}

// Read import statements from a file, and assign them groups.
func (p *Processor) readImports(fileName string, r io.Reader) (groupedImports, error) {
	fset := token.NewFileSet()
	tree, err := parser.ParseFile(fset, fileName, r, parser.ImportsOnly|parser.ParseComments)
	if err != nil {
		return nil, err
	}

	gs := groupedImports{}
	for _, ispec := range tree.Imports {
		var path string
		path, err = strconv.Unquote(ispec.Path.Value)
		if err != nil {
			return nil, err
		}

		startPos, endPos := ispec.Pos(), ispec.End()
		if ispec.Doc != nil {
			// Comments go with the following import statement.
			startPos = ispec.Doc.Pos()
		}

		file := fset.File(startPos)
		gs = append(gs, &groupedImport{
			path: path,
			// Line numbers are one-based in token.File.
			startLine: file.Line(startPos) - 1,
			endLine:   file.Line(endPos) - 1,
			group:     p.grouper.Group(path),
		})
	}

	return gs, nil
}

func (p *Processor) validate(fileName string, r io.Reader) (validErr *ValidationError, err error) {
	gs, err := p.readImports(fileName, r)
	if err != nil {
		return nil, err
	}
	return gs.validate(), nil
}

func readLines(r io.Reader) ([]string, error) {
	scanner := bufio.NewScanner(r)
	ret := []string{}
	for scanner.Scan() {
		ret = append(ret, scanner.Text())
	}
	if scanner.Err() != nil {
		return nil, scanner.Err()
	}
	return ret, nil
}

func writeLines(w io.Writer, lines []string) error {
	for _, line := range lines {
		_, err := fmt.Fprintln(w, line)
		if err != nil {
			return err
		}
	}
	return nil
}

func sortedImportLines(gs groupedImports, lines []string) []string {
	sort.Sort(gs)

	ret := []string{}
	var prev *groupedImport
	for _, g := range gs {
		if prev != nil && g.group != prev.group {
			// Time for an empty line.
			ret = append(ret, "")
		}
		ret = append(ret, lines[g.startLine:g.endLine+1]...)
		prev = g
	}

	return ret
}

func writeFixed(src []byte, gs groupedImports) (io.Reader, error) {
	lines, err := readLines(bytes.NewReader(src))
	if err != nil {
		return nil, err
	}

	min := gs[0].startLine
	max := gs[len(gs)-1].endLine

	// Need to start a new slice, or we may modify lines as we append.
	out := []string{}
	out = append(out, lines[:min]...)
	out = append(out, sortedImportLines(gs, lines)...)
	out = append(out, lines[max+1:]...)

	var dst bytes.Buffer
	if err = writeLines(&dst, out); err != nil {
		return nil, err
	}

	return &dst, nil
}

func (p *Processor) repair(fileName string, r io.Reader) (io.Reader, error) {
	// Get the full contents.
	src, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	// Check if the file needs any fixing.
	gs, err := p.readImports(fileName, bytes.NewReader(src))
	if err != nil {
		return nil, err
	}
	if gs.validate() == nil {
		return nil, nil
	}

	// Generate the fixed version.
	dst, err := writeFixed(src, gs)
	if err != nil {
		return nil, err
	}

	return dst, nil
}

func (p *Processor) reformat(fileName string, r io.Reader) (io.Reader, error) {
	// Get the full contents.
	src, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	formatted, err := imports.Process(fileName, src, nil)
	if err != nil {
		return nil, err
	}

	ret, err := p.repair(fileName, bytes.NewReader(formatted))
	if err != nil {
		return nil, err
	}
	if ret == nil && bytes.Equal(src, formatted) {
		// No change by either goimports or grouping.
		return nil, nil
	}
	return ret, nil
}
