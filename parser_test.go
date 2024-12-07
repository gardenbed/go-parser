package parser

import (
	"errors"
	"regexp"
	"testing"

	goast "go/ast"

	"github.com/gardenbed/charm/ui"
	"github.com/stretchr/testify/assert"
)

func TestTypeInfo_IsExported(t *testing.T) {
	tests := []struct {
		name               string
		info               *Type
		expectedIsExported bool
	}{
		{
			name: "Exported",
			info: &Type{
				Name: "Controller",
			},
			expectedIsExported: true,
		},
		{
			name: "Unexported",
			info: &Type{
				Name: "controller",
			},
			expectedIsExported: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			isExported := tc.info.IsExported()

			assert.Equal(t, tc.expectedIsExported, isExported)
		})
	}
}

func TestFuncInfo_IsExported(t *testing.T) {
	tests := []struct {
		name               string
		info               *Func
		expectedIsExported bool
	}{
		{
			name: "Exported",
			info: &Func{
				Name: "Lookup",
			},
			expectedIsExported: true,
		},
		{
			name: "Unexported",
			info: &Func{
				Name: "lookup",
			},
			expectedIsExported: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			isExported := tc.info.IsExported()

			assert.Equal(t, tc.expectedIsExported, isExported)
		})
	}
}

func TestFuncInfo_IsMethod(t *testing.T) {
	tests := []struct {
		name             string
		info             *Func
		expectedIsMethod bool
	}{
		{
			name:             "Function",
			info:             &Func{},
			expectedIsMethod: false,
		},
		{
			name: "Method",
			info: &Func{
				RecvName: "Lookup",
				RecvType: &goast.StarExpr{
					X: &goast.Ident{Name: "service"},
				},
			},
			expectedIsMethod: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			isMethod := tc.info.IsMethod()

			assert.Equal(t, tc.expectedIsMethod, isMethod)
		})
	}
}

func TestParseOptions_MatchType(t *testing.T) {
	tests := []struct {
		name            string
		opts            ParseOptions
		typeName        *goast.Ident
		expectedMatched bool
	}{
		{
			name:            "Matched_NoFilter",
			opts:            ParseOptions{},
			typeName:        &goast.Ident{Name: "Request"},
			expectedMatched: true,
		},
		{
			name: "Matched_Name",
			opts: ParseOptions{
				TypeFilter: TypeFilter{
					Names: []string{"Response"},
				},
			},
			typeName:        &goast.Ident{Name: "Response"},
			expectedMatched: true,
		},
		{
			name: "Matched_Regexp",
			opts: ParseOptions{
				TypeFilter: TypeFilter{
					Regexp: regexp.MustCompile(`Service$`),
				},
			},
			typeName:        &goast.Ident{Name: "ExampleService"},
			expectedMatched: true,
		},
		{
			name: "NotMatched",
			opts: ParseOptions{
				TypeFilter: TypeFilter{
					Names:  []string{"Request", "Response"},
					Regexp: regexp.MustCompile(`Service$`),
				},
			},
			typeName:        &goast.Ident{Name: "service"},
			expectedMatched: false,
		},
		{
			name: "Matched_Exported",
			opts: ParseOptions{
				TypeFilter: TypeFilter{
					Exported: true,
				},
			},
			typeName:        &goast.Ident{Name: "Client"},
			expectedMatched: true,
		},
		{
			name: "NotMatched_Unexported",
			opts: ParseOptions{
				TypeFilter: TypeFilter{
					Exported: true,
				},
			},
			typeName:        &goast.Ident{Name: "client"},
			expectedMatched: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			matched := tc.opts.matchType(tc.typeName)

			assert.Equal(t, tc.expectedMatched, matched)
		})
	}
}

func TestParser_Parse(t *testing.T) {
	tests := []struct {
		name          string
		consumers     []*Consumer
		packages      string
		opts          ParseOptions
		expectedError string
	}{
		{
			name:          "PathNotExist",
			packages:      "/foo",
			opts:          ParseOptions{},
			expectedError: "stat /foo: no such file or directory",
		},
		{
			name:          "PathNotDirectory",
			packages:      "./test/valid/main.go",
			opts:          ParseOptions{},
			expectedError: `"./test/valid/main.go" is not a directory`,
		},
		{
			name:          "InvalidModule",
			packages:      "./test/invalid_module",
			opts:          ParseOptions{},
			expectedError: "invalid go.mod file: no module name found",
		},
		{
			name:          "InvalidCode",
			packages:      "./test/invalid_code",
			opts:          ParseOptions{},
			expectedError: "test/invalid_code/main.go:3:11: missing import path (and 10 more errors)",
		},
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
			packages:      "./test/valid/...",
			opts:          ParseOptions{},
			expectedError: "file error",
		},
		{
			name: "Success_SkipTestFiles",
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
			packages:      "./test/valid/...",
			opts:          ParseOptions{},
			expectedError: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := &parser{
				ui:        ui.NewNop(),
				consumers: tc.consumers,
			}

			err := p.Parse(tc.packages, tc.opts)

			if tc.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tc.expectedError)
			}
		})
	}
}
