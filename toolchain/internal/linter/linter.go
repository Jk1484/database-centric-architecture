package linter

import (
	"fmt"
	"os"
	"path/filepath"
	"sqlm/internal/parser"
	"strings"
)

// Issue represents a single lint finding.
type Issue struct {
	Path    string
	Message string
}

func (i Issue) String() string {
	return fmt.Sprintf("%s: %s", i.Path, i.Message)
}

// Lint runs all checks against the entities directory and returns any issues found.
func Lint(entitiesDir string) ([]Issue, error) {
	var issues []Issue

	mainPath := filepath.Join(entitiesDir, "main.sqlm")
	main, err := parser.Parse(mainPath)
	if err != nil {
		return nil, fmt.Errorf("main.sqlm: %w", err)
	}

	graph, err := parser.Load(entitiesDir)
	if err != nil {
		return nil, err
	}

	// check imported packages exist
	issues = append(issues, checkImportsExist(main, graph)...)

	// resolve reachable files
	reachable, err := graph.Resolve(main.Imports...)
	if err != nil {
		// circular import or unknown package — already caught above
		return issues, nil
	}

	// check for unreachable files
	unreachable, err := findUnreachable(entitiesDir, reachable)
	if err != nil {
		return nil, err
	}
	issues = append(issues, unreachable...)

	return issues, nil
}

// checkImportsExist verifies every import in main.sqlm resolves to a known package.
func checkImportsExist(main *parser.File, graph *parser.Graph) []Issue {
	var issues []Issue
	for _, imp := range main.Imports {
		if !graph.HasPackage(imp) {
			issues = append(issues, Issue{
				Path:    main.Path,
				Message: fmt.Sprintf("imported package %q not found", imp),
			})
		}
	}
	return issues
}

// findUnreachable returns issues for any .sqlm files not reachable from main.sqlm.
func findUnreachable(entitiesDir string, reachable []*parser.File) ([]Issue, error) {
	reachableSet := make(map[string]bool)
	for _, f := range reachable {
		reachableSet[f.Path] = true
	}

	var issues []Issue
	err := filepath.WalkDir(entitiesDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".sqlm") {
			return nil
		}
		// skip main.sqlm itself
		if filepath.Base(path) == "main.sqlm" {
			return nil
		}
		if !reachableSet[path] {
			issues = append(issues, Issue{
				Path:    path,
				Message: "file is not reachable from main.sqlm",
			})
		}
		return nil
	})

	return issues, err
}
