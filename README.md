# go-parser

This repo provides an abstraction and an implementation for a Go parser.
It can be used for building all sorts of Go compilers such as interpreters, converters, code generators, and so forth.

## Quick Start

```go
package main

import (
  "go/ast"

  "github.com/gardenbed/charm/ui"
  "github.com/gardenbed/go-parser"
)

func main() {
  compiler := parser.NewCompiler(
    ui.New(ui.Debug),
    &parser.Consumer{
      Name:      "compiler",
      Package:   Package,
      FilePre:   FilePre,
      Import:    Import,
      Struct:    Struct,
      Interface: Interface,
      FuncType:  FuncType,
      FuncDecl:  FuncDecl,
      FilePost:  FilePost,
    },
  )

  if err := compiler.Compile("...", parser.ParseOptions{}); err != nil {
    panic(err)
  }
}

func Package(*parser.Package, *ast.Package) bool {
  return true
}

func FilePre(*parser.File, *ast.File) bool {
  return true
}

func Import(*parser.File, *ast.ImportSpec)                 {}
func Struct(*parser.Type, *ast.StructType)                 {}
func Interface(*parser.Type, *ast.InterfaceType)           {}
func FuncType(*parser.Type, *ast.FuncType)                 {}
func FuncDecl(*parser.Func, *ast.FuncType, *ast.BlockStmt) {}

func FilePost(*parser.File, *ast.File) error {
  return nil
}
```
