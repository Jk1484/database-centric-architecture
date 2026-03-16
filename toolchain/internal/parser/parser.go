package parser

import (
	"fmt"
	"os"
	"strings"
)

// Parse reads and parses a single .sqlm file.
func Parse(path string) (*File, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	s := newScanner(string(raw), path)
	file := &File{Path: path}

	// Phase 1: package and import declarations.
	// These must appear before any SQL or func init blocks.
	if err := parseDirectives(s, file); err != nil {
		return nil, err
	}

	// Phase 2: func init() blocks and raw SQL body, interleaved.
	if err := parseBody(s, file); err != nil {
		return nil, err
	}

	return file, nil
}

// parseDirectives reads the package declaration and any import declarations.
func parseDirectives(s *scanner, file *File) error {
	s.skipWhitespace()
	skipLineComments(s)

	// package
	if !s.matchKeyword("package") {
		return fmt.Errorf("%s:%d: expected package declaration", s.path, s.line)
	}
	s.consumeN(7) // "package"
	s.skipWhitespace()
	name := s.readIdent()
	if name == "" {
		return fmt.Errorf("%s:%d: expected package name", s.path, s.line)
	}
	file.Package = name

	// imports
	for {
		s.skipWhitespace()
		skipLineComments(s)
		if !s.matchKeyword("import") {
			break
		}
		s.consumeN(6) // "import"
		s.skipWhitespace()
		if s.eof() || s.peek() != '"' {
			return fmt.Errorf("%s:%d: expected quoted import path", s.path, s.line)
		}
		s.advance() // opening "
		imp, err := s.readQuotedString()
		if err != nil {
			return err
		}
		file.Imports = append(file.Imports, imp)
	}

	return nil
}

// parseBody reads the rest of the file, extracting func init() blocks and
// collecting everything else as raw SQL body.
func parseBody(s *scanner, file *File) error {
	var body strings.Builder

	for !s.eof() {
		if s.isFuncInit() {
			s.tryConsumeFuncInit() // advance past `func init() {`
			initBody, err := s.readBraceBlock()
			if err != nil {
				return err
			}
			file.Inits = append(file.Inits, initBody)
			continue
		}

		// everything else is raw SQL — read one unit at a time
		unit, err := readSQLUnit(s)
		if err != nil {
			return err
		}
		body.WriteString(unit)
	}

	file.Body = strings.TrimSpace(body.String())
	return nil
}

// readSQLUnit reads one "unit" of raw SQL — handling $$ blocks, strings,
// and comments as opaque, stopping before func init() patterns.
func readSQLUnit(s *scanner) (string, error) {
	var b strings.Builder
	ch := s.advance()

	// $$ ... $$ block
	if ch == '$' && !s.eof() && s.peek() == '$' {
		s.advance()
		b.WriteString("$$")
		for !s.eof() {
			c := s.advance()
			b.WriteRune(c)
			if c == '$' && !s.eof() && s.peek() == '$' {
				s.advance()
				b.WriteString("$")
				break
			}
		}
		return b.String(), nil
	}

	// ' ... ' string
	if ch == '\'' {
		b.WriteRune(ch)
		for !s.eof() {
			c := s.advance()
			b.WriteRune(c)
			if c == '\'' {
				break
			}
		}
		return b.String(), nil
	}

	// -- comment
	if ch == '-' && !s.eof() && s.peek() == '-' {
		s.advance()
		b.WriteString("--")
		for !s.eof() && s.peek() != '\n' {
			b.WriteRune(s.advance())
		}
		b.WriteRune('\n')
		return b.String(), nil
	}

	b.WriteRune(ch)
	return b.String(), nil
}

// skipLineComments skips any lines that are purely comments or whitespace.
func skipLineComments(s *scanner) {
	for {
		s.skipWhitespace()
		if s.eof() {
			return
		}
		if s.peek() == '-' && s.pos+1 < len(s.src) && s.src[s.pos+1] == '-' {
			s.advance()
			s.advance()
			s.skipLineComment()
			continue
		}
		return
	}
}
