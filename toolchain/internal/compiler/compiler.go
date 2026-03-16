package compiler

import (
	"fmt"
	"os"
	"path/filepath"
	"sqlm/internal/parser"
	"strings"
)

// Compile resolves and compiles all .sqlm files rooted at entitiesDir
// into a single SQL string, starting from main.sqlm.
func Compile(entitiesDir string) (string, error) {
	mainPath := filepath.Join(entitiesDir, "main.sqlm")
	main, err := parser.Parse(mainPath)
	if err != nil {
		return "", fmt.Errorf("main.sqlm: %w", err)
	}

	graph, err := parser.Load(entitiesDir)
	if err != nil {
		return "", err
	}

	files, err := graph.Resolve(main.Imports...)
	if err != nil {
		return "", err
	}

	return emit(files), nil
}

// CompileToFile compiles and writes the result to outPath.
func CompileToFile(entitiesDir, outPath string) error {
	sql, err := Compile(entitiesDir)
	if err != nil {
		return err
	}
	return os.WriteFile(outPath, []byte(sql), 0644)
}

// emit produces the final SQL from an ordered list of files.
func emit(files []*parser.File) string {
	var out strings.Builder
	seen := make(map[string]bool) // track schemas already emitted

	for _, f := range files {
		// emit CREATE SCHEMA once per package
		if !seen[f.Package] {
			seen[f.Package] = true
			fmt.Fprintf(&out, "CREATE SCHEMA IF NOT EXISTS %s;\n\n", f.Package)
		}

		// emit func init() bodies
		for _, init := range f.Inits {
			out.WriteString(init)
			out.WriteString("\n\n")
		}

		// emit raw SQL body
		if f.Body != "" {
			out.WriteString(f.Body)
			out.WriteString("\n\n")
		}
	}

	return strings.TrimSpace(out.String())
}
