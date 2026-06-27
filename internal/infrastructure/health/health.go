// Package health provides the /api/health endpoint.
package health

import (
	"encoding/json"
	"net/http"
)

// Handler responds with {"status": "ok"}.
//
// @Summary     Health check
// @Tags        system
// @Produce     json
// @Success     200  {object}  map[string]string
// @Router      /health [get]
func Handler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
