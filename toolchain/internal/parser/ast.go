package parser

// File is the AST for a single .sqlm source file.
type File struct {
	Path    string
	Package string
	Imports []string
	Inits   []string // bodies of func init() blocks, in order
	Body    string   // raw SQL — everything that is not a directive
}
