package index

import (
	"bufio"
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
	Definitions map[string]*Symbol     // qualified name → symbol
	References  map[string][]Location  // qualified name → all usage locations
}

var (
	reType     = regexp.MustCompile(`(?i)CREATE\s+TYPE\s+(\w+\.\w+)`)
	reFunction = regexp.MustCompile(`(?i)CREATE\s+(?:OR\s+REPLACE\s+)?FUNCTION\s+(\w+\.\w+)`)
	reIdent    = regexp.MustCompile(`\b(\w+\.\w+)\b`)
)

// Build constructs an index from a set of parsed files.
func Build(files []*parser.File) *Index {
	idx := &Index{
		Definitions: make(map[string]*Symbol),
		References:  make(map[string][]Location),
	}

	for _, f := range files {
		scanFile(idx, f)
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

func scanFile(idx *Index, f *parser.File) {
	// scan init bodies and SQL body together
	var sources []string
	sources = append(sources, f.Inits...)
	if f.Body != "" {
		sources = append(sources, f.Body)
	}

	for _, src := range sources {
		scanDefinitions(idx, f.Path, src)
	}
	for _, src := range sources {
		scanReferences(idx, f.Path, src)
	}
}

func scanDefinitions(idx *Index, path, src string) {
	scanner := bufio.NewScanner(strings.NewReader(src))
	line := 0
	for scanner.Scan() {
		text := scanner.Text()

		if m := reType.FindStringSubmatchIndex(text); m != nil {
			name := text[m[2]:m[3]]
			idx.Definitions[name] = &Symbol{
				Name: name,
				Kind: "type",
				Location: Location{
					Path:   path,
					Line:   line,
					Column: m[2],
				},
			}
		}

		if m := reFunction.FindStringSubmatchIndex(text); m != nil {
			name := text[m[2]:m[3]]
			idx.Definitions[name] = &Symbol{
				Name: name,
				Kind: "function",
				Location: Location{
					Path:   path,
					Line:   line,
					Column: m[2],
				},
			}
		}

		line++
	}
}

func scanReferences(idx *Index, path, src string) {
	scanner := bufio.NewScanner(strings.NewReader(src))
	line := 0
	for scanner.Scan() {
		text := scanner.Text()
		matches := reIdent.FindAllStringSubmatchIndex(text, -1)
		for _, m := range matches {
			name := text[m[2]:m[3]]
			if _, isDef := idx.Definitions[name]; isDef {
				idx.References[name] = append(idx.References[name], Location{
					Path:   path,
					Line:   line,
					Column: m[2],
				})
			}
		}
		line++
	}
}
