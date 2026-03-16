package parser

import (
	"fmt"
	"strings"
	"unicode"
)

type scanner struct {
	src  []rune
	pos  int
	line int
	path string
}

func newScanner(src, path string) *scanner {
	return &scanner{src: []rune(src), path: path, line: 1}
}

func (s *scanner) eof() bool {
	return s.pos >= len(s.src)
}

func (s *scanner) peek() rune {
	if s.eof() {
		return 0
	}
	return s.src[s.pos]
}

func (s *scanner) advance() rune {
	ch := s.src[s.pos]
	s.pos++
	if ch == '\n' {
		s.line++
	}
	return ch
}

func (s *scanner) skipWhitespace() {
	for !s.eof() && unicode.IsSpace(s.peek()) {
		s.advance()
	}
}

// skipLineComment skips from current position to end of line.
func (s *scanner) skipLineComment() {
	for !s.eof() && s.peek() != '\n' {
		s.advance()
	}
}

// matchKeyword checks if the next characters match kw followed by a
// non-identifier character (word boundary). Does not advance.
func (s *scanner) matchKeyword(kw string) bool {
	runes := []rune(kw)
	if s.pos+len(runes) > len(s.src) {
		return false
	}
	for i, r := range runes {
		if s.src[s.pos+i] != r {
			return false
		}
	}
	after := s.pos + len(runes)
	if after < len(s.src) {
		next := s.src[after]
		if unicode.IsLetter(next) || unicode.IsDigit(next) || next == '_' {
			return false
		}
	}
	return true
}

func (s *scanner) consumeN(n int) {
	for i := 0; i < n; i++ {
		s.advance()
	}
}

// readIdent reads an identifier from the current position.
func (s *scanner) readIdent() string {
	var b strings.Builder
	for !s.eof() {
		ch := s.peek()
		if !unicode.IsLetter(ch) && !unicode.IsDigit(ch) && ch != '_' {
			break
		}
		b.WriteRune(s.advance())
	}
	return b.String()
}

// readQuotedString reads a "-quoted string. Opening " must already be consumed.
func (s *scanner) readQuotedString() (string, error) {
	var b strings.Builder
	for !s.eof() {
		ch := s.advance()
		if ch == '"' {
			return b.String(), nil
		}
		b.WriteRune(ch)
	}
	return "", fmt.Errorf("%s:%d: unterminated string", s.path, s.line)
}

// readBraceBlock reads raw content from after an opening { until the
// matching }, handling $$, '', and -- correctly.
func (s *scanner) readBraceBlock() (string, error) {
	var b strings.Builder
	depth := 1

	for !s.eof() {
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
			continue
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
			continue
		}

		// -- comment
		if ch == '-' && !s.eof() && s.peek() == '-' {
			s.advance()
			b.WriteString("--")
			for !s.eof() && s.peek() != '\n' {
				b.WriteRune(s.advance())
			}
			b.WriteRune('\n')
			continue
		}

		if ch == '{' {
			depth++
			b.WriteRune(ch)
			continue
		}
		if ch == '}' {
			depth--
			if depth == 0 {
				return strings.TrimSpace(b.String()), nil
			}
			b.WriteRune(ch)
			continue
		}

		b.WriteRune(ch)
	}

	return "", fmt.Errorf("%s:%d: unclosed block", s.path, s.line)
}

// tryConsumeFuncInit attempts to match and consume `func init() {`.
// Returns true and leaves the scanner positioned after `{` on success.
// Restores position on failure.
func (s *scanner) tryConsumeFuncInit() bool {
	saved := s.pos
	savedLine := s.line

	restore := func() bool {
		s.pos = saved
		s.line = savedLine
		return false
	}

	s.skipWhitespace()

	if !s.matchKeyword("func") {
		return restore()
	}
	s.consumeN(4) // "func"
	s.skipWhitespace()

	if !s.matchKeyword("init") {
		return restore()
	}
	s.consumeN(4) // "init"
	s.skipWhitespace()

	if s.eof() || s.peek() != '(' {
		return restore()
	}
	s.advance() // (
	s.skipWhitespace()

	if s.eof() || s.peek() != ')' {
		return restore()
	}
	s.advance() // )
	s.skipWhitespace()

	if s.eof() || s.peek() != '{' {
		return restore()
	}
	s.advance() // {

	return true
}

// isFuncInit peeks ahead to check if current position starts a func init()
// without consuming anything.
func (s *scanner) isFuncInit() bool {
	saved := s.pos
	savedLine := s.line
	result := s.tryConsumeFuncInit()
	s.pos = saved
	s.line = savedLine
	return result
}
