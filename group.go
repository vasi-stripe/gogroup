package group_imports

import (
	"fmt"
	"go/parser"
	"go/token"
	"io"
	"strconv"
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
				if g.path < prev.path {
					return validationError(g, errstrStatementOrder)
				} else if emptyLines > 0 {
					return validationError(g, errstrStatementExtraLine)
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
		path, err := strconv.Unquote(ispec.Path.Value)
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

func (p *Processor) repair(fileName string, r io.Reader) (io.Reader, error) {
	// TODO
	return nil, nil
}

func (p *Processor) reformat(fileName string, r io.Reader) (io.Reader, error) {
	// TODO
	return nil, nil
}
