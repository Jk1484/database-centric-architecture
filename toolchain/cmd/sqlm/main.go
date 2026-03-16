package main

import (
	"fmt"
	"os"
	"sqlm/internal/compiler"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: sqlm <command> [args]")
		fmt.Fprintln(os.Stderr, "commands: build, lint, lsp")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "build":
		cmdBuild()
	case "lint":
		fmt.Println("lint: not yet implemented")
	case "lsp":
		fmt.Println("lsp: not yet implemented")
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
}

// cmdBuild compiles migrations/entities/ to a single .sql file.
// Usage: sqlm build <entities-dir> <out-file>
func cmdBuild() {
	if len(os.Args) < 4 {
		fmt.Fprintln(os.Stderr, "usage: sqlm build <entities-dir> <out-file>")
		os.Exit(1)
	}
	entitiesDir := os.Args[2]
	outFile := os.Args[3]

	if err := compiler.CompileToFile(entitiesDir, outFile); err != nil {
		fmt.Fprintf(os.Stderr, "build failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("compiled %s → %s\n", entitiesDir, outFile)
}
