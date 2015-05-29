package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/doc"
	"go/parser"
	"go/token"
	"os"
)

// Func represents a function declaration.
type Func struct {
	Doc               string    `json:"doc"`
	Name              string    `json:"name"`
	PackageName       string    `json:"packageName"`
	PackageImportPath string    `json:"packageImportPath"`
	Filename          string    `json:"filename"`
	Line              int       `json:"line"`
	Decl              *FuncDecl `json:"declaration"`

	// methods
	// (for functions, these fields have the respective zero value)
	Recv string `json:"recv"` // actual   receiver "T" or "*T"
	Orig string `json:"orig"` // original receiver "T" or "*T"
	// Level int    // embedding level; 0 means not embedded
}

// Package represents a package declaration.
type Package struct {
	Type       string             `json:"type"`
	Doc        string             `json:"doc"`
	Name       string             `json:"name"`
	ImportPath string             `json:"importPath"`
	Imports    []string           `json:"imports"`
	Filenames  []string           `json:"filenames"`
	Notes      map[string][]*Note `json:"notes"`
	// DEPRECATED. For backward compatibility Bugs is still populated,
	// but all new code should use Notes instead.
	Bugs []string `json:"bugs"`

	// declarations
	Consts []*Value `json:"consts"`
	Types  []*Type  `json:"types"`
	Vars   []*Value `json:"vars"`
	Funcs  []*Func  `json:"funcs"`
}

// Note represents a note comment.
type Note struct {
	Pos  token.Pos `json:"pos"`
	End  token.Pos `json:"end"`  // position range of the comment containing the marker
	UID  string    `json:"uid"`  // uid found with the marker
	Body string    `json:"body"` // note body text
}

// Type represents a type declaration.
type Type struct {
	PackageName       string `json:"packageName"`
	PackageImportPath string `json:"packageImportPath"`
	Doc               string `json:"doc"`
	Name              string `json:"name"`
	Type              string `json:"type"`
	Filename          string `json:"filename"`
	Line              int    `json:"line"`
	// Decl              *ast.GenDecl

	// associated declarations
	Consts  []*Value `json:"consts"`  // sorted list of constants of (mostly) this type
	Vars    []*Value `json:"vars"`    // sorted list of variables of (mostly) this type
	Funcs   []*Func  `json:"funcs"`   // sorted list of functions returning this type
	Methods []*Func  `json:"methods"` // sorted list of methods (including embedded ones) of this type
}

// Value represents a value declaration.
type Value struct {
	PackageName       string   `json:"packageName"`
	PackageImportPath string   `json:"packageImportPath"`
	Doc               string   `json:"doc"`
	Names             []string `json:"names"` // var or const names in declaration order
	Type              string   `json:"type"`
	Filename          string   `json:"filename"`
	Line              int      `json:"line"`
	// Decl              *ast.GenDecl
}

// FuncDecl represents interesting information from an ast.FuncDecl, attached to a function.
type FuncDecl struct {
	Parameters []FuncParam `json:"parameters"`
}

// FuncParam represents a parameter to a function.
type FuncParam struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

func processFuncDecl(d *ast.FuncDecl) *FuncDecl {

	params := make([]FuncParam, d.Type.Params.NumFields())
	for i, f := range d.Type.Params.List {
		params[i] = FuncParam{
			Type: "", // TODO: do a type switch on reflect.TypeOf(f.Type)
			Name: f.Names[0].String(),
		}
	}

	return &FuncDecl{
		Parameters: params,
	}
}

// CopyFuncs produces a json-annotated array of Func objects from an array of GoDoc Func objects.
func CopyFuncs(f []*doc.Func, packageName string, packageImportPath string, fileSet *token.FileSet) []*Func {
	newFuncs := make([]*Func, len(f))
	for i, n := range f {
		position := fileSet.Position(n.Decl.Pos())
		newFuncs[i] = &Func{
			Doc:               n.Doc,
			Name:              n.Name,
			PackageName:       packageName,
			PackageImportPath: packageImportPath,
			Orig:              n.Orig,
			Recv:              n.Recv,
			Filename:          position.Filename,
			Line:              position.Line,
			Decl:              processFuncDecl(n.Decl),
		}
	}
	return newFuncs
}

// CopyValues produces a json-annotated array of Value objects from an array of GoDoc Value objects.
func CopyValues(c []*doc.Value, packageName string, packageImportPath string, fileSet *token.FileSet) []*Value {
	newConsts := make([]*Value, len(c))
	for i, c := range c {
		position := fileSet.Position(c.Decl.TokPos)
		newConsts[i] = &Value{
			Doc:               c.Doc,
			Names:             c.Names,
			PackageName:       packageName,
			PackageImportPath: packageImportPath,
			Type:              c.Decl.Tok.String(),
			Filename:          position.Filename,
			Line:              position.Line,
		}
	}
	return newConsts
}

// CopyPackage produces a json-annotated Package object from a GoDoc Package object.
func CopyPackage(pkg *doc.Package, fileSet *token.FileSet) Package {
	newPkg := Package{
		Type:       "package",
		Doc:        pkg.Doc,
		Name:       pkg.Name,
		ImportPath: pkg.ImportPath,
		Imports:    pkg.Imports,
		Filenames:  pkg.Filenames,
		Bugs:       pkg.Bugs,
	}

	newPkg.Notes = map[string][]*Note{}
	for key, value := range pkg.Notes {
		notes := make([]*Note, len(value))
		for i, note := range value {
			notes[i] = &Note{
				Pos:  note.Pos,
				End:  note.End,
				UID:  note.UID,
				Body: note.Body,
			}
		}
		newPkg.Notes[key] = notes
	}

	newPkg.Consts = CopyValues(pkg.Consts, pkg.Name, pkg.ImportPath, fileSet)
	newPkg.Funcs = CopyFuncs(pkg.Funcs, pkg.Name, pkg.ImportPath, fileSet)

	newPkg.Types = make([]*Type, len(pkg.Types))
	for i, t := range pkg.Types {
		newPkg.Types[i] = &Type{
			Name:              t.Name,
			PackageName:       pkg.Name,
			PackageImportPath: pkg.ImportPath,
			Consts:            CopyValues(t.Consts, pkg.Name, pkg.ImportPath, fileSet),
			Doc:               t.Doc,
			Funcs:             CopyFuncs(t.Funcs, pkg.Name, pkg.ImportPath, fileSet),
			Methods:           CopyFuncs(t.Methods, pkg.Name, pkg.ImportPath, fileSet),
			Vars:              CopyValues(t.Vars, pkg.Name, pkg.ImportPath, fileSet),
		}
	}

	newPkg.Vars = CopyValues(pkg.Vars, pkg.Name, pkg.ImportPath, fileSet)
	return newPkg
}

func main() {
	if len(os.Args) > 2 {
		panic("Please specify a single directory as the argument.\n")
	}
	directory := os.Args[1]
	fileSet := token.NewFileSet()
	pkgs, firstError := parser.ParseDir(fileSet, directory, nil, parser.ParseComments|parser.AllErrors)
	if firstError != nil {
		panic(firstError)
	}
	if len(pkgs) > 1 {
		panic("Multiple packages found in directory!\n")
	}
	for _, pkg := range pkgs {
		docPkg := doc.New(pkg, directory, 0)
		cleanedPkg := CopyPackage(docPkg, fileSet)
		pkgJSON, err := json.MarshalIndent(cleanedPkg, "", "  ")
		if err != nil {
			panic(err)
		}
		fmt.Printf("%s\n", pkgJSON)
	}
}
