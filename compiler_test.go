package parser

import (
	"errors"
	"testing"

	goast "go/ast"

	"github.com/gardenbed/charm/ui"
	"github.com/stretchr/testify/assert"
)

func TestNewCompiler(t *testing.T) {
	tests := []struct {
		name      string
		ui        ui.UI
		consumers []*Consumer
	}{
		{
			name:      "OK",
			ui:        ui.New(ui.Info),
			consumers: []*Consumer{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := NewCompiler(tc.ui, tc.consumers...)

			assert.NotNil(t, c)
			assert.NotNil(t, c.parser)
			assert.Equal(t, tc.ui, c.parser.ui)
			assert.Equal(t, tc.consumers, c.parser.consumers)
		})
	}
}

func TestCompiler_Compile(t *testing.T) {
	tests := []struct {
		name          string
		consumers     []*Consumer
		packages      string
		opts          ParseOptions
		expectedError string
	}{
		{
			name: "Success_SkipPackages",
			consumers: []*Consumer{
				{
					Name:    "tester",
					Package: func(*Package, string) bool { return false },
				},
			},
			packages: "./test/valid/...",
			opts: ParseOptions{
				SkipTestFiles: true,
			},
			expectedError: "",
		},
		{
			name: "Success_SkipFiles",
			consumers: []*Consumer{
				{
					Name:    "tester",
					Package: func(*Package, string) bool { return true },
					FilePre: func(*File, *goast.File) bool { return false },
				},
			},
			packages: "./test/valid/...",
			opts: ParseOptions{
				SkipTestFiles: true,
			},
			expectedError: "",
		},
		{
			name: "Success",
			consumers: []*Consumer{
				{
					Name:      "tester",
					Package:   func(*Package, string) bool { return true },
					FilePre:   func(*File, *goast.File) bool { return true },
					Import:    func(*File, *goast.ImportSpec) {},
					Struct:    func(*Type, *goast.StructType) {},
					Interface: func(*Type, *goast.InterfaceType) {},
					FuncType:  func(*Type, *goast.FuncType) {},
					FuncDecl:  func(*Func, *goast.FuncType, *goast.BlockStmt) {},
					FilePost:  func(*File, *goast.File) error { return nil },
				},
			},
			packages: "./test/valid/...",
			opts: ParseOptions{
				SkipTestFiles: true,
			},
			expectedError: "",
		},
		{
			name: "FilePostFails",
			consumers: []*Consumer{
				{
					Name:      "tester",
					Package:   func(*Package, string) bool { return true },
					FilePre:   func(*File, *goast.File) bool { return true },
					Import:    func(*File, *goast.ImportSpec) {},
					Struct:    func(*Type, *goast.StructType) {},
					Interface: func(*Type, *goast.InterfaceType) {},
					FuncType:  func(*Type, *goast.FuncType) {},
					FuncDecl:  func(*Func, *goast.FuncType, *goast.BlockStmt) {},
					FilePost:  func(*File, *goast.File) error { return errors.New("file error") },
				},
			},
			packages: "./test/valid/...",
			opts: ParseOptions{
				SkipTestFiles: true,
			},
			expectedError: "file error",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ui := ui.NewNop()
			c := NewCompiler(ui, tc.consumers...)

			err := c.Compile(tc.packages, tc.opts)

			if tc.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tc.expectedError)
			}
		})
	}
}
