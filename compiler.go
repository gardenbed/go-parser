package parser

import "github.com/gardenbed/charm/ui"

// Compiler is used for parsing Go source code files and compiling new source code files.
type Compiler struct {
	parser *parser
}

// NewCompiler creates a new compiler.
// This is meant to be used by downstream packages that provide consumers.
func NewCompiler(ui ui.UI, consumers ...*Consumer) *Compiler {
	return &Compiler{
		parser: &parser{
			ui:        ui,
			consumers: consumers,
		},
	}
}

// Compile parses all Go source code files in a given path and generates new artifacts (source codes).
func (c *Compiler) Compile(path string, opts ParseOptions) error {
	return c.parser.Parse(path, opts)
}
