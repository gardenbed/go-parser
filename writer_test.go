package parser

import (
	"go/ast"
	"go/token"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWriteFile(t *testing.T) {
	mainFile := &ast.File{
		Name: &ast.Ident{Name: "main"},
		Decls: []ast.Decl{
			&ast.GenDecl{
				Tok: token.IMPORT,
				Specs: []ast.Spec{
					&ast.ImportSpec{
						Path: &ast.BasicLit{
							Value: `"fmt"`,
						},
					},
				},
			},
			&ast.FuncDecl{
				Name: &ast.Ident{Name: "main"},
				Type: &ast.FuncType{
					Params: &ast.FieldList{},
				},
				Body: &ast.BlockStmt{
					List: []ast.Stmt{
						&ast.ExprStmt{
							X: &ast.CallExpr{
								Fun: &ast.SelectorExpr{
									X:   &ast.Ident{Name: "fmt"},
									Sel: &ast.Ident{Name: "Println"},
								},
								Args: []ast.Expr{
									&ast.BasicLit{
										Value: `"Hello, World!"`,
									},
								},
							},
						},
					},
				},
			},
		},
	}

	tests := []struct {
		name          string
		path          string
		fset          *token.FileSet
		file          *ast.File
		expectedError string
	}{
		{
			name:          "InvalidPath",
			path:          ".",
			fset:          token.NewFileSet(),
			file:          mainFile,
			expectedError: "open .: is a directory",
		},
		{
			name: "InvalidFile",
			path: "./main.go",
			fset: token.NewFileSet(),
			file: &ast.File{
				Name: &ast.Ident{},
			},
			expectedError: "goimports error: ./main.go:1:9: expected 'IDENT', found 'EOF'",
		},
		{
			name:          "Success",
			path:          "./main.go",
			fset:          token.NewFileSet(),
			file:          mainFile,
			expectedError: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := WriteFile(tc.path, tc.fset, tc.file)

			// Cleanup
			defer os.Remove(tc.path)
			defer os.Remove(getDebugFilename(tc.path))

			if tc.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tc.expectedError)
			}
		})
	}
}
