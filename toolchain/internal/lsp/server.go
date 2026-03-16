package lsp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// Server reads JSON-RPC requests from stdin and writes responses to stdout.
type Server struct {
	handler *Handler
	in      *bufio.Reader
	out     io.Writer
}

func NewServer(entitiesDir string) (*Server, error) {
	h, err := NewHandler(entitiesDir)
	if err != nil {
		return nil, err
	}
	return &Server{
		handler: h,
		in:      bufio.NewReader(os.Stdin),
		out:     os.Stdout,
	}, nil
}

// Run starts the LSP read/write loop.
func (s *Server) Run() {
	for {
		msg, err := s.readMessage()
		if err != nil {
			if err == io.EOF {
				return
			}
			fmt.Fprintf(os.Stderr, "read error: %v\n", err)
			return
		}

		resp := s.handler.Handle(msg)
		if resp != nil {
			if err := s.writeMessage(resp); err != nil {
				fmt.Fprintf(os.Stderr, "write error: %v\n", err)
				return
			}
		}
	}
}

func (s *Server) readMessage() (map[string]json.RawMessage, error) {
	// read headers
	var contentLength int
	for {
		line, err := s.in.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		if strings.HasPrefix(line, "Content-Length: ") {
			n, err := strconv.Atoi(strings.TrimPrefix(line, "Content-Length: "))
			if err != nil {
				return nil, fmt.Errorf("invalid Content-Length: %w", err)
			}
			contentLength = n
		}
	}

	if contentLength == 0 {
		return nil, fmt.Errorf("missing Content-Length")
	}

	body := make([]byte, contentLength)
	if _, err := io.ReadFull(s.in, body); err != nil {
		return nil, err
	}

	var msg map[string]json.RawMessage
	if err := json.Unmarshal(body, &msg); err != nil {
		return nil, err
	}
	return msg, nil
}

func (s *Server) writeMessage(v any) error {
	body, err := json.Marshal(v)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(s.out, "Content-Length: %d\r\n\r\n%s", len(body), body)
	return err
}
