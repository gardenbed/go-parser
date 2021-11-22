[![Go Doc][godoc-image]][godoc-url]
[![Build Status][workflow-image]][workflow-url]
[![Go Report Card][goreport-image]][goreport-url]
[![Test Coverage][codecov-image]][codecov-url]

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


[godoc-url]: https://pkg.go.dev/github.com/gardenbed/go-parser
[godoc-image]: https://pkg.go.dev/badge/github.com/gardenbed/go-parser
[workflow-url]: https://github.com/gardenbed/go-parser/actions
[workflow-image]: https://github.com/gardenbed/go-parser/workflows/Go/badge.svg
[goreport-url]: https://goreportcard.com/report/github.com/gardenbed/go-parser
[goreport-image]: https://goreportcard.com/badge/github.com/gardenbed/go-parser
[codecov-url]: https://codecov.io/gh/gardenbed/go-parser
[codecov-image]: https://codecov.io/gh/gardenbed/go-parser/branch/main/graph/badge.svg
