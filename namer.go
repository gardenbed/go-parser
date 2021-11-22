package parser

import (
	"fmt"
	"go/ast"
	"regexp"
	"strings"
)

var (
	re1 = regexp.MustCompile(`^[a-z]`)
	re2 = regexp.MustCompile(`^[A-Z]+$`)
	re3 = regexp.MustCompile(`^[A-Z][0-9a-z_]`)
	re4 = regexp.MustCompile(`^([A-Z]+)[A-Z][0-9a-z_]`)
)

// IsExported determines whether or not a given name is exported.
func IsExported(name string) bool {
	first := name[0:1]
	return first == strings.ToUpper(first)
}

// InferName infers an identifier name from a type expression.
func InferName(expr ast.Expr) string {
	switch v := expr.(type) {
	case *ast.ArrayType:
		return InferName(v.Elt) + "Vals"
	case *ast.MapType:
		return InferName(v.Key) + strings.Title(InferName(v.Value)) + "Map"
	case *ast.ChanType:
		return InferName(v.Value) + "Ch"
	case *ast.StructType:
		return "structV"
	case *ast.InterfaceType:
		return "interfaceV"
	}

	var lastName string
	ast.Inspect(expr, func(n ast.Node) bool {
		if id, ok := n.(*ast.Ident); ok {
			lastName = id.Name
		}
		return true
	})

	return lastName
}

// ConvertToUnexported converts an exported identifier to an unexported one.
func ConvertToUnexported(name string) string {
	switch {
	// Unexported (e.g. client --> client)
	case re1.MatchString(name):
		return name

	// All in upper letters (e.g. ID --> id)
	case re2.MatchString(name):
		return strings.ToLower(name)

	// Starts with Title case (e.g. Request --> request)
	case re3.MatchString(name):
		return strings.ToLower(name[0:1]) + name[1:]

	// Starts with all upper letters followed by a Title case (e.g. HTTPRequest --> httpRequest)
	case re4.MatchString(name):
		m := re4.FindStringSubmatch(name)
		if len(m) == 2 {
			l := len(m[1])
			return strings.ToLower(name[0:l]) + name[l:]
		}
	}

	panic(fmt.Sprintf("ConvertToUnexported: unexpected identifer: %s", name))
}
