package lsp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
)

// Server implements standard JSON-RPC 2.0 framing over io.Reader and io.Writer.
type Server struct {
	reader  *bufio.Reader
	writer  io.Writer
	handler *LSPHandler
	mu      sync.Mutex
	running bool
}

// NewServer creates an LSP server attached to reader and writer streams.
func NewServer(r io.Reader, w io.Writer, handler *LSPHandler) *Server {
	return &Server{
		reader:  bufio.NewReader(r),
		writer:  w,
		handler: handler,
		running: true,
	}
}

// Serve runs the main message pump until EOF or exit notification.
func (s *Server) Serve() error {
	for s.running {
		msg, err := s.readMessage()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		s.dispatch(msg)
	}
	return nil
}

func (s *Server) readMessage() (*JSONRPCMessage, error) {
	contentLength := 0

	// 1. Read headers until blank line
	for {
		line, err := s.reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			break
		}

		if strings.HasPrefix(line, "Content-Length:") {
			val := strings.TrimSpace(strings.TrimPrefix(line, "Content-Length:"))
			contentLength, _ = strconv.Atoi(val)
		}
	}

	if contentLength <= 0 {
		return nil, fmt.Errorf("invalid or missing Content-Length")
	}

	// 2. Read exact payload bytes
	buf := make([]byte, contentLength)
	if _, err := io.ReadFull(s.reader, buf); err != nil {
		return nil, err
	}

	var msg JSONRPCMessage
	if err := json.Unmarshal(buf, &msg); err != nil {
		return nil, fmt.Errorf("malformed JSON payload: %w", err)
	}

	return &msg, nil
}

func (s *Server) writeResponse(msg *JSONRPCMessage) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(body))
	if _, err := s.writer.Write([]byte(header)); err != nil {
		return err
	}
	_, err = s.writer.Write(body)
	return err
}

func (s *Server) dispatch(msg *JSONRPCMessage) {
	resp := &JSONRPCMessage{
		JSONRPC: "2.0",
		ID:      msg.ID,
	}

	switch msg.Method {
	case "initialize":
		var params InitializeParams
		_ = json.Unmarshal(msg.Params, &params)
		res, err := s.handler.HandleInitialize(params)
		if err != nil {
			resp.Error = &ResponseError{Code: -32603, Message: err.Error()}
		} else {
			resp.Result = res
		}

	case "initialized":
		return // Notification, no response required

	case "shutdown":
		resp.Result = nil

	case "exit":
		s.running = false
		return

	case "workspace/symbol":
		var params WorkspaceSymbolParams
		_ = json.Unmarshal(msg.Params, &params)
		res, err := s.handler.HandleWorkspaceSymbol(params)
		if err != nil {
			resp.Error = &ResponseError{Code: -32603, Message: err.Error()}
		} else {
			resp.Result = res
		}

	case "textDocument/definition":
		var params TextDocumentPositionParams
		_ = json.Unmarshal(msg.Params, &params)
		res, err := s.handler.HandleDefinition(params)
		if err != nil {
			resp.Error = &ResponseError{Code: -32603, Message: err.Error()}
		} else {
			resp.Result = res
		}

	case "textDocument/references":
		var params TextDocumentPositionParams
		_ = json.Unmarshal(msg.Params, &params)
		res, err := s.handler.HandleReferences(params)
		if err != nil {
			resp.Error = &ResponseError{Code: -32603, Message: err.Error()}
		} else {
			resp.Result = res
		}

	case "codekernel/packContext":
		var params PackContextParams
		_ = json.Unmarshal(msg.Params, &params)
		res, err := s.handler.HandlePackContext(params)
		if err != nil {
			resp.Error = &ResponseError{Code: -32603, Message: err.Error()}
		} else {
			resp.Result = res
		}

	default:
		resp.Error = &ResponseError{Code: -32601, Message: fmt.Sprintf("method not found: %s", msg.Method)}
	}

	if msg.ID != nil {
		_ = s.writeResponse(resp)
	}
}
