package records

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/saintedlama/degubase/internal/infrastructure/events"
	httplib "github.com/saintedlama/degubase/internal/infrastructure/http"
)

type EventHandler struct{ Broker *events.Broker }

// @Summary     Server-sent events stream
// @Tags        events
// @Produce     text/event-stream
// @Param       wsCode     path  string  true  "workspace code"
// @Param       tableCode  path  string  true  "table code"
// @Success     200
// @Failure     500  {object}  http.ErrorResponse
// @Security    BearerAuth
// @Router      /workspaces/{wsCode}/tables/{tableCode}/events [get]
func (h *EventHandler) Stream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		httplib.InternalErr(w, fmt.Errorf("streaming not supported"))
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	t := httplib.TableFromCtx(r)

	ch := h.Broker.Subscribe(t.ID)
	defer h.Broker.Unsubscribe(t.ID, ch)

	cmdID := r.Header.Get("X-Command-Id")

	for {
		select {
		case ev := <-ch:
			// Only suppress echo when the subscriber has a specific command ID to match.
			// An empty cmdID (normal browser EventSource, which can't send custom headers)
			// must not match events from external callers that also lack a command ID.
			if cmdID != "" && ev.CommandID == cmdID {
				continue
			}
			var payload any = ev
			if ev.ID != 0 && ev.Row == nil {
				payload = map[string]any{
					"type":              ev.Type,
					"id":                ev.ID,
					"command_source_id": ev.CommandSourceID,
					"command_id":        ev.CommandID,
				}
			}
			data, _ := json.Marshal(payload)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}
