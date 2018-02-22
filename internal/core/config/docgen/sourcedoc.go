package docgen

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/doc"
	"go/parser"
	"go/printer"
	"go/token"
	"path/filepath"
	"reflect"
	"strings"
)

var astCache = make(map[string]struct {
	fset *token.FileSet
	pkgs map[string]*ast.Package
})

func structNodes(packageDir, structName string) (*ast.TypeSpec, *ast.CommentGroup) {
	var fset *token.FileSet
	var pkgs map[string]*ast.Package

	cached, ok := astCache[packageDir]
	if ok {
		fset = cached.fset
		pkgs = cached.pkgs
	} else {
		fset = token.NewFileSet()
		var err error
		pkgs, err = parser.ParseDir(fset, packageDir, nil, parser.ParseComments)
		if err != nil {
			panic(err)
		}
	}

	for _, p := range pkgs {
		for _, f := range p.Files {
			// Find the struct specified by structName by looking at all nodes
			// with comments.  This means that the config struct has to have a
			// comment on it or else it won't be found.
			cmap := ast.NewCommentMap(fset, f, f.Comments)
			for node := range cmap {
				switch t := node.(type) {
				case *ast.GenDecl:
					if t.Tok != token.TYPE {
						continue
					}

					if t.Specs[0].(*ast.TypeSpec).Name.Name == structName {
						return t.Specs[0].(*ast.TypeSpec), t.Doc
					}
				}
			}
		}
	}
	panic(fmt.Sprintf("Could not find %s in %s", structName, packageDir))
}

func structDoc(packageDir, structName string) string {
	_, commentGroup := structNodes(packageDir, structName)
	return commentTextToParagraphs(commentGroup.Text())
}

func packageDoc(packageDir string) *doc.Package {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, packageDir, nil, parser.ParseComments)
	if err != nil {
		panic(err)
	}
	if len(pkgs) > 1 {
		panic("Can't handle multiple packages")
	}
	return doc.New(pkgs[filepath.Base(packageDir)], packageDir, doc.AllDecls|doc.AllMethods)
}

func structFieldDocs(packageDir, structName string) map[string]string {
	configStruct, _ := structNodes(packageDir, structName)
	fieldDocs := make(map[string]string)
	for _, field := range configStruct.Type.(*ast.StructType).Fields.List {
		if field.Names != nil {
			fieldDocs[field.Names[0].Name] = commentTextToParagraphs(field.Doc.Text())
		}
	}

	return fieldDocs
}

func monitorDocFromPackageDoc(monitorType string, pkgDoc *doc.Package) string {
	for _, note := range pkgDoc.Notes["MONITOR"] {
		if note.UID == monitorType {
			return note.Body
		}
	}
	return ""
}

func commentTextToParagraphs(t string) string {
	return strings.TrimSpace(strings.Replace(
		strings.Replace(
			strings.Replace(t, "\n\n", "TWOLINES", -1),
			"\n", " ", -1),
		"TWOLINES", "\n", -1))
}

func nodeToString(fset *token.FileSet, n interface{}) string {
	b := bytes.NewBuffer(nil)
	printer.Fprint(b, fset, n)
	return b.String()
}

func getStructTagValue(f *ast.Field, tagName string) string {
	if f.Tag == nil {
		return ""
	}

	tag := reflect.StructTag(strings.Trim(f.Tag.Value, "`"))
	return tag.Get(tagName)
}
