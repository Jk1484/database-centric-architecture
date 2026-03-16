package lsp

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sqlm/internal/index"
	"strings"
)

// Handler dispatches LSP requests using the symbol index.
type Handler struct {
	idx         *index.Index
	entitiesDir string
}

func NewHandler(entitiesDir string) (*Handler, error) {
	idx, err := index.BuildFromDir(entitiesDir)
	if err != nil {
		return nil, err
	}
	return &Handler{idx: idx, entitiesDir: entitiesDir}, nil
}

func (h *Handler) Handle(msg map[string]json.RawMessage) any {
	var method string
	json.Unmarshal(msg["method"], &method)

	var id any
	json.Unmarshal(msg["id"], &id)

	switch method {
	case "initialize":
		return response(id, InitializeResult{
			Capabilities: ServerCapabilities{
				DefinitionProvider: true,
				ReferencesProvider: true,
				HoverProvider:      true,
			},
		})

	case "initialized", "$/cancelRequest":
		return nil // notifications, no response

	case "shutdown":
		return response(id, nil)

	case "exit":
		os.Exit(0)

	case "textDocument/definition":
		var params TextDocumentPositionParams
		json.Unmarshal(msg["params"], &params)
		return response(id, h.definition(params))

	case "textDocument/references":
		var params ReferenceParams
		json.Unmarshal(msg["params"], &params)
		return response(id, h.references(params))

	case "textDocument/hover":
		var params TextDocumentPositionParams
		json.Unmarshal(msg["params"], &params)
		return response(id, h.hover(params))
	}

	return nil
}

func (h *Handler) definition(params TextDocumentPositionParams) any {
	name := h.symbolAtPosition(params.TextDocument.URI, params.Position)
	if name == "" {
		return nil
	}
	sym, ok := h.idx.Definitions[name]
	if !ok {
		return nil
	}
	return locationToLSP(sym.Location)
}

func (h *Handler) references(params ReferenceParams) any {
	name := h.symbolAtPosition(params.TextDocument.URI, params.TextDocumentPositionParams.Position)
	if name == "" {
		return nil
	}
	refs, ok := h.idx.References[name]
	if !ok {
		return nil
	}
	var locs []LSPLocation
	for _, ref := range refs {
		locs = append(locs, locationToLSP(ref))
	}
	return locs
}

func (h *Handler) hover(params TextDocumentPositionParams) any {
	name := h.symbolAtPosition(params.TextDocument.URI, params.Position)
	if name == "" {
		return nil
	}
	sym, ok := h.idx.Definitions[name]
	if !ok {
		return nil
	}
	return Hover{
		Contents: MarkupContent{
			Kind:  "markdown",
			Value: fmt.Sprintf("**%s** `%s`", sym.Kind, sym.Name),
		},
	}
}

// symbolAtPosition reads the file at uri and returns the schema.name
// identifier under the given position, if any.
func (h *Handler) symbolAtPosition(uri string, pos Position) string {
	path := uriToPath(uri)
	content, err := os.ReadFile(path)
	if err != nil {
		return ""
	}

	lines := strings.Split(string(content), "\n")
	if pos.Line >= len(lines) {
		return ""
	}
	line := lines[pos.Line]
	col := pos.Character
	if col >= len(line) {
		return ""
	}

	// expand left and right to find word boundaries (letters, digits, _, .)
	isIdent := func(ch byte) bool {
		return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') ||
			(ch >= '0' && ch <= '9') || ch == '_' || ch == '.'
	}

	start := col
	for start > 0 && isIdent(line[start-1]) {
		start--
	}
	end := col
	for end < len(line) && isIdent(line[end]) {
		end++
	}

	word := line[start:end]
	if !strings.Contains(word, ".") {
		return ""
	}
	return word
}

func locationToLSP(loc index.Location) LSPLocation {
	return LSPLocation{
		URI: pathToURI(loc.Path),
		Range: Range{
			Start: Position{Line: loc.Line, Character: loc.Column},
			End:   Position{Line: loc.Line, Character: loc.Column + 1},
		},
	}
}

func response(id any, result any) *Response {
	return &Response{JSONRPC: "2.0", ID: id, Result: result}
}

func pathToURI(path string) string {
	abs, _ := filepath.Abs(path)
	return "file://" + abs
}

func uriToPath(uri string) string {
	u, err := url.Parse(uri)
	if err != nil {
		return uri
	}
	return u.Path
}
