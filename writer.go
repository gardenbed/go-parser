package parser

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	"io/ioutil"
	"os"
	"path/filepath"

	"golang.org/x/tools/imports"
)

func getDebugFilename(path string) string {
	filename := filepath.Base(path)
	ext := filepath.Ext(filename)
	name := filename[0 : len(filename)-len(ext)]
	return fmt.Sprintf("%s-debug.log", name)
}

// WriteFile formats and writes a Go source code file to disk.
func WriteFile(path string, fset *token.FileSet, file *ast.File) error {
	buf := new(bytes.Buffer)
	if err := format.Node(buf, fset, file); err != nil {
		return fmt.Errorf("gofmt error: %s", err)
	}

	// Format the modified Go file
	b, err := imports.Process(path, buf.Bytes(), &imports.Options{
		TabWidth:  8,
		TabIndent: true,
		Comments:  true,
		Fragment:  true,
	})

	if err != nil {
		// Write a log file for debugging purposes
		_ = ioutil.WriteFile(getDebugFilename(path), buf.Bytes(), 0644)
		return fmt.Errorf("goimports error: %s", err)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}

	if _, err := f.Write(b); err != nil {
		return err
	}

	return nil
}
