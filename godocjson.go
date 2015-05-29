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

type Func struct {
	Doc  string `json:"doc"`
	Name string `json:"name"`
	Decl *ast.FuncDecl

	// methods
	// (for functions, these fields have the respective zero value)
	Recv  string `json:"recv"` // actual   receiver "T" or "*T"
	Orig  string `json:"orig"` // original receiver "T" or "*T"
	Level int    // embedding level; 0 means not embedded
}

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

type Note struct {
	Pos  token.Pos `json:"pos"`
	End  token.Pos `json:"end"`  // position range of the comment containing the marker
	UID  string    `json:"uid"`  // uid found with the marker
	Body string    `json:"body"` // note body text
}

type Type struct {
	Doc  string `json:"doc"`
	Name string `json:"name"`
	Decl *ast.GenDecl

	// associated declarations
	Consts  []*Value `json:"consts"`  // sorted list of constants of (mostly) this type
	Vars    []*Value `json:"vars"`    // sorted list of variables of (mostly) this type
	Funcs   []*Func  `json:"funcs"`   // sorted list of functions returning this type
	Methods []*Func  `json:"methods"` // sorted list of methods (including embedded ones) of this type
}

type Value struct {
	Doc   string   `json:"doc"`
	Names []string `json:"names"` // var or const names in declaration order
	Decl  *ast.GenDecl
}

func CopyFuncs(f []*doc.Func) []*Func {
	newFuncs := make([]*Func, len(f))
	for i, n := range f {
		newFuncs[i] = &Func{
			Doc:  n.Doc,
			Name: n.Name,
			Orig: n.Orig,
			Recv: n.Recv,
		}
	}
	return newFuncs
}

func CopyValues(c []*doc.Value) []*Value {
	newConsts := make([]*Value, len(c))
	for i, c := range c {
		newConsts[i] = &Value{
			Doc:   c.Doc,
			Names: c.Names,
		}
	}
	return newConsts
}

func CopyPackage(pkg *doc.Package) Package {
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

	newPkg.Consts = CopyValues(pkg.Consts)
	newPkg.Funcs = CopyFuncs(pkg.Funcs)

	newPkg.Types = make([]*Type, len(pkg.Types))
	for i, t := range pkg.Types {
		newPkg.Types[i] = &Type{
			Consts:  CopyValues(t.Consts),
			Doc:     t.Doc,
			Funcs:   CopyFuncs(t.Funcs),
			Methods: CopyFuncs(t.Methods),
			Vars:    CopyValues(t.Vars),
		}
	}

	newPkg.Vars = CopyValues(pkg.Vars)
	return newPkg
}

func main() {
	directories := os.Args[1:]
	for _, dir := range directories {
		fileSet := token.NewFileSet()
		pkgs, firstError := parser.ParseDir(fileSet, dir, nil, parser.ParseComments|parser.AllErrors)
		if firstError != nil {
			panic(firstError)
		}
		for _, pkg := range pkgs {
			docPkg := doc.New(pkg, dir, 0)
			cleanedPkg := CopyPackage(docPkg)
			pkgJson, err := json.MarshalIndent(cleanedPkg, "", "  ")
			if err != nil {
				panic(err)
			}
			fmt.Printf("%s\n", pkgJson)
		}
	}
}
