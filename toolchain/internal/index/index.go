package index

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"sqlm/internal/parser"
	"strings"
)

// Location is a position within a file.
type Location struct {
	Path   string
	Line   int // 0-based
	Column int // 0-based
}

// Symbol is a named definition (type or function).
type Symbol struct {
	Name     string
	Kind     string // "type" or "function"
	Location Location
}

// Index holds all symbols and their references across a set of files.
type Index struct {
	Definitions map[string]*Symbol    // qualified name → symbol
	References  map[string][]Location // qualified name → all usage locations
}

var (
	reType     = regexp.MustCompile(`(?i)CREATE\s+TYPE\s+(\w+\.\w+)`)
	reFunction = regexp.MustCompile(`(?i)CREATE\s+(?:OR\s+REPLACE\s+)?FUNCTION\s+(\w+\.\w+)`)
	reIdent    = regexp.MustCompile(`\b(\w+\.\w+)\b`)
)

// Build constructs an index from a set of parsed files.
// It reads raw file content for accurate line numbers.
func Build(files []*parser.File) *Index {
	idx := &Index{
		Definitions: make(map[string]*Symbol),
		References:  make(map[string][]Location),
	}

	for _, f := range files {
		scanRawFile(idx, f.Path)
	}

	return idx
}

// BuildFromDir builds an index from an entities directory.
func BuildFromDir(entitiesDir string) (*Index, error) {
	mainPath := filepath.Join(entitiesDir, "main.sqlm")
	main, err := parser.Parse(mainPath)
	if err != nil {
		return nil, err
	}

	graph, err := parser.Load(entitiesDir)
	if err != nil {
		return nil, err
	}

	files, err := graph.Resolve(main.Imports...)
	if err != nil {
		return nil, err
	}

	return Build(files), nil
}

// scanRawFile scans a raw .sqlm file line by line for definitions and references.
// Reading the raw file ensures line numbers match what the editor sees.
func scanRawFile(idx *Index, path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	// first pass: collect definitions
	for lineNum, text := range lines {
		if m := reType.FindStringSubmatchIndex(text); m != nil {
			name := text[m[2]:m[3]]
			idx.Definitions[name] = &Symbol{
				Name: name,
				Kind: "type",
				Location: Location{Path: path, Line: lineNum, Column: m[2]},
			}
		}
		if m := reFunction.FindStringSubmatchIndex(text); m != nil {
			name := text[m[2]:m[3]]
			idx.Definitions[name] = &Symbol{
				Name: name,
				Kind: "function",
				Location: Location{Path: path, Line: lineNum, Column: m[2]},
			}
		}
	}

	// second pass: collect references
	for lineNum, text := range lines {
		for _, m := range reIdent.FindAllStringSubmatchIndex(text, -1) {
			name := text[m[2]:m[3]]
			if _, isDef := idx.Definitions[name]; isDef {
				// skip if this position IS the definition line and column
				if sym := idx.Definitions[name]; sym.Location.Path == path && sym.Location.Line == lineNum && sym.Location.Column == m[2] {
					continue
				}
				idx.References[name] = append(idx.References[name], Location{
					Path:   path,
					Line:   lineNum,
					Column: m[2],
				})
			}
		}
	}

	// deduplicate references
	for name, refs := range idx.References {
		idx.References[name] = dedupLocations(refs)
	}
}

func dedupLocations(locs []Location) []Location {
	seen := make(map[string]bool)
	var result []Location
	for _, loc := range locs {
		key := strings.Join([]string{loc.Path, string(rune(loc.Line)), string(rune(loc.Column))}, ":")
		if !seen[key] {
			seen[key] = true
			result = append(result, loc)
		}
	}
	return result
}
