package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Graph holds all parsed packages indexed by package name.
type Graph struct {
	packages map[string][]*File // package name → files in that package
	dirs     map[string]string  // package name → directory path
}

// Load builds a graph by scanning a root directory for packages.
// Each subdirectory is treated as a package.
func Load(root string) (*Graph, error) {
	g := &Graph{
		packages: make(map[string][]*File),
		dirs:     make(map[string]string),
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		pkg := entry.Name()
		dir := filepath.Join(root, pkg)
		g.dirs[pkg] = dir

		files, err := loadPackage(dir, pkg)
		if err != nil {
			return nil, err
		}
		g.packages[pkg] = files
	}

	return g, nil
}

// loadPackage reads all .sqlm files from a directory and parses them.
func loadPackage(dir, expectedPkg string) ([]*File, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var files []*File
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sqlm") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		file, err := Parse(path)
		if err != nil {
			return nil, err
		}
		if file.Package != expectedPkg {
			return nil, fmt.Errorf("%s: package %q does not match directory %q", path, file.Package, expectedPkg)
		}
		files = append(files, file)
	}

	return files, nil
}

// HasPackage reports whether a package with the given name exists in the graph.
func (g *Graph) HasPackage(name string) bool {
	_, ok := g.packages[name]
	return ok
}

// Resolve traverses the graph from one or more entry packages and returns
// all files in dependency order. Each package is visited at most once.
func (g *Graph) Resolve(entryPkgs ...string) ([]*File, error) {
	visited := make(map[string]bool)
	inStack := make(map[string]bool)
	var ordered []*File

	var visit func(pkg string) error
	visit = func(pkg string) error {
		if visited[pkg] {
			return nil
		}
		if inStack[pkg] {
			return fmt.Errorf("circular import detected involving package %q", pkg)
		}

		files, ok := g.packages[pkg]
		if !ok {
			return fmt.Errorf("unknown package %q", pkg)
		}

		inStack[pkg] = true

		seen := make(map[string]bool)
		for _, f := range files {
			for _, imp := range f.Imports {
				if !seen[imp] {
					seen[imp] = true
					if err := visit(imp); err != nil {
						return err
					}
				}
			}
		}

		inStack[pkg] = false
		visited[pkg] = true
		ordered = append(ordered, files...)
		return nil
	}

	for _, pkg := range entryPkgs {
		if err := visit(pkg); err != nil {
			return nil, err
		}
	}

	return ordered, nil
}
