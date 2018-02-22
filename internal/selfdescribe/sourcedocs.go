package selfdescribe

import (
	"fmt"
	"go/ast"
	"go/doc"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
)

var astCache = make(map[string]struct {
	fset *token.FileSet
	pkgs map[string]*ast.Package
})

// Returns the ast node of the struct itself and the comment group on the
// struct type.
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

func commentTextToParagraphs(t string) string {
	return strings.TrimSpace(strings.Replace(
		strings.Replace(
			strings.Replace(t, "\n\n", "TWOLINES", -1),
			"\n", " ", -1),
		"TWOLINES", "\n", -1))
}
