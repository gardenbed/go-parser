// Package parser provides functionality to parse Go source code files
// and extract information about packages, types, and functions.
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
	Package   func(*Package, string) bool
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
	SkipTestFiles bool
	TypeFilter    TypeFilter
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

// Parse processes all Go source code files in the specified path.
// If the path ends with "/...", all subdirectories will be considered too.
func (p *parser) Parse(path string, opts ParseOptions) error {
	subDirs := strings.HasSuffix(path, "/...")
	if subDirs {
		path = strings.TrimSuffix(path, "/...")
	}

	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	if !info.IsDir() {
		return fmt.Errorf("%q is not a directory", path)
	}

	p.ui.Infof(ui.White, "Parsing ...")

	fset := gotoken.NewFileSet()

	module, err := getModuleName(path)
	if err != nil {
		return err
	}

	moduleInfo := Module{
		Name: module,
	}

	return visitPackages(subDirs, path, func(basePath, relPath string) error {
		absDir := filepath.Join(basePath, relPath)
		importPath := filepath.Join(module, relPath)

		p.ui.Debugf(ui.Cyan, "  Parsing directory: %s", absDir)

		entries, err := os.ReadDir(absDir)
		if err != nil {
			return fmt.Errorf("error on reading directory %s: %s", absDir, err)
		}

		// Parse all Go files in the current directory and build a map of package names to parsed files.
		files := make(map[string]map[string]*goast.File)
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") {
				continue
			}

			filename := filepath.Join(absDir, e.Name())

			file, err := goparser.ParseFile(fset, filename, nil, goparser.SkipObjectResolution|goparser.AllErrors)
			if err != nil {
				return err
			}

			pkgName := file.Name.Name
			if _, ok := files[pkgName]; !ok {
				files[pkgName] = make(map[string]*goast.File)
			}
			files[pkgName][filename] = file
		}

		// Visit all parsed Go files in each package
		for pkgName, pkgFiles := range files {
			p.ui.Debugf(ui.Magenta, "    Package: %s", pkgName)

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
					cont := c.Package(&pkgInfo, pkgName)
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

			for filename, file := range pkgFiles {
				if opts.SkipTestFiles && strings.HasSuffix(filename, "_test.go") {
					continue
				}

				if err := p.processFile(pkgInfo, fset, filename, file, fileConsumers, opts); err != nil {
					return err
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
