package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	goast "go/ast"
	goparser "go/parser"
	gotoken "go/token"

	"github.com/gardenbed/charm/ui"
)

// Module contains information about a Go module.
type Module struct {
	Name string
}

// Package contains information about a parsed package.
type Package struct {
	Module
	Name        string
	ImportPath  string
	BaseDir     string
	RelativeDir string
}

// File contains information about a parsed file.
type File struct {
	Package
	*gotoken.FileSet
	Name string
}

// Type contains information about a parsed type.
type Type struct {
	File
	Name string
}

// IsExported determines whether or not a type is exported.
func (t *Type) IsExported() bool {
	return IsExported(t.Name)
}

// Func contains information about a parsed function.
type Func struct {
	File
	Name     string
	RecvName string
	RecvType goast.Expr
}

// IsExported determines whether or not a function is exported.
func (f *Func) IsExported() bool {
	return IsExported(f.Name)
}

// IsMethod determines if a function is a method of a struct.
func (f *Func) IsMethod() bool {
	return f.RecvName != "" && f.RecvType != nil
}

// Consumer is used for processing AST nodes.
// This is meant to be provided by downstream packages.
type Consumer struct {
	Name      string
	Package   func(*Package, *goast.Package) bool
	FilePre   func(*File, *goast.File) bool
	Import    func(*File, *goast.ImportSpec)
	Struct    func(*Type, *goast.StructType)
	Interface func(*Type, *goast.InterfaceType)
	FuncType  func(*Type, *goast.FuncType)
	FuncDecl  func(*Func, *goast.FuncType, *goast.BlockStmt)
	FilePost  func(*File, *goast.File) error
}

type TypeFilter struct {
	// Exported filters unexported types.
	Exported bool
	// Names filters types based on their names.
	Names []string
	// Regexp filters types based on a regular expression.
	Regexp *regexp.Regexp
}

// ParseOptions configure how Go source code files should be parsed.
type ParseOptions struct {
	MergePackageFiles bool
	SkipTestFiles     bool
	TypeFilter        TypeFilter
}

// matchType determines if a type is matching the provided options.
func (o ParseOptions) matchType(name *goast.Ident) bool {
	// If no filter specified, it is a match
	if len(o.TypeFilter.Names) == 0 && o.TypeFilter.Regexp == nil {
		return !o.TypeFilter.Exported || IsExported(name.Name)
	}

	// Name takes precedence over regexp
	for _, t := range o.TypeFilter.Names {
		if name.Name == t {
			return !o.TypeFilter.Exported || IsExported(name.Name)
		}
	}

	if o.TypeFilter.Regexp != nil && o.TypeFilter.Regexp.MatchString(name.Name) {
		return !o.TypeFilter.Exported || IsExported(name.Name)
	}

	return false
}

// Parser is used for parsing Go source code files.
type parser struct {
	ui        ui.UI
	consumers []*Consumer
}

// Parse parses all Go source code files recursively in the given packages.
func (p *parser) Parse(packages string, opts ParseOptions) error {
	var includeSubs bool
	var path string

	if strings.HasSuffix(packages, "/...") {
		includeSubs, path = true, strings.TrimSuffix(packages, "/...")
	} else {
		includeSubs, path = false, packages
	}

	// Verify the path
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	if !info.IsDir() {
		return fmt.Errorf("%q is not a package", path)
	}

	p.ui.Infof(ui.White, "Parsing ...")

	module, err := getModuleName(path)
	if err != nil {
		return err
	}

	moduleInfo := Module{
		Name: module,
	}

	// Create a new file set for each package
	fset := gotoken.NewFileSet()

	return visitPackages(includeSubs, path, func(basePath, relPath string) error {
		pkgDir := filepath.Join(basePath, relPath)
		importPath := filepath.Join(module, relPath)

		// Parse all Go packages and files in the currecnt directory
		p.ui.Debugf(ui.Cyan, "  Parsing directory: %s", pkgDir)
		pkgs, err := goparser.ParseDir(fset, pkgDir, nil, goparser.AllErrors)
		if err != nil {
			return err
		}

		// Visit all parsed Go files in the current directory
		for pkgName, pkg := range pkgs {
			p.ui.Debugf(ui.Magenta, "    Package: %s", pkg.Name)

			pkgInfo := Package{
				Module:      moduleInfo,
				Name:        pkgName,
				ImportPath:  importPath,
				BaseDir:     basePath,
				RelativeDir: relPath,
			}

			// Keeps track of interested consumers in the files in the current package
			fileConsumers := make([]*Consumer, 0)

			// PACKAGE
			for _, c := range p.consumers {
				if c.Package != nil {
					cont := c.Package(&pkgInfo, pkg)
					if cont {
						fileConsumers = append(fileConsumers, c)
					}
					p.ui.Tracef(ui.Blue, "      %s.Package: %t", c.Name, cont)
				}
			}

			// Proceed to the next package if no consumer
			if len(fileConsumers) == 0 {
				continue
			}

			// Merge all file ASTs in the package and process a single file
			if opts.MergePackageFiles {
				mergedFile := goast.MergePackageFiles(pkg, goast.FilterImportDuplicates|goast.FilterUnassociatedComments)
				if err := p.processFile(pkgInfo, fset, "merged.go", mergedFile, fileConsumers, opts); err != nil {
					return err
				}
			} else {
				for fileName, file := range pkg.Files {
					if opts.SkipTestFiles && strings.HasSuffix(fileName, "_test.go") {
						continue
					}

					if err := p.processFile(pkgInfo, fset, fileName, file, fileConsumers, opts); err != nil {
						return err
					}
				}
			}
		}

		return nil
	})
}

func (p *parser) processFile(pkgInfo Package, fset *gotoken.FileSet, fileName string, file *goast.File, fileConsumers []*Consumer, opts ParseOptions) error {
	p.ui.Debugf(ui.Green, "      File: %s", fileName)

	fileInfo := File{
		Package: pkgInfo,
		FileSet: fset,
		Name:    filepath.Base(fileName),
	}

	// Keeps track of interested consumers in the declarations in the current file
	declConsumers := make([]*Consumer, 0)

	// FILE (pre)
	for _, c := range fileConsumers {
		if c.FilePre != nil {
			cont := c.FilePre(&fileInfo, file)
			if cont {
				declConsumers = append(declConsumers, c)
			}
			p.ui.Tracef(ui.Blue, "        %s.FilePre: %t", c.Name, cont)
		}
	}

	// Proceed to the next file if no consumer
	if len(declConsumers) == 0 {
		return nil
	}

	goast.Inspect(file, func(n goast.Node) bool {
		switch v := n.(type) {
		// IMPORT
		case *goast.ImportSpec:
			p.ui.Debugf(ui.Yellow, "          ImportSpec: %s", v.Path.Value)
			for _, c := range declConsumers {
				if c.Import != nil {
					c.Import(&fileInfo, v)
					p.ui.Tracef(ui.Blue, "            %s.Import", c.Name)
				}
			}
			return false

		// Handle Types
		case *goast.TypeSpec:
			typeInfo := Type{
				File: fileInfo,
				Name: v.Name.Name,
			}

			switch w := v.Type.(type) {
			// STRUCT
			case *goast.StructType:
				p.ui.Debugf(ui.Yellow, "          StructType: %s", v.Name.Name)
				for _, c := range declConsumers {
					if c.Struct != nil {
						if opts.matchType(v.Name) {
							c.Struct(&typeInfo, w)
							p.ui.Tracef(ui.Blue, "            %s.Struct", c.Name)
						}
					}
				}
				return false

			// INTERFACE
			case *goast.InterfaceType:
				p.ui.Debugf(ui.Yellow, "          InterfaceType: %s", v.Name.Name)
				for _, c := range declConsumers {
					if c.Interface != nil {
						if opts.matchType(v.Name) {
							c.Interface(&typeInfo, w)
							p.ui.Tracef(ui.Blue, "            %s.Interface", c.Name)
						}
					}
				}
				return false

			// FUNCTION (type)
			case *goast.FuncType:
				p.ui.Debugf(ui.Yellow, "          FuncType: %s", v.Name.Name)
				for _, c := range declConsumers {
					if c.FuncType != nil {
						if opts.matchType(v.Name) {
							c.FuncType(&typeInfo, w)
							p.ui.Tracef(ui.Blue, "            %s.FuncType", c.Name)
						}
					}
				}
				return false
			}

		// FUNCTION (declaration)
		case *goast.FuncDecl:
			p.ui.Debugf(ui.Yellow, "          FuncDecl: %s", v.Name.Name)

			funcInfo := Func{
				File: fileInfo,
				Name: v.Name.Name,
			}

			if v.Recv != nil && len(v.Recv.List) == 1 {
				if len(v.Recv.List[0].Names) == 1 {
					funcInfo.RecvName = v.Recv.List[0].Names[0].Name
				}
				funcInfo.RecvType = v.Recv.List[0].Type
			}

			for _, c := range declConsumers {
				if c.FuncDecl != nil {
					c.FuncDecl(&funcInfo, v.Type, v.Body)
					p.ui.Tracef(ui.Blue, "            %s.FuncDecl", c.Name)
				}
			}

			return false
		}

		return true
	})

	// FILE (post)
	for _, c := range declConsumers {
		if c.FilePost != nil {
			err := c.FilePost(&fileInfo, file)
			if err != nil {
				return err
			}
			p.ui.Tracef(ui.Blue, "        %s.FilePost", c.Name)
		}
	}

	return nil
}
