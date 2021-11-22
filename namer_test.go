package parser

import (
	"go/ast"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsExported(t *testing.T) {
	tests := []struct {
		name           string
		expectedResult bool
	}{
		{"internal", false},
		{"External", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := IsExported(tc.name)

			assert.Equal(t, tc.expectedResult, result)
		})
	}
}

func TestInferName(t *testing.T) {
	tests := []struct {
		name        string
		expr        ast.Expr
		expecteName string
	}{
		{
			name:        "Int",
			expr:        &ast.Ident{Name: "int"},
			expecteName: "int",
		},
		{
			name:        "Error",
			expr:        &ast.Ident{Name: "error"},
			expecteName: "error",
		},
		{
			name: "Array",
			expr: &ast.ArrayType{
				Elt: &ast.Ident{Name: "string"},
			},
			expecteName: "stringVals",
		},
		{
			name: "Map",
			expr: &ast.MapType{
				Key:   &ast.Ident{Name: "string"},
				Value: &ast.Ident{Name: "int"},
			},
			expecteName: "stringIntMap",
		},
		{
			name: "Channel",
			expr: &ast.ChanType{
				Value: &ast.Ident{Name: "error"},
			},
			expecteName: "errorCh",
		},
		{
			name: "Struct",
			expr: &ast.StructType{
				Fields: &ast.FieldList{},
			},
			expecteName: "structV",
		},
		{
			name: "StructArray",
			expr: &ast.ArrayType{
				Elt: &ast.StructType{
					Fields: &ast.FieldList{},
				},
			},
			expecteName: "structVVals",
		},
		{
			name: "Interface",
			expr: &ast.InterfaceType{
				Methods: &ast.FieldList{},
			},
			expecteName: "interfaceV",
		},
		{
			name: "InterfaceArray",
			expr: &ast.ArrayType{
				Elt: &ast.InterfaceType{
					Methods: &ast.FieldList{},
				},
			},
			expecteName: "interfaceVVals",
		},
		{
			name:        "ExportedStruct",
			expr:        &ast.Ident{Name: "Embedded"},
			expecteName: "Embedded",
		},
		{
			name:        "UnexportedStruct",
			expr:        &ast.Ident{Name: "embedded"},
			expecteName: "embedded",
		},
		{
			name: "PackageStruct",
			expr: &ast.SelectorExpr{
				X:   &ast.Ident{Name: "entity"},
				Sel: &ast.Ident{Name: "Embedded"},
			},
			expecteName: "Embedded",
		},
		{
			name: "PointerPackageStruct",
			expr: &ast.StarExpr{
				X: &ast.SelectorExpr{
					X:   &ast.Ident{Name: "entity"},
					Sel: &ast.Ident{Name: "Embedded"},
				},
			},
			expecteName: "Embedded",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			name := InferName(tc.expr)

			assert.Equal(t, tc.expecteName, name)
		})
	}
}

func TestConvertToUnexported(t *testing.T) {
	tests := []struct {
		name         string
		expectedName string
	}{
		{
			name:         "err",
			expectedName: "err",
		},
		{
			name:         "ID",
			expectedName: "id",
		},
		{
			name:         "URL",
			expectedName: "url",
		},
		{
			name:         "User",
			expectedName: "user",
		},
		{
			name:         "UserID",
			expectedName: "userID",
		},
		{
			name:         "HTTPRequest",
			expectedName: "httpRequest",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			name := ConvertToUnexported(tc.name)

			assert.Equal(t, tc.expectedName, name)
		})
	}
}
