// Package mcp exposes a per-workspace MCP (Model Context Protocol) server over
// HTTP Server-Sent Events. Each workspace gets its own SSE endpoint protected
// by the existing workspace-token auth middleware.
package mcp

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/saintedlama/degubase/internal/automation"
	httplib "github.com/saintedlama/degubase/internal/infrastructure/http"
	"github.com/saintedlama/degubase/internal/records"
	"github.com/saintedlama/degubase/internal/schema"
)

// Handler serves the MCP SSE and message endpoints for a single workspace.
type Handler struct {
	Schema schema.Store
	Rows   records.Store
	RowSvc *records.RowService
	Auto   automation.Store

	mu       sync.Mutex
	sessions map[string]*session
}

type session struct {
	ch chan []byte // buffered response channel
}

// NewHandler constructs a Handler ready to be wired into the router.
func NewHandler(schem schema.Store, rows records.Store, rowSvc *records.RowService, auto automation.Store) *Handler {
	return &Handler{
		Schema:   schem,
		Rows:     rows,
		RowSvc:   rowSvc,
		Auto:     auto,
		sessions: make(map[string]*session),
	}
}

// SSE handles GET …/mcp/sse. It opens a Server-Sent Events stream, registers a
// session, and sends the MCP endpoint URL as the first event.
func (h *Handler) SSE(w http.ResponseWriter, r *http.Request) {
	ws := httplib.WorkspaceFromCtx(r)
	if !ws.MCPEnabled {
		http.Error(w, "MCP not enabled for this workspace", http.StatusForbidden)
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	id := newSessionID()
	sess := &session{ch: make(chan []byte, 64)}
	h.mu.Lock()
	h.sessions[id] = sess
	h.mu.Unlock()
	defer func() {
		h.mu.Lock()
		delete(h.sessions, id)
		h.mu.Unlock()
	}()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	endpoint := fmt.Sprintf("/api/workspaces/%s/mcp/message?sessionId=%s", ws.Code, id)
	fmt.Fprintf(w, "event: endpoint\ndata: %s\n\n", endpoint)
	flusher.Flush()

	for {
		select {
		case <-r.Context().Done():
			return
		case msg := <-sess.ch:
			fmt.Fprintf(w, "event: message\ndata: %s\n\n", msg)
			flusher.Flush()
		}
	}
}

// Message handles POST …/mcp/message?sessionId=<id>. It decodes a JSON-RPC 2.0
// request, dispatches it, and sends the response back on the SSE stream.
func (h *Handler) Message(w http.ResponseWriter, r *http.Request) {
	ws := httplib.WorkspaceFromCtx(r)
	if !ws.MCPEnabled {
		http.Error(w, "MCP not enabled for this workspace", http.StatusForbidden)
		return
	}

	id := r.URL.Query().Get("sessionId")
	h.mu.Lock()
	sess, ok := h.sessions[id]
	h.mu.Unlock()
	if !ok {
		http.Error(w, "session not found", http.StatusGone)
		return
	}

	var req jsonRPCRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Notifications have no id and expect no response.
	if req.ID == nil {
		w.WriteHeader(http.StatusAccepted)
		return
	}

	resp := h.dispatch(r.Context(), ws, &req)
	data, err := json.Marshal(resp)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	select {
	case sess.ch <- data:
		w.WriteHeader(http.StatusAccepted)
	default:
		http.Error(w, "session buffer full", http.StatusServiceUnavailable)
	}
}

// ── JSON-RPC 2.0 types ────────────────────────────────────────────────────────

type jsonRPCRequest struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      *json.RawMessage `json:"id,omitempty"`
	Method  string           `json:"method"`
	Params  json.RawMessage  `json:"params,omitempty"`
}

type jsonRPCResponse struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      *json.RawMessage `json:"id,omitempty"`
	Result  any              `json:"result,omitempty"`
	Error   *jsonRPCError    `json:"error,omitempty"`
}

type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// toolResult is the MCP tools/call result envelope.
type toolResult struct {
	Content []contentItem `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

type contentItem struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func textResult(v any) toolResult {
	data, _ := json.MarshalIndent(v, "", "  ")
	return toolResult{Content: []contentItem{{Type: "text", Text: string(data)}}}
}

func errResult(msg string) toolResult {
	return toolResult{
		Content: []contentItem{{Type: "text", Text: msg}},
		IsError: true,
	}
}

func newSessionID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func safeArgs(args json.RawMessage) json.RawMessage {
	if len(args) == 0 {
		return json.RawMessage("{}")
	}
	return args
}
